// SPDX-License-Identifier: GPL-3.0-or-later
package transcribe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseWhisperJSON(t *testing.T) {
	sample := `{
  "params": {"model": "/d/models/whisper-models/ggml-large-v3-turbo.bin"},
  "result": {"language": "en"},
  "transcription": [
    {"offsets": {"from": 0, "to": 3420}, "text": " Welcome to the interview."},
    {"offsets": {"from": 3420, "to": 7810}, "text": " Today we discuss security."},
    {"offsets": {"from": 9000, "to": 9000}, "text": "   "}
  ]
}`
	tmp := filepath.Join(t.TempDir(), ".whisper.json")
	if err := os.WriteFile(tmp, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}

	segs, meta, err := parseWhisperJSON(tmp, "fallback-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Whitespace-only segment should be filtered.
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments (whitespace filtered), got %d", len(segs))
	}
	if segs[0].StartS != 0 {
		t.Errorf("seg[0].StartS = %.3f, want 0", segs[0].StartS)
	}
	if segs[0].EndS != 3.42 {
		t.Errorf("seg[0].EndS = %.3f, want 3.42", segs[0].EndS)
	}
	if segs[0].Text != "Welcome to the interview." {
		t.Errorf("seg[0].Text = %q, want trimmed", segs[0].Text)
	}
	if segs[0].Index != 0 || segs[1].Index != 1 {
		t.Errorf("segments re-indexed incorrectly: %d, %d", segs[0].Index, segs[1].Index)
	}
	if meta.Source != "whisper" {
		t.Errorf("meta.Source = %q, want whisper", meta.Source)
	}
	if meta.Language != "en" {
		t.Errorf("meta.Language = %q, want en", meta.Language)
	}
	if meta.Model != "/d/models/whisper-models/ggml-large-v3-turbo.bin" {
		t.Errorf("meta.Model = %q", meta.Model)
	}
}

func TestParseWhisperJSONFallbackModel(t *testing.T) {
	sample := `{"params": {}, "result": {}, "transcription": []}`
	tmp := filepath.Join(t.TempDir(), ".whisper.json")
	if err := os.WriteFile(tmp, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	_, meta, err := parseWhisperJSON(tmp, "fallback-model")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Model != "fallback-model" {
		t.Errorf("meta.Model = %q, want fallback-model", meta.Model)
	}
}
