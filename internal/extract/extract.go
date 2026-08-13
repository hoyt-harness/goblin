// SPDX-License-Identifier: GPL-3.0-or-later
package extract

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Scene struct {
	Index      int
	StartS     float64
	FramePath  string
	SceneScore float64
}

type Frame struct {
	Filename   string
	TimestampS float64
	SceneIndex int
}

// Extract runs ffmpeg scene detection on input, writes frame PNGs to outDir/frames/,
// and returns the ordered scene list and per-frame metadata.
// threads is passed as -threads to ffmpeg; 0 means ffmpeg auto-selects.
func Extract(input, outDir string, threshold float64, threads int) ([]Scene, []Frame, error) {
	// Resolve input to absolute so it remains valid when cmd.Dir is set.
	absInput, err := filepath.Abs(input)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve input path: %w", err)
	}
	input = absInput

	if err := os.MkdirAll(filepath.Join(outDir, "frames"), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create frames dir: %w", err)
	}

	// Run ffmpeg with outDir as the working directory so we can use relative paths
	// inside the filtergraph, avoiding Windows drive-letter colons that conflict
	// with ffmpeg's filtergraph option separator.
	metaRel := ".frame-meta"
	metaPath := filepath.Join(outDir, metaRel)

	filter := fmt.Sprintf(`select=gt(scene\,%s),metadata=print:file=%s`, formatThreshold(threshold), metaRel)

	var args []string
	if threads > 0 {
		args = append(args, "-threads", strconv.Itoa(threads))
	}
	args = append(args,
		"-i", filepath.ToSlash(input),
		"-vf", filter,
		"-fps_mode", "vfr",
		"-q:v", "2",
		"-loglevel", "error",
		"frames/frame_%05d.png",
	)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Dir = outDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("ffmpeg extract: %w\n%s", err, out)
	}

	frames, err := parseFrameMeta(metaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("parse frame-meta: %w", err)
	}
	_ = os.Remove(metaPath)

	if len(frames) == 0 {
		// No scene changes detected: extract first frame as single-scene representative.
		var fcArgs []string
		if threads > 0 {
			fcArgs = append(fcArgs, "-threads", strconv.Itoa(threads))
		}
		fcArgs = append(fcArgs,
			"-i", filepath.ToSlash(input),
			"-vframes", "1",
			"-q:v", "2",
			"-loglevel", "error",
			"frames/frame_00001.png",
		)
		fc := exec.Command("ffmpeg", fcArgs...)
		fc.Dir = outDir
		if out, err := fc.CombinedOutput(); err != nil {
			return nil, nil, fmt.Errorf("ffmpeg first-frame fallback: %w\n%s", err, out)
		}
		frames = []rawFrame{{filename: "frame_00001.png", timestampS: 0}}
	}

	scenes := make([]Scene, len(frames))
	allFrames := make([]Frame, len(frames))
	for i, f := range frames {
		relPath := "frames/" + f.filename
		scenes[i] = Scene{
			Index:      i,
			StartS:     f.timestampS,
			FramePath:  relPath,
			SceneScore: f.sceneScore,
		}
		allFrames[i] = Frame{
			Filename:   f.filename,
			TimestampS: f.timestampS,
			SceneIndex: i,
		}
	}

	return scenes, allFrames, nil
}

type rawFrame struct {
	filename   string
	timestampS float64
	sceneScore float64
}

// parseFrameMeta reads the key=value blocks written by ffmpeg's metadata=print filter.
// Each block starts with "frame:N  pts:P pts_time:T" and may include lavfi.scene_score=V.
func parseFrameMeta(path string) ([]rawFrame, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var frames []rawFrame
	frameCount := 1 // ffmpeg output pattern frame_%05d.png starts at 1
	var current rawFrame
	inBlock := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "frame:") {
			if inBlock {
				current.filename = fmt.Sprintf("frame_%05d.png", frameCount)
				frames = append(frames, current)
				frameCount++
			}
			current = rawFrame{}
			inBlock = true
			// Parse pts_time from "frame:N  pts:P pts_time:T"
			for _, field := range strings.Fields(line) {
				if strings.HasPrefix(field, "pts_time:") {
					val := strings.TrimPrefix(field, "pts_time:")
					current.timestampS, _ = strconv.ParseFloat(val, 64)
				}
			}
			continue
		}

		if line == "" {
			if inBlock {
				current.filename = fmt.Sprintf("frame_%05d.png", frameCount)
				frames = append(frames, current)
				frameCount++
				current = rawFrame{}
				inBlock = false
			}
			continue
		}

		if inBlock && strings.HasPrefix(line, "lavfi.scene_score=") {
			val := strings.TrimPrefix(line, "lavfi.scene_score=")
			current.sceneScore, _ = strconv.ParseFloat(val, 64)
		}
	}

	// Flush last block if file didn't end with blank line.
	if inBlock {
		current.filename = fmt.Sprintf("frame_%05d.png", frameCount)
		frames = append(frames, current)
	}

	return frames, scanner.Err()
}

func formatThreshold(t float64) string {
	return strconv.FormatFloat(t, 'f', 2, 64)
}
