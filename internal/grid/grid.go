// SPDX-License-Identifier: GPL-3.0-or-later
package grid

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Grid creates contact sheet grid images from frame PNGs in outDir/frames/.
// cols is the number of columns per page; rows per page equals cols (square pages),
// so each page holds cols×cols frames.
// Returns forward-slash relative paths to the generated grid files,
// sorted by filename. Returns nil if no frames are found.
func Grid(outDir string, cols int) ([]string, error) {
	framesDir := filepath.Join(outDir, "frames")
	entries, err := os.ReadDir(framesDir)
	if err != nil {
		return nil, fmt.Errorf("read frames dir: %w", err)
	}

	var frameNames []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			frameNames = append(frameNames, e.Name())
		}
	}
	if len(frameNames) == 0 {
		return nil, nil
	}
	sort.Strings(frameNames)

	if cols < 1 {
		cols = 1
	}
	rows := cols // square pages

	gridsDir := filepath.Join(outDir, "grids")
	if err := os.MkdirAll(gridsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create grids dir: %w", err)
	}

	tileSpec := fmt.Sprintf("tile=%dx%d:padding=2", cols, rows)

	// Use sequential %05d input; frames/ are numbered from 1 with no gaps.
	// Run from outDir so relative paths avoid Windows drive-letter colon
	// conflicts inside filtergraph option strings.
	cmd := exec.Command("ffmpeg",
		"-framerate", "1",
		"-start_number", "1",
		"-i", "frames/frame_%05d.png",
		"-vf", tileSpec,
		"-loglevel", "error",
		"grids/grid_%03d.png",
	)
	cmd.Dir = outDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg grid: %w\n%s", err, out)
	}

	gridsEntries, err := os.ReadDir(gridsDir)
	if err != nil {
		return nil, fmt.Errorf("read grids dir: %w", err)
	}

	var paths []string
	for _, e := range gridsEntries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			paths = append(paths, "grids/"+e.Name())
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// SortedFrameNames returns the sorted list of PNG filenames from outDir/frames/.
// Callers use the position index to map scenes to their grid page.
func SortedFrameNames(outDir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(outDir, "frames"))
	if err != nil {
		return nil, fmt.Errorf("read frames dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
