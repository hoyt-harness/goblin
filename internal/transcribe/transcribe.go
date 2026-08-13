// SPDX-License-Identifier: GPL-3.0-or-later
package transcribe

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hoyt-harness/goblin/internal/probe"
)

type TranscriptSegment struct {
	Index  int     `json:"index"`
	StartS float64 `json:"start_s"`
	EndS   float64 `json:"end_s"`
	Text   string  `json:"text"`
}

type TranscriptMeta struct {
	Source   string
	Model    string
	Language string
}

// Transcribe extracts audio from input and runs whisper, returning normalized segments.
// If the input has embedded subtitle tracks and prefer-whisper is false, it delegates to ExtractSubtitles.
func Transcribe(
	input, outDir, whisperCmd, model string,
	pr *probe.ProbeResult,
	preferWhisper bool,
	threads int,
) ([]TranscriptSegment, TranscriptMeta, error) {
	if !preferWhisper && ShouldExtractSubtitles(pr) {
		return ExtractSubtitles(input, outDir, pr)
	}
	return runWhisper(input, outDir, whisperCmd, model, threads)
}

func runWhisper(input, outDir, whisperCmd, model string, threads int) ([]TranscriptSegment, TranscriptMeta, error) {
	audioPath := filepath.Join(outDir, ".audio.wav")
	audioFwd := filepath.ToSlash(audioPath)
	inputFwd := filepath.ToSlash(input)

	// Extract audio as 16kHz mono PCM for whisper.
	args := []string{
		"-i", inputFwd,
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		"-loglevel", "error",
		"-y",
		audioFwd,
	}
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		return nil, TranscriptMeta{}, fmt.Errorf("audio extraction: %w\n%s", err, out)
	}
	defer os.Remove(audioPath)

	// Whisper writes to outDir/.whisper.json (appends .json to the -of path).
	whisperBase := filepath.ToSlash(filepath.Join(outDir, ".whisper"))
	whisperArgs := []string{
		"-m", filepath.ToSlash(model),
		"-f", audioFwd,
		"-oj",
		"-ojf",
		"-of", whisperBase,
		"--no-prints",
	}
	if threads > 0 {
		whisperArgs = append(whisperArgs, "-t", fmt.Sprintf("%d", threads))
	}
	whisperOut, err := exec.Command(whisperCmd, whisperArgs...).CombinedOutput()
	if err != nil {
		return nil, TranscriptMeta{}, fmt.Errorf("whisper: %w\n%s", err, whisperOut)
	}

	whisperJSON := filepath.Join(outDir, ".whisper.json")
	defer os.Remove(whisperJSON)

	segs, meta, err := parseWhisperJSON(whisperJSON, model)
	if err != nil {
		return nil, TranscriptMeta{}, fmt.Errorf("parse whisper output: %w", err)
	}
	return segs, meta, nil
}

// whisperOutput is the structure produced by whisper.cpp with -oj.
type whisperOutput struct {
	Params struct {
		Model string `json:"model"`
	} `json:"params"`
	Result struct {
		Language string `json:"language"`
	} `json:"result"`
	Transcription []struct {
		Offsets struct {
			From int `json:"from"`
			To   int `json:"to"`
		} `json:"offsets"`
		Text string `json:"text"`
	} `json:"transcription"`
}

func parseWhisperJSON(path, modelFallback string) ([]TranscriptSegment, TranscriptMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, TranscriptMeta{}, err
	}

	var raw whisperOutput
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, TranscriptMeta{}, fmt.Errorf("json unmarshal: %w", err)
	}

	meta := TranscriptMeta{
		Source:   "whisper",
		Language: raw.Result.Language,
		Model:    raw.Params.Model,
	}
	if meta.Model == "" {
		meta.Model = modelFallback
	}

	segs := make([]TranscriptSegment, 0, len(raw.Transcription))
	for i, t := range raw.Transcription {
		text := strings.TrimSpace(t.Text)
		if text == "" {
			continue
		}
		segs = append(segs, TranscriptSegment{
			Index:  i,
			StartS: float64(t.Offsets.From) / 1000.0,
			EndS:   float64(t.Offsets.To) / 1000.0,
			Text:   text,
		})
	}

	// Re-index after filtering empty segments.
	for i := range segs {
		segs[i].Index = i
	}

	return segs, meta, nil
}
