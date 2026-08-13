// SPDX-License-Identifier: GPL-3.0-or-later
package transcribe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hoyt-harness/goblin/internal/probe"
)

func TestParseSRT(t *testing.T) {
	content := `1
00:00:00,000 --> 00:00:03,420
<i>Welcome to the interview.</i>

2
00:00:03,420 --> 00:00:07,810
Today we discuss <b>security</b>.

3
00:00:09,000 --> 00:00:10,000


`
	tmp := filepath.Join(t.TempDir(), "test.srt")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	segs, err := parseSRT(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Whitespace-only entry should be filtered.
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0].StartS != 0 {
		t.Errorf("seg[0].StartS = %.3f, want 0", segs[0].StartS)
	}
	if segs[0].EndS != 3.42 {
		t.Errorf("seg[0].EndS = %.3f, want 3.42", segs[0].EndS)
	}
	if segs[0].Text != "Welcome to the interview." {
		t.Errorf("seg[0].Text = %q (tags not stripped?)", segs[0].Text)
	}
	if segs[1].Text != "Today we discuss security." {
		t.Errorf("seg[1].Text = %q (bold tag not stripped?)", segs[1].Text)
	}
	if segs[0].Index != 0 || segs[1].Index != 1 {
		t.Errorf("wrong indices: %d, %d", segs[0].Index, segs[1].Index)
	}
}

func TestParseSRTTimestamp(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"00:00:00,000", 0},
		{"00:01:23,456", 83.456},
		{"01:00:00,000", 3600},
		{"00:00:03,420", 3.42},
	}
	for _, c := range cases {
		got, err := parseSRTTimestamp(c.in)
		if err != nil {
			t.Errorf("parseSRTTimestamp(%q) error: %v", c.in, err)
			continue
		}
		if got < c.want-0.001 || got > c.want+0.001 {
			t.Errorf("parseSRTTimestamp(%q) = %.3f, want %.3f", c.in, got, c.want)
		}
	}
}

func TestStripSRTTags(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"<i>hello</i>", "hello"},
		{"<b>bold</b> text", "bold text"},
		{"no tags", "no tags"},
		{"<font color=\"red\">red</font>", "red"},
	}
	for _, c := range cases {
		got := stripSRTTags(c.in)
		if got != c.want {
			t.Errorf("stripSRTTags(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestShouldExtractSubtitlesNoTracks(t *testing.T) {
	pr := &probe.ProbeResult{SubtitleTracks: []probe.SubtitleTrack{}}
	if ShouldExtractSubtitles(pr) {
		t.Error("ShouldExtractSubtitles should return false when no subtitle tracks")
	}
}

func TestShouldExtractSubtitlesNilTracks(t *testing.T) {
	pr := &probe.ProbeResult{SubtitleTracks: nil}
	if ShouldExtractSubtitles(pr) {
		t.Error("ShouldExtractSubtitles should return false when SubtitleTracks is nil")
	}
}
