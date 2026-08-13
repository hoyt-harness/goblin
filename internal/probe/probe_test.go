// SPDX-License-Identifier: GPL-3.0-or-later
package probe

import (
	"testing"
)

func TestProbeNonExistent(t *testing.T) {
	_, err := Probe("/nonexistent/path/video.mp4")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestParseRational(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"30000/1001", 29.97002997},
		{"25/1", 25.0},
		{"0/0", 0.0},
		{"bad", 0.0},
	}
	for _, c := range cases {
		got := parseRational(c.in)
		if c.want != 0 && got == 0 {
			t.Errorf("parseRational(%q) = 0, want ~%.2f", c.in, c.want)
		}
		if c.want == 0 && got != 0 {
			t.Errorf("parseRational(%q) = %.4f, want 0", c.in, got)
		}
	}
}

// TestProbeFixture runs against the generated test fixture if available.
// If the fixture doesn't exist, the test is skipped.
func TestProbeFixture(t *testing.T) {
	const fixturePath = "../../test/fixtures/test.mp4"
	r, err := Probe(fixturePath)
	if err != nil {
		t.Skipf("fixture not available (%v); run test/fixtures/gen.sh to generate", err)
	}

	if r.VideoCodec == "" {
		t.Error("VideoCodec is empty")
	}
	if r.DurationS < 29 || r.DurationS > 31 {
		t.Errorf("DurationS = %.2f, want ~30", r.DurationS)
	}
	if !r.HasVideo {
		t.Error("HasVideo = false, want true")
	}
	if !r.HasAudio {
		t.Error("HasAudio = false, want true")
	}
	if r.SubtitleTracks == nil {
		t.Error("SubtitleTracks is nil, want empty slice")
	}
}
