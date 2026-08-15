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

Use `-grid-rows` to make rectangular pages that match the source aspect ratio
(see example 6).

### 6. Frame dimension cap — reduce token cost for high-resolution video

```sh
goblin -frame-max-dim 1280 -no-transcript lecture_4k.mp4
```

Scales all extracted frames so the longest edge is ≤ 1280 pixels, preserving
aspect ratio. For a 4K (3840×2160) source this reduces each frame from ~30,000
tokens to ~3,000 tokens when read by Claude — a 10× reduction. Goblin never
upscales; sources already smaller than the cap are written at native resolution.

`MANIFEST.json` will contain `"frame_max_dim": 1280` when the flag is set.

### 7. Rectangular grid pages

```sh
goblin -grid -grid-cols 8 -grid-rows 3 -no-transcript long_film.mp4
```

Produces contact sheets with 8 columns and 3 rows (24 frames per page) instead
of the default square layout. This fills the sheet image more efficiently for
widescreen content. Most useful in combination with `-frame-max-dim`:

```sh
goblin -frame-max-dim 640 -grid -grid-cols 8 -grid-rows 3 -no-transcript long_film.mp4
```

Small frames at 8×3 = 24 per sheet means each contact image is compact and
costs fewer tokens for the same scene coverage.

`MANIFEST.json` will contain `"grid_rows": 3` when explicitly set.
Omitted when using default square pages (`grid_rows == grid_cols`).

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
| `-grid-cols N` | 4 | Columns per grid page |
| `-grid-rows N` | 0 | Rows per grid page (0 = same as cols, square pages) |
| `-frame-max-dim N` | 0 | Cap longest frame edge in pixels; 0 = no limit; never upscales |
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
  "goblin_version": "0.2.0",
  "generated_at": "2026-08-14T12:00:00Z",
  "source_path": "/abs/path/to/input.mp4",
  "duration_s": 169.4,
  "stages_run": ["probe", "extract", "transcribe"],
  "probe_path": "probe.json",
  "transcript_path": "transcript.json",
  "frame_max_dim": 1280,
  "grid_mode": true,
  "grid_cols": 8,
  "grid_rows": 3,
  "scenes": [
    {
      "index": 0,
      "start_s": 0.0,
      "end_s": 12.3,
      "frame": "frames/frame_00001.png",
      "grid": "grids/grid_001.png",
      "scene_score": 0.72,
      "transcript_segments": [0, 1, 2]
    }
  ],
  "warnings": []
}
```

`frame_max_dim` and `grid_rows` are omitted when their values equal the
defaults (0), to avoid uninformative zero fields in the common case.

`transcript_segments` is a list of indices into `transcript.json`'s `segments`
array. All indices are guaranteed in-bounds when the manifest is written.

---

## Strategy for AI agents

This section is written for the AI consuming goblin's output, not for the human
running it. If you are an AI agent with goblin on your PATH, read this before
your first invocation on any new video.

### 1. Probe before you commit

Always run probe-only first:

```sh
goblin -probe-only -output /tmp/probe video.mp4
```

The probe takes under a second and tells you everything you need to plan the
full run: source resolution, duration, codec, whether the file has audio, and
whether embedded subtitles exist. Without this, you are guessing at how much
the full run will cost you.

From probe output, decide:
- What `-frame-max-dim` to set (see §2 below)
- Whether to skip transcription (`-no-transcript`) — if `has_audio` is false,
  or if the question is purely visual, skip it
- Whether to skip frames (`-no-frames`) — if the question is purely about
  speech content, skipping frames saves significant time
- A rough estimate of scene count: `duration_s / 10` is a loose lower bound
  at the default threshold; adjust expectations for action-heavy or static content

### 2. Set frame-max-dim from your context budget

Token cost for images is determined by pixel dimensions, not file size. Claude
tiles images into 750×750-pixel cells; each tile costs a fixed number of tokens.

Formula for a 16:9 source capped at N pixels (longest edge):

```
width  = N
height = N × 9/16
tiles  = ceil(N/750) × ceil(height/750)
```

Common values:

| `-frame-max-dim` | Frame size (16:9) | Tiles | Approximate tokens |
|---|---|---|---|
| 0 (4K source) | 3840×2160 | 18 | ~29,000 |
| 0 (1080p source) | 1920×1080 | 6 | ~9,600 |
| 1280 | 1280×720 | 2 | ~3,200 |
| 640 | 640×360 | 1 | ~1,600 |

For a video with 40 scenes at 1280px: 40 × 3,200 ≈ 128,000 tokens in frames alone,
before any text. At 640px: 40 × 1,600 ≈ 64,000 tokens. Choose the cap that
leaves room for your analysis within your context budget.

Scene recognition works reliably at 1280px for most content. Use 640px when
you need to fit many scenes and the question does not require reading text or
fine detail.

### 3. Read MANIFEST.json first; load frames selectively

After the run, read `MANIFEST.json` before opening any frame files. The manifest
gives you the complete scene index — timestamps, transcript cross-links, grid
references, and frame paths — at negligible cost. From it you can identify
which specific scenes are relevant to the question and load only those frames.

Loading all frames upfront is almost always the wrong move. A 90-minute video
at the default threshold can produce 200+ scenes.

### 4. Use transcript_segments — it is already cross-linked

Each scene in the manifest has a `transcript_segments` array. These are
indices into `transcript.json`'s `segments` array. The cross-linking is done
at write time by goblin; you do not need to scan the transcript yourself and
match timestamps. Read the relevant segment objects directly by index.

```json
// MANIFEST scene:
{ "index": 5, "start_s": 62.1, "end_s": 71.4,
  "transcript_segments": [12, 13] }

