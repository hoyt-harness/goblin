// SPDX-License-Identifier: GPL-3.0-or-later
package download

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// buildFormatString returns the yt-dlp -f format string for the given height cap.
// maxHeight=0 means no cap (best available).
func buildFormatString(maxHeight int) string {
	if maxHeight <= 0 {
		return "bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio/best"
	}
	h := fmt.Sprintf("%d", maxHeight)
	return "bestvideo[height<=" + h + "][ext=mp4]+bestaudio[ext=m4a]" +
		"/bestvideo[height<=" + h + "]+bestaudio" +
		"/best[height<=" + h + "]" +
		"/best"
}

// Download fetches url using yt-dlp and writes the video into outDir.
// ytdlpPath is the yt-dlp binary path; empty string uses PATH lookup ("yt-dlp").
// maxHeight caps the video height in pixels; 0 = no cap; default 720 is applied by the caller.
// Returns the absolute path of the downloaded file on success.
func Download(url, outDir, ytdlpPath string, maxHeight int) (string, error) {
	bin := "yt-dlp"
	if ytdlpPath != "" {
		bin = ytdlpPath
	}

	outputTemplate := filepath.Join(outDir, "%(title)s.%(ext)s")

	args := []string{
		"-f", buildFormatString(maxHeight),
		"-o", outputTemplate,
		"--no-playlist",
		"--print", "after_move:filepath",
		url,
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("yt-dlp: %s", msg)
	}

	// --print after_move:filepath writes the final path to stdout.
	outPath := strings.TrimSpace(stdout.String())
	if outPath == "" {
		return "", fmt.Errorf("yt-dlp produced no output file path")
	}

	abs, err := filepath.Abs(outPath)
	if err != nil {
		return "", fmt.Errorf("resolving download path: %w", err)
	}
	return abs, nil
}
