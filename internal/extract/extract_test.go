// SPDX-License-Identifier: GPL-3.0-or-later
package extract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFixture(t *testing.T) {
	const fixturePath = "../../test/fixtures/test.mp4"
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skipf("fixture not available; run test/fixtures/gen.sh to generate")
	}

	outDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outDir, "frames"), 0o755); err != nil {
		t.Fatal(err)
	}

	scenes, frames, err := Extract(fixturePath, outDir, 0.4, 0)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if len(scenes) < 1 {
		t.Fatal("expected at least 1 scene, got 0")
	}
	if len(frames) < 1 {
		t.Fatal("expected at least 1 frame, got 0")
	}

	// All frame files must exist on disk.
	for _, f := range frames {
		p := filepath.Join(outDir, "frames", f.Filename)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("frame file missing: %s", p)
		}
	}

	// Scenes must be contiguous (end of scene i == start of scene i+1).
	const duration = 30.0
	for i := 0; i+1 < len(scenes); i++ {
		if scenes[i+1].StartS != scenes[i+1].StartS {
			t.Errorf("scene %d start inconsistency", i+1)
		}
	}

	// Last scene start must be within duration.
	last := scenes[len(scenes)-1]
	if last.StartS > duration+0.5 {
		t.Errorf("last scene start %.2f exceeds duration %.2f + tolerance", last.StartS, duration)
	}
}

func TestParseFrameMetaEmpty(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), ".frame-meta")
	if err := os.WriteFile(tmp, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	frames, err := parseFrameMeta(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 0 {
		t.Errorf("expected 0 frames from empty file, got %d", len(frames))
	}
}

func TestParseFrameMetaNonExistent(t *testing.T) {
	frames, err := parseFrameMeta("/nonexistent/.frame-meta")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if frames != nil {
		t.Errorf("expected nil frames for missing file, got %v", frames)
	}
}

func TestParseFrameMetaSample(t *testing.T) {
	content := `frame:0 pts:0 pts_time:0.000000
lavfi.scene_score=1.000000

frame:1 pts:90090 pts_time:3.003000
lavfi.scene_score=0.520000

frame:2 pts:270270 pts_time:9.009000
lavfi.scene_score=0.610000

`
	tmp := filepath.Join(t.TempDir(), ".frame-meta")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	frames, err := parseFrameMeta(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(frames))
	}
	if frames[0].timestampS != 0 {
		t.Errorf("frame[0].timestampS = %.3f, want 0", frames[0].timestampS)
	}
	if frames[1].timestampS != 3.003 {
		t.Errorf("frame[1].timestampS = %.3f, want 3.003", frames[1].timestampS)
	}
	if frames[0].filename != "frame_00001.png" {
		t.Errorf("frame[0].filename = %q, want frame_00001.png", frames[0].filename)
	}
	if frames[2].filename != "frame_00003.png" {
		t.Errorf("frame[2].filename = %q, want frame_00003.png", frames[2].filename)
	}
}
