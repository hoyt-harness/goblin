// SPDX-License-Identifier: GPL-3.0-or-later
package grid

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakePNG creates a small solid-color PNG at path for use as a test frame.
func writeFakePNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// seedFrames creates n sequentially-named PNG files in outDir/frames/.
func seedFrames(t *testing.T, outDir string, n int) {
	t.Helper()
	framesDir := filepath.Join(outDir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= n; i++ {
		writeFakePNG(t, filepath.Join(framesDir, fmt.Sprintf("frame_%05d.png", i)))
	}
}

// ffmpegAvailable reports whether ffmpeg is on PATH.
func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func TestGridNoFrames(t *testing.T) {
	outDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outDir, "frames"), 0o755); err != nil {
		t.Fatal(err)
	}
	paths, err := Grid(outDir, 4, 0)
	if err != nil {
		t.Fatalf("Grid on empty frames dir: %v", err)
	}
	if paths != nil {
		t.Errorf("expected nil paths for empty frames dir, got %v", paths)
	}
}

func TestGridSinglePage(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not on PATH")
	}
	outDir := t.TempDir()
	seedFrames(t, outDir, 5) // 5 frames, cols=4 → 1 page (4×4, partial)

	paths, err := Grid(outDir, 4, 0)
	if err != nil {
		t.Fatalf("Grid: %v", err)
	}
	if len(paths) != 1 {
		t.Errorf("expected 1 grid file for 5 frames with cols=4, got %d: %v", len(paths), paths)
	}
	if !strings.HasPrefix(paths[0], "grids/") {
		t.Errorf("grid path should be relative grids/..., got %q", paths[0])
	}
	if _, err := os.Stat(filepath.Join(outDir, filepath.FromSlash(paths[0]))); err != nil {
		t.Errorf("grid file missing on disk: %v", err)
	}
}

func TestGridMultiPage(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not on PATH")
	}
	const (
		cols     = 4
		pageSize = cols * cols // 16
		nFrames  = 17          // 1 full page + 1 partial page
	)
	outDir := t.TempDir()
	seedFrames(t, outDir, nFrames)

	paths, err := Grid(outDir, cols, 0)
	if err != nil {
		t.Fatalf("Grid: %v", err)
	}
	wantPages := 2
	if len(paths) != wantPages {
		t.Errorf("expected %d grid files for %d frames (cols=%d), got %d: %v",
			wantPages, nFrames, cols, len(paths), paths)
	}
	// Partial last page must exist and be non-empty.
	for _, p := range paths {
		full := filepath.Join(outDir, filepath.FromSlash(p))
		info, err := os.Stat(full)
		if err != nil {
			t.Errorf("grid file %q missing: %v", p, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("grid file %q is empty", p)
		}
	}
}

func TestGridRectangular(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not on PATH")
	}
	const (
		cols    = 6
		rows    = 3
		nFrames = 20 // 1 full 6×3 page (18 frames) + 1 partial page (2 frames)
	)
	outDir := t.TempDir()
	seedFrames(t, outDir, nFrames)

	paths, err := Grid(outDir, cols, rows)
	if err != nil {
		t.Fatalf("Grid(cols=%d, rows=%d): %v", cols, rows, err)
	}
	wantPages := 2
	if len(paths) != wantPages {
		t.Errorf("expected %d grid files for %d frames (cols=%d rows=%d), got %d: %v",
			wantPages, nFrames, cols, rows, len(paths), paths)
	}
	for _, p := range paths {
		full := filepath.Join(outDir, filepath.FromSlash(p))
		info, err := os.Stat(full)
		if err != nil {
			t.Errorf("grid file %q missing: %v", p, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("grid file %q is empty", p)
		}
	}
}

func TestGridSquareDefault(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not on PATH")
	}
	// rows=0 should produce the same page count as rows=cols (square).
	const (
		cols    = 4
		nFrames = 17 // same as TestGridMultiPage
	)
	outDir := t.TempDir()
	seedFrames(t, outDir, nFrames)

	pathsDefault, err := Grid(outDir, cols, 0)
	if err != nil {
		t.Fatalf("Grid(rows=0): %v", err)
	}

	outDir2 := t.TempDir()
	seedFrames(t, outDir2, nFrames)
	pathsExplicit, err := Grid(outDir2, cols, cols)
	if err != nil {
		t.Fatalf("Grid(rows=cols): %v", err)
	}

	if len(pathsDefault) != len(pathsExplicit) {
		t.Errorf("rows=0 produced %d pages, rows=cols produced %d pages — should be equal",
			len(pathsDefault), len(pathsExplicit))
	}
}

func TestSortedFrameNames(t *testing.T) {
	outDir := t.TempDir()
	seedFrames(t, outDir, 3)

	names, err := SortedFrameNames(outDir)
	if err != nil {
		t.Fatalf("SortedFrameNames: %v", err)
	}
	want := []string{"frame_00001.png", "frame_00002.png", "frame_00003.png"}
	if len(names) != len(want) {
		t.Fatalf("expected %d names, got %d: %v", len(want), len(names), names)
	}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("names[%d] = %q, want %q", i, names[i], w)
		}
	}
}
