// SPDX-License-Identifier: GPL-3.0-or-later
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hoyt-harness/goblin/internal/extract"
	"github.com/hoyt-harness/goblin/internal/probe"
	"github.com/hoyt-harness/goblin/internal/transcribe"
)

const SchemaVersion = "1"

type SceneRef struct {
	Index              int     `json:"index"`
	StartS             float64 `json:"start_s"`
	EndS               float64 `json:"end_s"`
	Frame              string  `json:"frame"`
	Grid               string  `json:"grid"`
	SceneScore         float64 `json:"scene_score"`
	TranscriptSegments []int   `json:"transcript_segments"`
}

type Manifest struct {
	SchemaVersion  string     `json:"schema_version"`
	GoblinVersion  string     `json:"goblin_version"`
	GeneratedAt    string     `json:"generated_at"`
	SourcePath     string     `json:"source_path"`
	DurationS      float64    `json:"duration_s"`
	StagesRun      []string   `json:"stages_run"`
	ProbePath      string     `json:"probe_path"`
	TranscriptPath string     `json:"transcript_path,omitempty"`
	GridMode       bool       `json:"grid_mode"`
	GridCols       int        `json:"grid_cols"`
	Scenes         []SceneRef `json:"scenes"`
	Warnings       []string   `json:"warnings"`
}

// probeJSON is the on-disk representation of probe results.
type probeJSON struct {
	SchemaVersion    string                `json:"schema_version"`
	Path             string                `json:"path"`
	DurationS        float64               `json:"duration_s"`
	SizeBytes        int64                 `json:"size_bytes"`
	FormatName       string                `json:"format_name"`
	VideoCodec       string                `json:"video_codec"`
	AudioCodec       string                `json:"audio_codec"`
	Resolution       string                `json:"resolution"`
	FPS              float64               `json:"fps"`
	VideoStreamIndex int                   `json:"video_stream_index"`
	AudioStreamIndex int                   `json:"audio_stream_index"`
	HasVideo         bool                  `json:"has_video"`
	HasAudio         bool                  `json:"has_audio"`
	SubtitleTracks   []probe.SubtitleTrack `json:"subtitle_tracks"`
}

// transcriptJSON is the on-disk representation of transcript output.
type transcriptJSON struct {
	SchemaVersion string                         `json:"schema_version"`
	Source        string                         `json:"source"`
	Model         string                         `json:"model"`
	Language      string                         `json:"language"`
	Segments      []transcribe.TranscriptSegment `json:"segments"`
}

// BuildSceneRefs cross-links scenes with transcript segments and sets end_s values.
func BuildSceneRefs(scenes []extract.Scene, segs []transcribe.TranscriptSegment, durationS float64) []SceneRef {
	if len(scenes) == 0 {
		// Always return at least one scene ref (even if there were no frames).
		return []SceneRef{
			{
				Index:              0,
				StartS:             0,
				EndS:               durationS,
				Frame:              "",
				Grid:               "",
				SceneScore:         0,
				TranscriptSegments: []int{},
			},
		}
	}

	refs := make([]SceneRef, len(scenes))
	for i, sc := range scenes {
		var endS float64
		if i+1 < len(scenes) {
			endS = scenes[i+1].StartS
		} else {
			endS = durationS
		}

		// Collect transcript segments that overlap [startS, endS).
		var overlapping []int
		for j, seg := range segs {
			if seg.StartS < endS && seg.EndS > sc.StartS {
				overlapping = append(overlapping, j)
			}
		}
		if overlapping == nil {
			overlapping = []int{}
		}

		refs[i] = SceneRef{
			Index:              i,
			StartS:             sc.StartS,
			EndS:               endS,
			Frame:              path.Join("frames", filepath.Base(sc.FramePath)),
			Grid:               "",
			SceneScore:         sc.SceneScore,
			TranscriptSegments: overlapping,
		}
	}
	return refs
}

func WriteManifest(outDir string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(filepath.Join(outDir, "MANIFEST.json"), data, 0o644)
}

func WriteProbe(outDir string, r *probe.ProbeResult) error {
	pj := probeJSON{
		SchemaVersion:    SchemaVersion,
		Path:             r.Path,
		DurationS:        r.DurationS,
		SizeBytes:        r.SizeBytes,
		FormatName:       r.FormatName,
		VideoCodec:       r.VideoCodec,
		AudioCodec:       r.AudioCodec,
		Resolution:       r.Resolution,
		FPS:              r.FPS,
		VideoStreamIndex: r.VideoStreamIndex,
		AudioStreamIndex: r.AudioStreamIndex,
		HasVideo:         r.HasVideo,
		HasAudio:         r.HasAudio,
		SubtitleTracks:   r.SubtitleTracks,
	}
	data, err := json.MarshalIndent(pj, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal probe: %w", err)
	}
	return os.WriteFile(filepath.Join(outDir, "probe.json"), data, 0o644)
}

func WriteTranscript(outDir string, segs []transcribe.TranscriptSegment, meta transcribe.TranscriptMeta) error {
	tj := transcriptJSON{
		SchemaVersion: SchemaVersion,
		Source:        meta.Source,
		Model:         meta.Model,
		Language:      meta.Language,
		Segments:      segs,
	}
	data, err := json.MarshalIndent(tj, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal transcript: %w", err)
	}
	return os.WriteFile(filepath.Join(outDir, "transcript.json"), data, 0o644)
}

// VerifyFramePaths checks that every frame and grid path referenced in the manifest exists on disk.
func VerifyFramePaths(outDir string, m *Manifest) []string {
	var warns []string
	for _, sc := range m.Scenes {
		if sc.Frame != "" {
			full := filepath.Join(outDir, filepath.FromSlash(sc.Frame))
			if _, err := os.Stat(full); err != nil {
				warns = append(warns, fmt.Sprintf("referenced frame missing: %s", sc.Frame))
			}
		}
		if sc.Grid != "" {
			full := filepath.Join(outDir, filepath.FromSlash(sc.Grid))
			if _, err := os.Stat(full); err != nil {
				warns = append(warns, fmt.Sprintf("referenced grid missing: %s", sc.Grid))
			}
		}
	}
	return warns
}
