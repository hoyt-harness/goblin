// SPDX-License-Identifier: GPL-3.0-or-later
//
// consumer_validate reads a goblin MANIFEST directory and verifies it is
// fully self-consistent and consumable without further processing.
//
// Usage: go run test/consumer_validate.go <OUTPUT_DIR>
//
// Exit 0: all checks pass, ordered scene list printed to stdout.
// Exit 1: one or more checks failed; errors printed to stderr.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type manifest struct {
	SchemaVersion  string     `json:"schema_version"`
	GoblinVersion  string     `json:"goblin_version"`
	SourcePath     string     `json:"source_path"`
	DurationS      float64    `json:"duration_s"`
	StagesRun      []string   `json:"stages_run"`
	ProbePath      string     `json:"probe_path"`
	TranscriptPath string     `json:"transcript_path"`
	Scenes         []sceneRef `json:"scenes"`
	Warnings       []string   `json:"warnings"`
}

type sceneRef struct {
	Index              int     `json:"index"`
	StartS             float64 `json:"start_s"`
	EndS               float64 `json:"end_s"`
	Frame              string  `json:"frame"`
	TranscriptSegments []int   `json:"transcript_segments"`
}

type transcriptFile struct {
	SchemaVersion string    `json:"schema_version"`
	Source        string    `json:"source"`
	Segments      []segment `json:"segments"`
}

type segment struct {
	Index  int     `json:"index"`
	StartS float64 `json:"start_s"`
	EndS   float64 `json:"end_s"`
	Text   string  `json:"text"`
}

type sceneResult struct {
	SceneIndex int
	StartS     float64
	EndS       float64
	FramePath  string
	Captions   []string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: consumer_validate <OUTPUT_DIR>\n")
		os.Exit(1)
	}
	outDir := os.Args[1]

	if code := validate(outDir); code != 0 {
		os.Exit(code)
	}
}

func validate(outDir string) int {
	failures := 0
	fail := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
		failures++
	}

	// Load MANIFEST.json.
	manifestPath := filepath.Join(outDir, "MANIFEST.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		fail("cannot read MANIFEST.json: %v", err)
		return 1
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		fail("cannot parse MANIFEST.json: %v", err)
		return 1
	}

	// schema_version must be "1".
	if m.SchemaVersion != "1" {
		fail("schema_version = %q, want \"1\"", m.SchemaVersion)
	}

	// probe.json must exist.
	if _, err := os.Stat(filepath.Join(outDir, m.ProbePath)); err != nil {
		fail("probe_path %q missing: %v", m.ProbePath, err)
	}

	// scenes must be non-empty.
	if len(m.Scenes) == 0 {
		fail("scenes array is empty")
		return 1
	}

	// Load transcript if present.
	var segs []segment
	if m.TranscriptPath != "" {
		tdata, err := os.ReadFile(filepath.Join(outDir, m.TranscriptPath))
		if err != nil {
			fail("transcript_path %q missing: %v", m.TranscriptPath, err)
		} else {
			var tf transcriptFile
			if err := json.Unmarshal(tdata, &tf); err != nil {
				fail("cannot parse transcript.json: %v", err)
			} else {
				segs = tf.Segments
			}
		}
	}

	// Validate each scene and build output.
	results := make([]sceneResult, 0, len(m.Scenes))
	for _, sc := range m.Scenes {
		if sc.Index != len(results) {
			fail("scene index gap: expected %d, got %d", len(results), sc.Index)
		}
		if sc.StartS >= sc.EndS {
			fail("scene %d: start_s (%.3f) >= end_s (%.3f)", sc.Index, sc.StartS, sc.EndS)
		}

		// Frame file must exist if set.
		framePath := sc.Frame
		if framePath != "" {
			full := filepath.Join(outDir, filepath.FromSlash(framePath))
			if info, err := os.Stat(full); err != nil {
				fail("scene %d: frame %q missing: %v", sc.Index, framePath, err)
			} else if info.Size() == 0 {
				fail("scene %d: frame %q is empty", sc.Index, framePath)
			}
		}

		// All transcript segment indices must be in bounds.
		var captions []string
		for _, idx := range sc.TranscriptSegments {
			if idx < 0 || idx >= len(segs) {
				fail("scene %d: transcript_segments references index %d (out of bounds, have %d segments)",
					sc.Index, idx, len(segs))
				continue
			}
			captions = append(captions, segs[idx].Text)
		}

		results = append(results, sceneResult{
			SceneIndex: sc.Index,
			StartS:     sc.StartS,
			EndS:       sc.EndS,
			FramePath:  framePath,
			Captions:   captions,
		})
	}

	if failures > 0 {
		fmt.Fprintf(os.Stderr, "\n%d check(s) failed.\n", failures)
		return 1
	}

	// Print ordered scene list.
	fmt.Printf("goblin consumer validation: PASS\n")
	fmt.Printf("source: %s | duration: %.1fs | scenes: %d | segments: %d\n\n",
		filepath.Base(m.SourcePath), m.DurationS, len(m.Scenes), len(segs))
	for _, r := range results {
		fmt.Printf("scene %d  [%.2f–%.2f]  %s\n", r.SceneIndex, r.StartS, r.EndS, r.FramePath)
		for _, c := range r.Captions {
			fmt.Printf("  › %s\n", c)
		}
	}
	return 0
}
