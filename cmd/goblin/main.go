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

	"github.com/hoyt-harness/goblin/internal/extract"
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
	flag.IntVar(&cfg.FrameWarn, "frame-warn", 50, "Warn if output frame count exceeds N")
	flag.IntVar(&cfg.Threads, "threads", 0, "Thread count hint for ffmpeg and whisper (0 = auto)")
	flag.BoolVar(&cfg.Quiet, "quiet", false, "Suppress progress lines; errors and warnings always print")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit 0")

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
	if _, err := os.Stat(cfg.Output); err == nil && !cfg.Overwrite {
		fmt.Fprintf(os.Stderr,
			"goblin: error: output directory already exists: %s\n  Use -overwrite to replace it\n",
			cfg.Output)
		return 4
	}
	if err := os.MkdirAll(filepath.Join(cfg.Output, "frames"), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "goblin: error: cannot create output directory: %v\n", err)
		return 4
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

	// Stage: extract.
	var scenes []extract.Scene
	if !cfg.NoFrames && pr.HasVideo {
		progress(cfg.Quiet, "goblin: extract  threshold %.2f", cfg.Threshold)
		var err error
		scenes, _, err = extract.Extract(absInput, cfg.Output, cfg.Threshold)
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
