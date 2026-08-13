# Contract: MANIFEST.json Schema v1

**File**: `MANIFEST.json` in the output directory root
**Schema version**: `"1"`

---

## Top-Level Structure

```json
{
  "schema_version": "1",
  "goblin_version": "0.1.0",
  "generated_at": "2026-08-10T14:32:00Z",
  "source_path": "/d/Video/interview.mp4",
  "duration_s": 4523.7,
  "stages_run": ["probe", "extract", "transcribe"],
  "probe_path": "probe.json",
  "transcript_path": "transcript.json",
  "grid_mode": false,
  "grid_cols": 0,
  "scenes": [ ... ],
  "warnings": []
}
```

## Field Definitions

| Field | Type | Always present | Description |
|---|---|---|---|
| `schema_version` | string | yes | `"1"` in v0.1.0; increment on breaking change |
| `goblin_version` | string | yes | goblin semver string |
| `generated_at` | string | yes | ISO 8601 UTC (e.g., `"2026-08-10T14:32:00Z"`) |
| `source_path` | string | yes | Absolute path to the source file at time of processing |
| `duration_s` | float | yes | Source file duration in seconds |
| `stages_run` | []string | yes | Which stages ran: subset of `["probe","extract","transcribe"]` |
| `probe_path` | string | yes | Always `"probe.json"` |
| `transcript_path` | string | no | `"transcript.json"` if transcription ran; absent or `""` otherwise |
| `grid_mode` | bool | yes | Whether grids were produced |
| `grid_cols` | int | yes | Grid column count; `0` if `grid_mode` is false |
| `scenes` | []SceneEntry | yes | Ordered scene list; length ≥ 1 |
| `warnings` | []string | yes | Non-fatal conditions; empty array if none |

## SceneEntry

```json
{
  "index": 0,
  "start_s": 0.0,
  "end_s": 14.83,
  "frame": "frames/frame_00001.png",
  "grid": "",
  "transcript_segments": [0, 1, 2]
}
```

| Field | Type | Description |
|---|---|---|
| `index` | int | 0-based; ordered by `start_s` |
| `start_s` | float | Scene start in seconds |
| `end_s` | float | Scene end in seconds |
| `frame` | string | Relative path to representative PNG |
| `grid` | string | Relative path to grid sheet containing this scene; `""` if not grid mode |
| `transcript_segments` | []int | Indices of transcript segments overlapping this scene's time range; empty array if no transcript |

---

## probe.json Structure

```json
{
  "schema_version": "1",
  "path": "/d/Video/interview.mp4",
  "duration_s": 4523.7,
  "size_bytes": 2147483648,
  "format_name": "mov,mp4,m4a,3gp,3g2,mj2",
  "video_codec": "h264",
  "audio_codec": "aac",
  "resolution": "1920x1080",
  "fps": 29.97,
  "video_stream_index": 0,
  "audio_stream_index": 1,
  "has_video": true,
  "has_audio": true,
  "subtitle_tracks": []
}
```

SubtitleTrack entry:
```json
{
  "index": 2,
  "language": "en",
  "codec": "subrip",
  "title": "English"
}
```

---

## transcript.json Structure

```json
{
  "schema_version": "1",
  "source": "whisper",
  "model": "/d/models/whisper-models/ggml-large-v3-turbo.bin",
  "language": "en",
  "segments": [
    {
      "index": 0,
      "start_s": 0.0,
      "end_s": 3.42,
      "text": "Welcome to the interview."
    },
    {
      "index": 1,
      "start_s": 3.42,
      "end_s": 7.81,
      "text": "Today we are talking about security research."
    }
  ]
}
```

`source` is `"whisper"` when ASR was used, `"subtitle"` when embedded subtitle
tracks were extracted. Downstream consumers use `source` to calibrate confidence
— subtitle extraction is generally more accurate than ASR.

---

## Backward Compatibility Policy

- `schema_version` is a string to allow future dotted subversions (`"1.1"`).
- Consumers SHOULD check `schema_version` before parsing.
- Adding new optional fields to any object is a non-breaking change.
- Removing fields, changing field types, or changing field semantics requires
  incrementing `schema_version` to `"2"`.
