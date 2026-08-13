// SPDX-License-Identifier: GPL-3.0-or-later
package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hoyt-harness/goblin/internal/extract"
	"github.com/hoyt-harness/goblin/internal/transcribe"
)

func TestBuildSceneRefsContiguous(t *testing.T) {
	scenes := []extract.Scene{
		{Index: 0, StartS: 0, FramePath: "frames/frame_00001.png"},
		{Index: 1, StartS: 3.0, FramePath: "frames/frame_00002.png"},
		{Index: 2, StartS: 9.0, FramePath: "frames/frame_00003.png"},
	}
	duration := 30.0

	refs := BuildSceneRefs(scenes, nil, duration)

	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}
	if refs[0].EndS != 3.0 {
		t.Errorf("refs[0].EndS = %.2f, want 3.0", refs[0].EndS)
	}
	if refs[1].EndS != 9.0 {
		t.Errorf("refs[1].EndS = %.2f, want 9.0", refs[1].EndS)
	}
	if refs[2].EndS != 30.0 {
		t.Errorf("refs[2].EndS = %.2f, want 30.0", refs[2].EndS)
	}
	for i, r := range refs {
		if r.Index != i {
			t.Errorf("refs[%d].Index = %d", i, r.Index)
		}
	}
}

func TestBuildSceneRefsTranscriptOverlap(t *testing.T) {
	scenes := []extract.Scene{
		{Index: 0, StartS: 0, FramePath: "frames/frame_00001.png"},
		{Index: 1, StartS: 5.0, FramePath: "frames/frame_00002.png"},
	}
	segs := []transcribe.TranscriptSegment{
		{Index: 0, StartS: 0, EndS: 3.0, Text: "in scene 0"},
		{Index: 1, StartS: 3.0, EndS: 6.0, Text: "spans both"},
		{Index: 2, StartS: 7.0, EndS: 10.0, Text: "in scene 1"},
	}
	refs := BuildSceneRefs(scenes, segs, 12.0)

	// Seg 0 overlaps scene 0 [0,5): start 0 < 5 && end 3 > 0 → yes
	// Seg 1 overlaps scene 0 [0,5): start 3 < 5 && end 6 > 0 → yes
	// Seg 2 overlaps scene 0 [0,5): start 7 < 5? no → no
	if !contains(refs[0].TranscriptSegments, 0) || !contains(refs[0].TranscriptSegments, 1) {
		t.Errorf("scene 0 should reference segs 0 and 1, got %v", refs[0].TranscriptSegments)
	}
	if contains(refs[0].TranscriptSegments, 2) {
		t.Errorf("scene 0 should not reference seg 2")
	}

	// Seg 1 overlaps scene 1 [5,12): start 3 < 12 && end 6 > 5 → yes
	// Seg 2 overlaps scene 1 [5,12): start 7 < 12 && end 10 > 5 → yes
	if !contains(refs[1].TranscriptSegments, 1) || !contains(refs[1].TranscriptSegments, 2) {
		t.Errorf("scene 1 should reference segs 1 and 2, got %v", refs[1].TranscriptSegments)
	}
}

func TestBuildSceneRefsEmpty(t *testing.T) {
	refs := BuildSceneRefs(nil, nil, 30.0)
	if len(refs) != 1 {
		t.Fatalf("expected 1 fallback scene ref for empty input, got %d", len(refs))
	}
	if refs[0].EndS != 30.0 {
		t.Errorf("fallback scene EndS = %.2f, want 30.0", refs[0].EndS)
	}
}

func TestWriteManifestSchemaVersion(t *testing.T) {
	outDir := t.TempDir()
	m := &Manifest{
		SchemaVersion: SchemaVersion,
		GoblinVersion: "0.1.0-test",
		GeneratedAt:   "2026-08-10T00:00:00Z",
		SourcePath:    "/test/video.mp4",
		DurationS:     30.0,
		StagesRun:     []string{"probe"},
		ProbePath:     "probe.json",
		Warnings:      []string{},
		Scenes:        []SceneRef{},
	}
	if err := WriteManifest(outDir, m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "MANIFEST.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out["schema_version"] != "1" {
		t.Errorf("schema_version = %v, want \"1\"", out["schema_version"])
	}
}

func TestRelativePathsForwardSlash(t *testing.T) {
	scenes := []extract.Scene{
		{Index: 0, StartS: 0, FramePath: "frames/frame_00001.png"},
	}
	refs := BuildSceneRefs(scenes, nil, 10.0)
	for i, r := range refs {
		for _, c := range r.Frame {
			if c == '\\' {
				t.Errorf("refs[%d].Frame contains backslash: %q", i, r.Frame)
			}
		}
	}
}

func TestBuildSceneRefsSceneScore(t *testing.T) {
	scenes := []extract.Scene{
		{Index: 0, StartS: 0, FramePath: "frames/frame_00001.png", SceneScore: 0.72},
		{Index: 1, StartS: 5.0, FramePath: "frames/frame_00002.png", SceneScore: 0.45},
	}
	refs := BuildSceneRefs(scenes, nil, 10.0)
	if refs[0].SceneScore != 0.72 {
		t.Errorf("refs[0].SceneScore = %.2f, want 0.72", refs[0].SceneScore)
	}
	if refs[1].SceneScore != 0.45 {
		t.Errorf("refs[1].SceneScore = %.2f, want 0.45", refs[1].SceneScore)
	}
}

func contains(slice []int, v int) bool {
	for _, x := range slice {
		if x == v {
			return true
		}
	}
	return false
}
