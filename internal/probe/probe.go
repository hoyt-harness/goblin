// SPDX-License-Identifier: GPL-3.0-or-later
package probe

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type SubtitleTrack struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Codec    string `json:"codec"`
	Title    string `json:"title"`
}

type ProbeResult struct {
	Path             string          `json:"path"`
	DurationS        float64         `json:"duration_s"`
	SizeBytes        int64           `json:"size_bytes"`
	FormatName       string          `json:"format_name"`
	VideoCodec       string          `json:"video_codec"`
	AudioCodec       string          `json:"audio_codec"`
	Resolution       string          `json:"resolution"`
	FPS              float64         `json:"fps"`
	VideoStreamIndex int             `json:"video_stream_index"`
	AudioStreamIndex int             `json:"audio_stream_index"`
	SubtitleTracks   []SubtitleTrack `json:"subtitle_tracks"`
	HasVideo         bool            `json:"has_video"`
	HasAudio         bool            `json:"has_audio"`
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	Index      int               `json:"index"`
	CodecType  string            `json:"codec_type"`
	CodecName  string            `json:"codec_name"`
	Width      int               `json:"width"`
	Height     int               `json:"height"`
	RFrameRate string            `json:"r_frame_rate"`
	Tags       map[string]string `json:"tags"`
}

type ffprobeFormat struct {
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
}

func Probe(path string) (*ProbeResult, error) {
	cmd := exec.Command("ffprobe",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		"-loglevel", "error",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	var raw ffprobeOutput
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("ffprobe output parse: %w", err)
	}

	r := &ProbeResult{
		Path:             path,
		FormatName:       raw.Format.FormatName,
		VideoStreamIndex: -1,
		AudioStreamIndex: -1,
		SubtitleTracks:   []SubtitleTrack{},
	}

	r.DurationS, _ = strconv.ParseFloat(raw.Format.Duration, 64)
	r.SizeBytes, _ = strconv.ParseInt(raw.Format.Size, 10, 64)

	for _, s := range raw.Streams {
		switch s.CodecType {
		case "video":
			if !r.HasVideo {
				r.HasVideo = true
				r.VideoStreamIndex = s.Index
				r.VideoCodec = s.CodecName
				if s.Width > 0 && s.Height > 0 {
					r.Resolution = fmt.Sprintf("%dx%d", s.Width, s.Height)
				}
				r.FPS = parseRational(s.RFrameRate)
			}
		case "audio":
			if !r.HasAudio {
				r.HasAudio = true
				r.AudioStreamIndex = s.Index
				r.AudioCodec = s.CodecName
			}
		case "subtitle":
			track := SubtitleTrack{
				Index: s.Index,
				Codec: s.CodecName,
			}
			if s.Tags != nil {
				track.Language = s.Tags["language"]
				track.Title = s.Tags["title"]
			}
			r.SubtitleTracks = append(r.SubtitleTracks, track)
		}
	}

	return r, nil
}

func parseRational(s string) float64 {
	var num, den float64
	if _, err := fmt.Sscanf(s, "%f/%f", &num, &den); err != nil || den == 0 {
		return 0
	}
	return num / den
}
