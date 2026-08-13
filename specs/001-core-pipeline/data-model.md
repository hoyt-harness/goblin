# Data Model: goblin Core Pipeline

**Date**: 2026-08-10
**Branch**: 001-core-pipeline

---

## Entities

### ProbeResult

The technical metadata extracted from the source file by ffprobe.
Produced by `internal/probe`. Written to `probe.json`.

| Field | Type | Description |
|---|---|---|
| `path` | string | Absolute path of the source file |
| `duration_s` | float64 | Duration in seconds |
| `size_bytes` | int64 | File size in bytes |
| `format_name` | string | Container format (e.g., `mov,mp4,m4a,3gp,3g2,mj2`) |
| `video_codec` | string | Video codec name (e.g., `h264`, `hevc`) |
| `audio_codec` | string | Audio codec name (e.g., `aac`, `opus`); empty if none |
| `resolution` | string | `WIDTHxHEIGHT` (e.g., `1920x1080`) |
| `fps` | float64 | Frames per second (video stream) |
| `video_stream_index` | int | Index of the selected video stream |
| `audio_stream_index` | int | Index of the selected audio stream; -1 if none |
| `subtitle_tracks` | []SubtitleTrack | All detected subtitle tracks |
| `has_video` | bool | True if at least one video stream present |
| `has_audio` | bool | True if at least one audio stream present |

### SubtitleTrack

| Field | Type | Description |
|---|---|---|
| `index` | int | Stream index within the container |
| `language` | string | BCP-47 language code (e.g., `en`, `ja`); empty if unknown |
| `codec` | string | Subtitle codec (e.g., `subrip`, `ass`, `webvtt`) |
| `title` | string | Track title if present; empty otherwise |

---

### Scene

A contiguous time range with stable visual content. The scene list is the
primary output of the extract stage.

| Field | Type | Description |
|---|---|---|
| `index` | int | 0-based scene index |
| `start_s` | float64 | Scene start time in seconds |
| `end_s` | float64 | Scene end time in seconds (= next scene start, or file duration for last) |
| `frame_path` | string | Path to representative frame PNG, relative to output directory |
| `scene_score` | float64 | ffmpeg scene change score that triggered this boundary (0.0–1.0) |

**Invariant**: At least one Scene is always present (index 0, covering the
whole file if no scene changes detected).

---

### Frame

A single extracted keyframe. Frames are stored as PNG files in `frames/`.

| Field | Type | Description |
|---|---|---|
| `filename` | string | `frame_00001.png` format; 5-digit zero-padded |
| `timestamp_s` | float64 | Source timestamp of this frame in seconds |
| `scene_index` | int | Index of the Scene this frame represents |

---

### TranscriptSegment

A time-bounded span of speech. Normalized from whisper output or subtitle extraction.

| Field | Type | Description |
|---|---|---|
| `index` | int | 0-based segment index |
| `start_s` | float64 | Segment start time in seconds |
| `end_s` | float64 | Segment end time in seconds |
| `text` | string | Transcribed or extracted text, trimmed of leading/trailing whitespace |

---

### Manifest

The root index document. References all other output artifacts by relative
path. A consumer with only `MANIFEST.json` and the files it references has
complete output — no other state is required.

| Field | Type | Description |
|---|---|---|
| `schema_version` | string | `"1"` — increment on breaking changes |
| `goblin_version` | string | goblin binary version (semver) |
| `generated_at` | string | ISO 8601 UTC timestamp of generation |
| `source_path` | string | Absolute path of the source file |
| `duration_s` | float64 | Source file duration in seconds |
| `scenes` | []SceneRef | Ordered list of scene entries |
| `transcript_path` | string | Relative path to `transcript.json`; empty if skipped |
| `probe_path` | string | Relative path to `probe.json` (always present) |
| `grid_mode` | bool | True if contact sheet grids were produced |
| `grid_cols` | int | Columns per grid sheet (0 if grid_mode is false) |
| `warnings` | []string | Non-fatal conditions noted during processing |
| `stages_run` | []string | Which stages executed: `"probe"`, `"extract"`, `"transcribe"` |

### SceneRef (embedded in Manifest)

| Field | Type | Description |
|---|---|---|
| `index` | int | Scene index |
| `start_s` | float64 | Scene start time in seconds |
| `end_s` | float64 | Scene end time in seconds |
| `frame` | string | Relative path to the representative frame PNG |
| `grid` | string | Relative path to the grid sheet containing this scene; empty if not grid mode |
| `transcript_segments` | []int | Indices of TranscriptSegments whose time range overlaps this scene |

---

## Output Directory Layout

```
OUTPUT_DIR/
├── MANIFEST.json          — root index; always present
├── probe.json             — technical metadata; always present
├── transcript.json        — timestamped segments; present unless skipped
├── frames/
│   ├── frame_00001.png    — scene 0 representative frame
│   ├── frame_00002.png    — scene 1 representative frame
│   └── ...
└── grids/                 — present only in grid mode
    ├── grid_001.png
    └── ...
```

Relative paths in MANIFEST.json are always relative to the output directory
root. A consumer that moves the output directory intact can reconstruct all
paths without modification.

---

## State Transitions

```
INPUT FILE
    │
    ▼
[probe]──────────────────────────► probe.json
    │
    ├── has_video? ──yes──► [extract]─► frames/ + scene list
    │                                        │
    │   no-frames flag? ──yes──► skip        │
    │                                        ▼
    ├── has_audio? ──yes──► [transcribe]─► transcript.json
    │   no-transcript? ──yes──► skip
    │   subtitle track + mkvextract? ──yes──► extract subtitles instead
    │
    └── [manifest]─► MANIFEST.json (references all above)
```

`[manifest]` always runs last. It collects references to whatever stages
completed and writes a consistent MANIFEST regardless of which stages were
skipped.

---

## Validation Rules

- `Manifest.scenes` MUST have length ≥ 1
- `Scene.start_s` < `Scene.end_s`
- `Scene.end_s` of scene N = `Scene.start_s` of scene N+1 (contiguous, no gaps)
- `Scene.end_s` of last scene ≤ `ProbeResult.duration_s` (within tolerance of 0.5s)
- `TranscriptSegment.start_s` < `TranscriptSegment.end_s`
- All relative paths in MANIFEST.json MUST reference files that exist in the output directory
- `schema_version` MUST be `"1"` in v0.1.0
