// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"os"
	"testing"
)

// T053: overwrite guard — the output directory contract.

func TestSetupOutputDirFreshDir(t *testing.T) {
	dir := t.TempDir() + "/newdir"
	code, err := setupOutputDir(dir, false)
	if code != 0 || err != nil {
		t.Errorf("fresh dir: got code=%d err=%v; want code=0 err=nil", code, err)
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Errorf("directory not created: %v", statErr)
	}
}

func TestSetupOutputDirExistingNoOverwrite(t *testing.T) {
	dir := t.TempDir() // already exists
	code, err := setupOutputDir(dir, false)
	if code != 4 {
		t.Errorf("existing dir without -overwrite: got code=%d; want 4", code)
	}
	if err == nil {
		t.Error("expected non-nil error for existing dir")
	}
}

func TestSetupOutputDirExistingWithOverwrite(t *testing.T) {
	dir := t.TempDir()
	code, err := setupOutputDir(dir, true)
	if code != 0 || err != nil {
		t.Errorf("existing dir with -overwrite: got code=%d err=%v; want code=0 err=nil", code, err)
	}
}

// T054: missing tool — checkTools returns exit code 2 for any unresolvable binary.
// Using a name guaranteed absent from PATH. If ffprobe or ffmpeg are also missing
// (CI environments without FFmpeg), they trigger exit 2 first — still correct.
func TestCheckToolsMissingTool(t *testing.T) {
	cfg := &Config{
		WhisperCmd: "goblin-nonexistent-tool-xxxxxx",
	}
	code := checkTools(cfg)
	if code != 2 {
		t.Errorf("missing tool: got code=%d; want 2", code)
	}
}
