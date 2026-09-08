// SPDX-License-Identifier: GPL-3.0-or-later
package download

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildFormatString_Default(t *testing.T) {
	s := buildFormatString(720)
	if !strings.Contains(s, "height<=720") {
		t.Errorf("expected height<=720 in format string, got: %s", s)
	}
}

func TestBuildFormatString_WithCap(t *testing.T) {
	for _, h := range []int{480, 720, 1080, 1440} {
		s := buildFormatString(h)
		want := fmt.Sprintf("height<=%d", h)
		if !strings.Contains(s, want) {
			t.Errorf("maxHeight=%d: expected %q in format string, got: %s", h, want, s)
		}
	}
}

func TestBuildFormatString_NoCap(t *testing.T) {
	for _, h := range []int{0, -1} {
		s := buildFormatString(h)
		if strings.Contains(s, "height<=") {
			t.Errorf("maxHeight=%d: unexpected height filter in format string: %s", h, s)
		}
	}
}

func TestDownload_BinaryNotFound(t *testing.T) {
	_, err := Download("https://example.com/fake", t.TempDir(), "/nonexistent/yt-dlp-fake", 720)
	if err == nil {
		t.Error("expected error for missing binary, got nil")
	}
}
