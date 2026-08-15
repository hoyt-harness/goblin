# CLAUDE.md — goblin

Goblin is a media analysis pipeline that extracts keyframes and timestamped
transcripts from video/audio files and emits a structured MANIFEST directory
for Claude and editing-tool consumption.

Language: Go 1.26.5. Single binary distribution. No runtime Go dependencies
beyond external CLI tools (ffprobe, ffmpeg, whisper.cpp or equivalent).

## Build

```sh
make build
# or
go build -ldflags "-X main.version=$(git describe --tags --abbrev=0)" -o bin/goblin ./cmd/goblin
```

## Test

```sh
go test ./...
bash hooks/ci-check.sh
```

Generate test fixture: `bash test/fixtures/gen.sh`

## Key files

- `cmd/goblin/main.go` — flag parsing, tool checks, pipeline orchestration
- `internal/probe/` — ffprobe subprocess; returns ProbeResult
- `internal/extract/` — ffmpeg scene detection and frame extraction
- `internal/transcribe/` — whisper subprocess and SRT subtitle extraction
- `internal/manifest/` — writes MANIFEST.json, probe.json, transcript.json
- `internal/grid/` — ffmpeg tile filter; produces contact sheet grids from frames
- `specs/001-core-pipeline/` — full spec-kit (spec, plan, tasks, contracts)
- `specs/002-output-control/` — frame-max-dim and grid-rows spec-kit

## Flags (summary)

| Flag | Default | Purpose |
|---|---|---|
| `-model PATH` | `$GOBLIN_WHISPER_MODEL` | Whisper model file |
| `-output DIR` | `<file>_goblin/` | Output directory |
| `-threshold N` | `0.4` | Scene detection sensitivity (0–1) |
| `-probe-only` | false | Probe + probe.json only; skip extract/transcribe/grid |
| `-no-frames` | false | Skip frame extraction |
| `-no-transcript` | false | Skip transcription |
| `-prefer-whisper` | false | Use whisper even when embedded subtitles exist |
| `-overwrite` | false | Replace existing output directory |
| `-grid` | false | Produce contact sheet grid images in `grids/` |
| `-grid-cols N` | 4 | Columns per grid page |
| `-grid-rows N` | 0 | Rows per grid page (0 = same as cols, square pages) |
| `-frame-max-dim N` | 0 | Cap longest frame edge in pixels; 0 = no limit; never upscales |
| `-frame-warn N` | 50 | Warn when frame count exceeds N |
| `-threads N` | 0 | Thread count for ffmpeg and whisper (0 = auto) |
| `-quiet` | false | Suppress progress lines |
| `-whisper-cmd NAME` | `$GOBLIN_WHISPER_CMD` or `whisper-cli` | Whisper binary |
| `-version` | — | Print version and exit 0 |

## Output structure

```
<file>_goblin/
  MANIFEST.json   — top-level index (scenes, stages_run, warnings)
  probe.json      — ffprobe stream and format metadata
  transcript.json — whisper/subtitle segments (omitted with -no-transcript)
  frames/
    frame_00001.png  — one PNG per detected scene change
    ...
  grids/             — only with -grid
    grid_001.png     — contact sheet, cols×cols frames per page
    ...
```

## Grid mode

`-grid` tiles all extracted frames into contact sheets using ffmpeg's `tile`
filter. Default page size = `grid-cols × grid-cols` (square, 16 frames per
sheet with the default cols=4). Use `-grid-rows N` for rectangular pages —
e.g. `-grid-cols 8 -grid-rows 3` yields 24 frames per sheet, which fills
a 16:9 contact image more efficiently. Each scene's `grid` field in
MANIFEST.json references the sheet containing its frame. Partial last pages
are always written (never dropped).

## Frame dimension cap

`-frame-max-dim N` scales extracted frames so the longest edge is ≤ N pixels.
Aspect ratio is preserved. Goblin never upscales — if the source is already
smaller than N, frames are written at native resolution. Token cost to Claude
is determined by pixel dimensions, not file size; this flag is the primary
lever for fitting long or high-resolution video into a context window.
Example: `-frame-max-dim 1280` reduces a 4K frame from ~30,000 tokens to
~3,000 tokens — a 10× reduction.

## Known limitations

- Scene 0 starts at first detected change (not necessarily t=0) — content
  before the first scene change has no representative frame

Vault documentation: `C:\Users\hoyth\Obsidian\Positronikal\03-OPERATIONS\Engineering\`

Standards: `D:\Engineering\PositronikalCodingStandards\standards\`
Coding Bible: `D:\Engineering\_references\CODING_BIBLE.md`
