# Research: goblin Core Pipeline

**Date**: 2026-08-10
**Branch**: 001-core-pipeline

All design decisions resolved inline. No external research agents dispatched —
the technical context is fully determined from project constraints and known
tool interfaces.

---

## Decision 1: Package Layout

**Decision**: `cmd/goblin/main.go` entry point; internal logic in `internal/`
sub-packages, one per pipeline stage.

```
cmd/goblin/main.go     — flag parsing, stage orchestration, exit codes
internal/probe/        — ffprobe subprocess wrapper; ProbeResult type
internal/extract/      — ffmpeg frame extraction; Scene/Frame types
internal/transcribe/   — whisper subprocess + mkvextract subtitle path
internal/manifest/     — MANIFEST.json, probe.json, transcript.json writing
internal/grid/         — contact sheet generation via ffmpeg tile filter
```

**Rationale**: Each internal package maps 1:1 to a pipeline stage
(Constitution IV — Composable Pipeline). A caller can import any stage
independently. The `cmd/` wrapper holds the only policy logic (which stages
run, in what order). Standard Go CLI layout for a named binary.

**Alternatives considered**: Single `main.go` at root — simpler, but collapses
stage boundaries and makes unit testing harder. Rejected.

---

## Decision 2: Go Version

**Decision**: Go 1.26.5 — same as `conform`, the sibling Positronikal Go binary.

**Rationale**: Consistency across the toolchain. 1.26.5 is the current version
on this workstation. `goimports@latest` requires ≥1.25 (learned from conform's
CI fix).

---

## Decision 3: Frame Extraction — Single-Pass with Metadata Filter

**Decision**: Single `ffmpeg` invocation using `select` + `metadata=print`
filters. Produces PNG frames and a sidecar metadata file with per-frame
timestamps in one pass.

```bash
ffmpeg -i INPUT \
  -vf "select=gt(scene\,THRESHOLD),metadata=print:file=OUTDIR/.frame-meta" \
  -vsync vfr -q:v 2 \
  OUTDIR/frames/frame_%05d.png
```

`metadata=print` writes `frame_num=N\npts_time=T\nlavfi.scene_score=S\n\n`
blocks to `.frame-meta` for each output frame. goblin parses this file to
associate each frame PNG with its source timestamp.

**Alternatives considered**:
- Two-pass (ffprobe timestamps → ffmpeg `-ss` seek per frame): correct but
  slower on long videos; one seek per scene change.
- `showinfo` filter writing to stderr: requires stderr parsing, which is fragile
  and conflicts with error reporting. Rejected.

---

## Decision 4: Audio Extraction Before Whisper

**Decision**: Always extract audio to a temporary WAV file before calling
whisper-cli. Whisper requires PCM WAV, 16 kHz, mono.

```bash
ffmpeg -i INPUT -vn -acodec pcm_s16le -ar 16000 -ac 1 OUTDIR/.audio.wav
whisper-cli -m MODEL -f OUTDIR/.audio.wav -oj -ojf -of OUTDIR/transcript
```

The `.audio.wav` tempfile is deleted after transcription completes.
Whisper appends `.json` to the `-of` path, producing `transcript.json`
directly in the output directory.

**Rationale**: Decouples audio format from video container. Whisper's native
input handling for non-WAV containers is unreliable across versions.

---

## Decision 5: Whisper Output Normalization

**Decision**: goblin normalizes whisper's JSON output into its own transcript
schema before writing `transcript.json`. Downstream consumers read goblin's
schema, not whisper's.

**Rationale**: Insulates consumers from whisper version changes (Constitution
III — Machine-First Output). The transcript schema is a goblin contract, not
a whisper dependency.

---

## Decision 6: Embedded Subtitle Preference

**Decision**: When the probe detects subtitle tracks AND `mkvextract` is on
PATH, goblin extracts the first subtitle track and converts it to the transcript
schema instead of running whisper. The `--prefer-whisper` flag overrides this.

**Rationale**: Professional embedded subtitles are more accurate than ASR.
Using them is free (no GPU time). FR-006 requires this preference.

---

## Decision 7: Grid Mode Implementation

**Decision**: ffmpeg `tile` filter, not Go `image` compositing.

```bash
ffmpeg -framerate 1 -pattern_type glob \
  -i "OUTDIR/frames/frame_*.png" \
  -vf "tile=COLSxROWS:padding=2" \
  -frames:v 1 OUTDIR/grids/grid_%03d.png
```

MANIFEST.json references grid images instead of individual frames when grid
mode is active. Individual frames are still written; grids are an additional
output.

**Rationale**: ffmpeg already required; no new dependency. Go stdlib image
compositing is correct but verbose for something ffmpeg handles in one flag.

---

## Decision 8: CLI Flag Design

**Decision**: Use Go stdlib `flag` package (not `pflag` or `cobra`). goblin is
a single command with no subcommands.

```
goblin [flags] FILE

Flags:
  -o, -output DIR          Output directory (default: FILE_goblin/ next to FILE)
  -t, -threshold FLOAT     Scene detection threshold 0.0–1.0 (default: 0.4)
  -probe-only              Run probe stage only; skip frames and transcript
  -no-frames               Skip frame extraction
  -no-transcript           Skip transcription
  -overwrite               Overwrite existing output directory
  -whisper-cmd STRING      Whisper binary name or path (default: whisper-cli;
                           env: GOBLIN_WHISPER_CMD)
  -model STRING            Whisper model path (env: GOBLIN_WHISPER_MODEL)
  -grid                    Enable contact sheet grid output
  -grid-cols INT           Grid columns (default: 4)
  -frame-warn INT          Warn if frame count exceeds N (default: 50)
  -threads INT             Thread hint passed to ffmpeg and whisper (default: 0 = auto)
  -version                 Print version and exit
  -quiet                   Suppress progress output; errors only
```

**Rationale**: stdlib `flag` is sufficient. cobra/pflag add dependency weight
that is not justified by a single-command tool (Constitution — Rule of Parsimony).

---

## Decision 9: Environment Variable Precedence

**Decision**: CLI flag overrides env var; env var overrides compiled default.
Applies to: `GOBLIN_WHISPER_CMD`, `GOBLIN_WHISPER_MODEL`.

**Rationale**: Standard Unix convention. Allows a system-wide model path in the
environment while still permitting per-invocation override.

---

## Decision 10: Output Directory Default Naming

**Decision**: If `-output` is not specified, the output directory is named
`<basename>_goblin` in the same directory as the input file.

Example: `/d/Video/interview.mp4` → `/d/Video/interview_goblin/`

**Rationale**: Keeps output co-located with input by default; naming makes
origin obvious. The `_goblin` suffix is visually distinctive and avoids
collision with likely directory names.

---

## Decision 11: Tool Presence Check

**Decision**: At startup, goblin checks for required tools before reading
the input file. Check: `exec.LookPath("ffprobe")`, `exec.LookPath("ffmpeg")`.
Whisper is checked only if transcription is not disabled. mkvextract is probed
opportunistically (absence is not an error).

**Rationale**: FR-002 requires a sub-1-second diagnostic on missing tools.
`exec.LookPath` is instant. Failing early is better than failing mid-pipeline
after spending time on frame extraction. Constitution — Rule of Repair.

---

## Decision 12: MANIFEST Schema Version

**Decision**: `schema_version: "1"` (string, not integer). Increment to "2"
on any breaking change. Consumers check this field before parsing.

**Rationale**: String allows future dotted versions ("1.1") without a type
change. Assumption from spec: schema is versioned from v1.
