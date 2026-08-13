# Using Goblin

Goblin extracts keyframes and a timestamped transcript from any ffprobe-readable
video or audio file and writes them to a self-contained output directory. The
output is designed to be consumed by Claude or any editing tool without further
parsing or transformation.

---

## Prerequisites

All three tools must be on your PATH:

```
ffprobe    (FFmpeg suite — https://ffmpeg.org/download.html)
ffmpeg     (FFmpeg suite)
whisper-cli (whisper.cpp — https://github.com/ggerganov/whisper.cpp)
```

A whisper model file is also required. Download one from the whisper.cpp
releases or Hugging Face. The large-v3-turbo model is a good default for most
content:

```
whisper-cli -m /path/to/ggml-large-v3-turbo.bin --help
```

Goblin checks for all required tools at startup and exits 2 with a named
diagnostic if any are missing.

---

## Installation

Build from source:

```sh
git clone https://github.com/hoyt-harness/goblin
cd goblin
make build
# Binary: bin/goblin (or bin/goblin.exe on Windows)
```

Copy the binary to a directory on your PATH.

---

## Worked examples

### 1. Full pipeline — frames, transcript, and MANIFEST

```sh
goblin -model /path/to/ggml-large-v3-turbo.bin interview.mp4
```

Output directory `interview_goblin/` contains:

```
MANIFEST.json       — scene index with timestamps and transcript cross-links
probe.json          — technical metadata (codec, resolution, duration)
transcript.json     — timestamped transcript segments
frames/
  frame_00001.png   — representative frame for each detected scene
  frame_00002.png
  ...
```

Read `MANIFEST.json` to get everything Claude needs to describe the video:
scene timecodes, frame paths, and which transcript segments belong to each scene.

### 2. Technical probe only

```sh
goblin -probe-only interview.mp4
```

Writes `probe.json` and a minimal `MANIFEST.json` in under a second. Useful for
checking codec, duration, and stream presence before committing to a full run.
No frames or transcript are produced.

### 3. Frames without transcription

```sh
goblin -no-transcript interview.mp4
```

Extracts keyframes only. Use this when the audio contains music, silence, or a
language the whisper model handles poorly.

### 4. Grid mode — contact sheets for large frame sets

```sh
goblin -grid -grid-cols 4 -model /path/to/model.bin long_film.mp4
```

In addition to the standard output, goblin produces contact sheets in `grids/`
where each sheet tiles up to `grid-cols × grid-cols` frames (default: 4×4 = 16
per page). Each scene's `grid` field in `MANIFEST.json` references the sheet
containing its frame. Useful for visual review of long videos without opening
every individual frame.

### 5. Embedded subtitle extraction (MKV files)

```sh
goblin -model /path/to/model.bin subtitled.mkv
```

When `mkvextract` is on PATH and the input contains subtitle tracks, goblin uses
the embedded subtitles as the transcript instead of running whisper. The
`transcript.json` source field is `"subtitle"`. To force whisper regardless:

```sh
goblin -prefer-whisper -model /path/to/model.bin subtitled.mkv
```

---

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-model PATH` | `$GOBLIN_WHISPER_MODEL` | Whisper model file |
| `-output DIR` | `<file>_goblin/` | Output directory |
| `-threshold N` | `0.4` | Scene detection sensitivity (0.0–1.0; lower = more scenes) |
| `-probe-only` | false | Probe and write probe.json only |
| `-no-frames` | false | Skip frame extraction |
| `-no-transcript` | false | Skip transcription |
| `-prefer-whisper` | false | Use whisper even when embedded subtitles exist |
| `-overwrite` | false | Replace an existing output directory |
| `-grid` | false | Produce contact sheet grid images |
| `-grid-cols N` | 4 | Columns per grid page (rows = cols) |
| `-frame-warn N` | 50 | Warn when frame count exceeds N |
| `-threads N` | 0 | Thread count hint for ffmpeg and whisper (0 = auto) |
| `-whisper-cmd NAME` | `whisper-cli` or `$GOBLIN_WHISPER_CMD` | Whisper binary |
| `-quiet` | false | Suppress progress lines (errors always print) |
| `-version` | — | Print version and exit 0 |

---

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Bad arguments (missing FILE, extra positional args) |
| 2 | Tool or subprocess error (missing ffprobe/ffmpeg/whisper, subprocess failure) |
| 3 | Input file not found or not readable |
| 4 | Output directory error (already exists without -overwrite, or cannot be created) |

---

## Environment variables

| Variable | Purpose |
|---|---|
| `GOBLIN_WHISPER_CMD` | Default whisper binary name or path (overridden by `-whisper-cmd`) |
| `GOBLIN_WHISPER_MODEL` | Default model path (overridden by `-model`) |

---

## MANIFEST.json schema

The output directory is anchored by `MANIFEST.json`. All paths inside are
relative to the output directory and use forward slashes on all platforms.

```json
{
  "schema_version": "1",
  "goblin_version": "0.1.0",
  "generated_at": "2026-08-13T12:00:00Z",
  "source_path": "/abs/path/to/input.mp4",
  "duration_s": 169.4,
  "stages_run": ["probe", "extract", "transcribe"],
  "probe_path": "probe.json",
  "transcript_path": "transcript.json",
  "grid_mode": false,
  "grid_cols": 0,
  "scenes": [
    {
      "index": 0,
      "start_s": 0.0,
      "end_s": 12.3,
      "frame": "frames/frame_00001.png",
      "grid": "",
      "transcript_segments": [0, 1, 2]
    }
  ],
  "warnings": []
}
```

`transcript_segments` is a list of indices into `transcript.json`'s `segments`
array. All indices are guaranteed in-bounds when the manifest is written.

---

## Troubleshooting

See `BUGS` for known issues and workarounds.
