// SPDX-License-Identifier: GPL-3.0-or-later
package transcribe

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hoyt-harness/goblin/internal/probe"
)

// ShouldExtractSubtitles returns true when mkvextract is available and the input has subtitle tracks.
func ShouldExtractSubtitles(pr *probe.ProbeResult) bool {
	if len(pr.SubtitleTracks) == 0 {
		return false
	}
	_, err := exec.LookPath("mkvextract")
	return err == nil
}

// ExtractSubtitles uses mkvextract to pull the first subtitle track and returns normalized segments.
func ExtractSubtitles(input, outDir string, pr *probe.ProbeResult) ([]TranscriptSegment, TranscriptMeta, error) {
	if len(pr.SubtitleTracks) == 0 {
		return nil, TranscriptMeta{}, fmt.Errorf("no subtitle tracks")
	}
	track := pr.SubtitleTracks[0]
	srtPath := filepath.Join(outDir, ".subtitles.srt")
	defer os.Remove(srtPath)

	trackSpec := fmt.Sprintf("%d:%s", track.Index, filepath.ToSlash(srtPath))
	cmd := exec.Command("mkvextract", "tracks", filepath.ToSlash(input), trackSpec)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, TranscriptMeta{}, fmt.Errorf("mkvextract: %w\n%s", err, out)
	}

	segs, err := parseSRT(srtPath)
	if err != nil {
		return nil, TranscriptMeta{}, fmt.Errorf("parse SRT: %w", err)
	}

	meta := TranscriptMeta{
		Source:   "subtitle",
		Language: track.Language,
		Model:    "",
	}
	return segs, meta, nil
}

// parseSRT reads an SRT file and returns normalized TranscriptSegments.
func parseSRT(path string) ([]TranscriptSegment, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var segs []TranscriptSegment
	var idx int
	var startS, endS float64
	var textLines []string
	state := "seq"

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch state {
		case "seq":
			if line == "" {
				continue
			}
			state = "time"
		case "time":
			if line == "" {
				state = "seq"
				continue
			}
			var err error
			startS, endS, err = parseSRTTime(line)
			if err != nil {
				return nil, fmt.Errorf("SRT time parse: %w", err)
			}
			textLines = nil
			state = "text"
		case "text":
			if line == "" {
				text := strings.TrimSpace(strings.Join(textLines, " "))
				if text != "" {
					segs = append(segs, TranscriptSegment{
						Index:  idx,
						StartS: startS,
						EndS:   endS,
						Text:   text,
					})
					idx++
				}
				state = "seq"
			} else {
				textLines = append(textLines, stripSRTTags(line))
			}
		}
	}
	// Flush last block if file doesn't end with blank line.
	if state == "text" && len(textLines) > 0 {
		text := strings.TrimSpace(strings.Join(textLines, " "))
		if text != "" {
			segs = append(segs, TranscriptSegment{
				Index:  idx,
				StartS: startS,
				EndS:   endS,
				Text:   text,
			})
		}
	}

	return segs, scanner.Err()
}

// parseSRTTime parses "00:01:23,456 --> 00:01:26,789" and returns (start, end) in seconds.
func parseSRTTime(line string) (float64, float64, error) {
	parts := strings.SplitN(line, " --> ", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected 'start --> end', got %q", line)
	}
	start, err := parseSRTTimestamp(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	end, err := parseSRTTimestamp(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func parseSRTTimestamp(s string) (float64, error) {
	// Format: HH:MM:SS,mmm
	s = strings.ReplaceAll(s, ",", ".")
	var h, m int
	var sec float64
	if _, err := fmt.Sscanf(s, "%d:%d:%f", &h, &m, &sec); err != nil {
		return 0, fmt.Errorf("parse timestamp %q: %w", s, err)
	}
	return float64(h)*3600 + float64(m)*60 + sec, nil
}

func stripSRTTags(s string) string {
	var out strings.Builder
	inTag := false
	for _, c := range s {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			out.WriteRune(c)
		}
	}
	return out.String()
}