// transcript.json segment 12:
{ "index": 12, "start_s": 61.0, "end_s": 66.3, "text": "..." }
```

### 5. Tune the threshold for the question

The default threshold (0.4) is calibrated for general-purpose content. Adjust
when the question demands it:

- **Lower (0.2–0.3)**: more scenes, finer temporal resolution — use when you
  need to find a specific moment, read something on screen, or analyze rapid
  visual changes
- **Higher (0.5–0.7)**: fewer scenes, coarser — use when you need a structural
  overview of a long video and scene count is the binding cost constraint

If you overshoot (too many frames), the `-frame-warn` threshold will alert you.
Re-run with a higher threshold rather than loading all frames anyway.

### 6. Grid for scanning; individual frames for detail

Grid contact sheets (produced with `-grid`) pack multiple frames per image.
Use them for rapid scanning — "what is the general visual rhythm of this
video?" or "which section contains the diagram I'm looking for?" — and then
load the individual frame from that scene for detail work. Grid thumbnails are
too small to read text or identify faces reliably.

### 7. Suggested invocation patterns

**"What is this video about?"** — structural overview of unknown content:
```sh
goblin -probe-only -output /tmp/p video.mp4
# → read resolution, duration; decide frame-max-dim
goblin -frame-max-dim 640 -grid -grid-cols 6 -grid-rows 3 \
       -model $GOBLIN_WHISPER_MODEL video.mp4
# → read MANIFEST, skim grid pages, read transcript summary
```

**"What happens at [timestamp]?"** — targeted moment lookup:
```sh
goblin -probe-only -output /tmp/p video.mp4
goblin -frame-max-dim 1280 -threshold 0.2 -no-transcript video.mp4
# → read MANIFEST, find scenes bracketing the timestamp, load those frames
```

**"Transcribe and summarize this talk"** — speech-primary content:
```sh
goblin -no-frames -model $GOBLIN_WHISPER_MODEL talk.mp4
# → read transcript.json directly; no frame cost at all
```

**"Find the slide showing X"** — text on screen:
```sh
goblin -frame-max-dim 1280 -threshold 0.2 -no-transcript slides.mp4
# → read MANIFEST, load frames in suspected range; 1280px needed to read text
```

## Troubleshooting

See `BUGS` for known issues and workarounds.
