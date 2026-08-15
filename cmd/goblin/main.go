// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	goblinpkg "github.com/hoyt-harness/goblin"
	"github.com/hoyt-harness/goblin/internal/extract"
	"github.com/hoyt-harness/goblin/internal/grid"
	"github.com/hoyt-harness/goblin/internal/manifest"
	"github.com/hoyt-harness/goblin/internal/probe"
	"github.com/hoyt-harness/goblin/internal/transcribe"
)

// version is set at build time via -ldflags "-X main.version=x.y.z".
var version = "0.1.0-dev"

type Config struct {
	Output        string
	Threshold     float64
	ProbeOnly     bool
	NoFrames      bool
	NoTranscript  bool
	PreferWhisper bool
	Overwrite     bool
	WhisperCmd    string
	Model         string
	Grid          bool
	GridCols      int
	GridRows      int
	FrameMaxDim   int
	FrameWarn     int
	Threads       int
	Quiet         bool
}

func main() {
	os.Exit(run())
}

func run() int {
	var cfg Config
	var showVersion bool
	var showGuide bool

	flag.StringVar(&cfg.Output, "output", "", "Output directory path (default: <FILE_basename>_goblin/ next to FILE)")
	flag.Float64Var(&cfg.Threshold, "threshold", 0.4, "Scene detection sensitivity (0.0–1.0)")
	flag.BoolVar(&cfg.ProbeOnly, "probe-only", false, "Run probe stage only; skip extract and transcribe")
	flag.BoolVar(&cfg.NoFrames, "no-frames", false, "Skip frame extraction")
	flag.BoolVar(&cfg.NoTranscript, "no-transcript", false, "Skip transcription")
	flag.BoolVar(&cfg.PreferWhisper, "prefer-whisper", false, "Run whisper even when embedded subtitles exist")
	flag.BoolVar(&cfg.Overwrite, "overwrite", false, "Overwrite existing output directory")
	flag.StringVar(&cfg.WhisperCmd, "whisper-cmd", "", "Whisper binary name or full path (overrides GOBLIN_WHISPER_CMD)")
	flag.StringVar(&cfg.Model, "model", "", "Whisper model file path (overrides GOBLIN_WHISPER_MODEL)")
	flag.BoolVar(&cfg.Grid, "grid", false, "Produce contact sheet grid images")
	flag.IntVar(&cfg.GridCols, "grid-cols", 4, "Columns per grid sheet")
	flag.IntVar(&cfg.GridRows, "grid-rows", 0, "Rows per grid sheet (0 = same as cols, square pages)")
	flag.IntVar(&cfg.FrameMaxDim, "frame-max-dim", 0, "Cap longest frame edge in pixels; 0 = no limit; never upscales")
	flag.IntVar(&cfg.FrameWarn, "frame-warn", 50, "Warn if output frame count exceeds N")
	flag.IntVar(&cfg.Threads, "threads", 0, "Thread count hint for ffmpeg and whisper (0 = auto)")
	flag.BoolVar(&cfg.Quiet, "quiet", false, "Suppress progress lines; errors and warnings always print")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit 0")
	flag.BoolVar(&showGuide, "guide", false, "Print the full usage guide (USING.md) and exit 0")

	flag.Usage = func() {
		fmt.Printf("goblin %s\n\nUsage: goblin [flags] FILE\n\nFlags:\n", version)
		flag.PrintDefaults()
		fmt.Println("\nPrerequisites: ffprobe, ffmpeg, whisper-cli (or set GOBLIN_WHISPER_CMD)")
	}

	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return 0
	}
	if showGuide {
		fmt.Print(goblinpkg.Guide)
		return 0
	}

	// Validate new numeric flags.
	if cfg.FrameMaxDim < 0 {
		fmt.Fprintf(os.Stderr, "goblin: error: -frame-max-dim must be ≥ 0, got %d\n", cfg.FrameMaxDim)
		return 1
	}
	if cfg.GridRows < 0 {
		fmt.Fprintf(os.Stderr, "goblin: error: -grid-rows must be ≥ 0, got %d\n", cfg.GridRows)
		return 1
	}

	// Env var resolution — flag takes precedence over env.
	if cfg.WhisperCmd == "" {
		if v := os.Getenv("GOBLIN_WHISPER_CMD"); v != "" {
			cfg.WhisperCmd = v
		} else {
			cfg.WhisperCmd = "whisper-cli"
		}
	}
	if cfg.Model == "" {
		cfg.Model = os.Getenv("GOBLIN_WHISPER_MODEL")
	}

	// Positional argument validation.
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "goblin: error: FILE argument required\n")
		flag.Usage()
		return 1
	}
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "goblin: error: exactly one FILE argument required, got %d\n", len(args))
		return 1
	}
	inputFile := args[0]

	if _, err := os.Stat(inputFile); err != nil {
		fmt.Fprintf(os.Stderr, "goblin: error: cannot read %s: %v\n", inputFile, err)
		return 3
	}

	// Tool presence checks.
	if code := checkTools(&cfg); code != 0 {
		return code
	}

	// Output directory setup.
	if cfg.Output == "" {
		base := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
		cfg.Output = filepath.Join(filepath.Dir(inputFile), base+"_goblin")
	}
	if code, setupErr := setupOutputDir(cfg.Output, cfg.Overwrite); code != 0 {
		fmt.Fprintf(os.Stderr, "goblin: error: %v\n", setupErr)
		return code
	}

	// Pipeline.
	absInput, _ := filepath.Abs(inputFile)

	m := &manifest.Manifest{
		SchemaVersion: manifest.SchemaVersion,
		GoblinVersion: version,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		SourcePath:    absInput,
		ProbePath:     "probe.json",
		Warnings:      []string{},
		StagesRun:     []string{},
		Scenes:        []manifest.SceneRef{},
		GridMode:      cfg.Grid,
		GridCols:      0,
	}
	if cfg.Grid {
		m.GridCols = cfg.GridCols
	}

	// Stage: probe.
	progress(cfg.Quiet, "goblin: probe  %s", filepath.Base(inputFile))
	pr, err := probe.Probe(absInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goblin: error: probe failed: %v\n", err)
		return 2
	}
	m.DurationS = pr.DurationS
	m.StagesRun = append(m.StagesRun, "probe")

	if err := manifest.WriteProbe(cfg.Output, pr); err != nil {
		fmt.Fprintf(os.Stderr, "goblin: error: %v\n", err)
		return 4
	}

	if cfg.ProbeOnly {
		if err := manifest.WriteManifest(cfg.Output, m); err != nil {
			fmt.Fprintf(os.Stderr, "goblin: error: %v\n", err)
			return 4
		}
		progress(cfg.Quiet, "goblin: done  %s (probe only)", cfg.Output)
		return 0
	}

	// Stage: extract (and grid, if enabled).
	var scenes []extract.Scene
	var gridPaths []string
	if !cfg.NoFrames && pr.HasVideo {
		progress(cfg.Quiet, "goblin: extract  threshold %.2f", cfg.Threshold)
		var err error
		scenes, _, err = extract.Extract(absInput, cfg.Output, cfg.Threshold, cfg.Threads, cfg.FrameMaxDim)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goblin: error: extract failed: %v\n", err)
			return 2
		}
		progress(cfg.Quiet, "goblin: extract  %d scenes detected", len(scenes))
		if len(scenes) > cfg.FrameWarn {
			msg := fmt.Sprintf("%d frames exceed warn threshold %d; consider -grid", len(scenes), cfg.FrameWarn)
			fmt.Fprintf(os.Stderr, "goblin: warning: %s\n", msg)
			m.Warnings = append(m.Warnings, msg)
		}
		m.StagesRun = append(m.StagesRun, "extract")

		// Stage: grid (runs immediately after extract while frames are fresh).
		if cfg.Grid {
			progress(cfg.Quiet, "goblin: grid  %dx%d per page", cfg.GridCols, cfg.GridCols)
			var gridErr error
			gridPaths, gridErr = grid.Grid(cfg.Output, cfg.GridCols, cfg.GridRows)
			if gridErr != nil {
				fmt.Fprintf(os.Stderr, "goblin: error: grid failed: %v\n", gridErr)
				return 2
			}
			m.StagesRun = append(m.StagesRun, "grid")
		}
	} else if !pr.HasVideo && !cfg.NoFrames {
		msg := "no video stream, skipping frame extraction"
		fmt.Fprintf(os.Stderr, "goblin: warning: %s\n", msg)
		m.Warnings = append(m.Warnings, msg)
	}

	// Stage: transcribe.
	var segments []transcribe.TranscriptSegment
	needTranscribe := !cfg.NoTranscript
	if needTranscribe && pr.HasAudio {
		progress(cfg.Quiet, "goblin: transcribe  %s", filepath.Base(cfg.Model))
		var transcriptMeta transcribe.TranscriptMeta
		segments, transcriptMeta, err = transcribe.Transcribe(
			absInput, cfg.Output, cfg.WhisperCmd, cfg.Model,
			pr, cfg.PreferWhisper, cfg.Threads)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goblin: error: transcribe failed: %v\n", err)
			return 2
		}
		progress(cfg.Quiet, "goblin: transcribe  %d segments", len(segments))
		m.StagesRun = append(m.StagesRun, "transcribe")
		m.TranscriptPath = "transcript.json"
		if err := manifest.WriteTranscript(cfg.Output, segments, transcriptMeta); err != nil {
			fmt.Fprintf(os.Stderr, "goblin: error: %v\n", err)
			return 4
		}
	} else if needTranscribe && !pr.HasAudio {
		msg := "no audio stream, skipping transcription"
		fmt.Fprintf(os.Stderr, "goblin: warning: %s\n", msg)
		m.Warnings = append(m.Warnings, msg)
	}

	// Build scene refs with end_s values and transcript cross-links.
	m.Scenes = manifest.BuildSceneRefs(scenes, segments, pr.DurationS)

	// Assign each scene to its grid page using frame filename position in sorted order,
	// matching the order ffmpeg consumed frames during grid generation.
	if cfg.Grid && len(gridPaths) > 0 {
		frameNames, _ := grid.SortedFrameNames(cfg.Output)
		framePosMap := make(map[string]int, len(frameNames))
		for i, name := range frameNames {
			framePosMap[name] = i
		}
		effectiveGridRows := cfg.GridRows
		if effectiveGridRows == 0 {
			effectiveGridRows = cfg.GridCols
		}
		pageSize := cfg.GridCols * effectiveGridRows
		for i := range m.Scenes {
			base := filepath.Base(filepath.FromSlash(m.Scenes[i].Frame))
			if pos, ok := framePosMap[base]; ok {
				page := pos / pageSize
				if page < len(gridPaths) {
					m.Scenes[i].Grid = gridPaths[page]
				}
			}
		}
	}

	if cfg.FrameMaxDim > 0 {
		m.FrameMaxDim = cfg.FrameMaxDim
	}
	if cfg.Grid && cfg.GridRows > 0 {
		m.GridRows = cfg.GridRows
	}

	if err := manifest.WriteManifest(cfg.Output, m); err != nil {
		fmt.Fprintf(os.Stderr, "goblin: error: %v\n", err)
		return 4
	}

	// Post-write verification.
	if warns := manifest.VerifyFramePaths(cfg.Output, m); len(warns) > 0 {
		for _, w := range warns {
			fmt.Fprintf(os.Stderr, "goblin: warning: %s\n", w)
			m.Warnings = append(m.Warnings, w)
		}
	}

	progress(cfg.Quiet, "goblin: done  %s (%d scenes, %d segments)", cfg.Output, len(scenes), len(segments))
	return 0
}

func checkTools(cfg *Config) int {
	tools := []struct {
		name string
		hint string
		skip bool
	}{
		{"ffprobe", "Install the FFmpeg suite: https://ffmpeg.org/download.html", false},
		{"ffmpeg", "Install the FFmpeg suite: https://ffmpeg.org/download.html", cfg.ProbeOnly},
		{cfg.WhisperCmd, "Install whisper.cpp: https://github.com/ggerganov/whisper.cpp", cfg.ProbeOnly || cfg.NoTranscript},
	}
	for _, t := range tools {
		if t.skip {
			continue
		}
		if _, err := exec.LookPath(t.name); err != nil {
			fmt.Fprintf(os.Stderr, "goblin: error: %s not found on PATH\n  %s\n", t.name, t.hint)
			return 2
		}
	}
	return 0
}

func progress(quiet bool, format string, args ...any) {
	if !quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// setupOutputDir validates and creates the output directory.
// When overwrite is true and the directory exists, it is removed before recreation
// so the run starts with a clean slate — no stale frames or grids from prior runs.
// Returns (exit code, error) — code 0 on success, 4 on failure.
func setupOutputDir(output string, overwrite bool) (int, error) {
	if _, err := os.Stat(output); err == nil {
		if !overwrite {
			return 4, fmt.Errorf("output directory already exists: %s\n  Use -overwrite to replace it", output)
		}
		if err := os.RemoveAll(output); err != nil {
			return 4, fmt.Errorf("cannot remove existing output directory: %v", err)
		}
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return 4, fmt.Errorf("cannot create output directory: %v", err)
	}
	return 0, nil
}
