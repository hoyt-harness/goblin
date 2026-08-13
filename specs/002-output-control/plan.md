# Implementation Plan: goblin Output Control (002)

**Branch**: `002-output-control` | **Date**: 2026-08-13 | **Spec**: [spec.md](spec.md)

## Summary

Three new flags that give the operator control over the form of goblin's output
without altering the analysis pipeline: `-frame-max-dim N` caps frame pixel
dimensions (the primary context-efficiency lever), `-grid-rows N` enables
rectangular grid pages, and `-frame-format png|jpg` selects the frame image
codec. No new Go package dependencies. The existing MANIFEST schema gains three
optional fields; all existing consumers continue to work.

## Technical Context

**Language/Version**: Go 1.26.5 — unchanged from v0.1.0.

**Dependencies**: stdlib only for goblin itself. No new Go package dependencies.
Runtime tools: ffmpeg gains one new flag combination (`scale` filter for
`-frame-max-dim`; `-q:v N` for JPEG; `tile=COLSxROWS` already used).

**Storage**: Filesystem. Frame files change extension when `-frame-format jpg`.
Grid output files remain `.png` unconditionally.

**Testing**: `go test ./...` + integration tests in `test/`. Consumer validate
script must handle `.jpg` frame paths.

**Target Platform**: Windows x64 (primary). No Windows-specific code introduced.

**Constraints**: No CGO. No new external Go packages. Backward compatible —
all v0.1.0 invocations without new flags produce identical output.

## Constitution Check

| Principle | Status | Notes |
|---|---|---|
| I. Analyze, Never Modify | ✓ PASS | No changes to source file handling |
| II. Local by Default | ✓ PASS | No network calls added |
| III. Machine-First Output | ✓ PASS | JSON output unchanged; frames remain image files |
| IV. Composable Pipeline | ✓ PASS | New flags are orthogonal to stage selection |
| V. Format-Agnostic Input | ✓ PASS | Input side entirely unchanged |

No violations. No complexity justification required.

## MANIFEST Schema Additions (additive only)

Three new optional top-level fields added to MANIFEST.json when the
corresponding flag is set. Existing consumers that ignore unknown fields
continue to work unchanged.

```json
{
  "schema_version": "1",
  "frame_format": "jpg",           // new: "png" (default) or "jpg"
  "frame_max_dim": 1280,           // new: omitted when 0 (no cap)
  "grid_rows": 4,                  // new: omitted when 0 (square pages)
  ...
}
```

`frame_format` is always written (consumers can detect the frame codec without
inspecting file extensions). `frame_max_dim` and `grid_rows` are omitted from
MANIFEST when their values equal the defaults (0), to avoid polluting the
output with uninformative zero values.

## Source Locations Requiring Change

The grep of the existing codebase identified all `.png`-hardcoded sites:

**`internal/extract/extract.go`** (7 sites):
- Line 60: `"frames/frame_%05d.png"` — ffmpeg output pattern
- Line 85: `"frames/frame_00001.png"` — first-frame fallback path
- Line 92: `frames = []rawFrame{{filename: "frame_00001.png", ...}}` — raw frame record
- Lines 134, 144, 162, 179: `fmt.Sprintf("frame_%05d.png", ...)` — frame filename construction

**`internal/grid/grid.go`** (4 sites):
- Line 27: `strings.HasSuffix(e.Name(), ".png")` — frame file filter
- Line 54: `"-i", "frames/frame_%05d.png"` — ffmpeg sequential input
- Line 71: `strings.HasSuffix(e.Name(), ".png")` — post-grid frame path collector
- Line 88: `strings.HasSuffix(e.Name(), ".png")` — `SortedFrameNames()` filter

All 11 sites must be updated to use a configurable extension. The `extract`
package receives the extension from `Config` via its function signature.
The `grid` package receives it the same way.

## Key Implementation Notes

### `-frame-max-dim` — ffmpeg scale filter

When `config.FrameMaxDim > 0`, append to the `-vf` filtergraph:

```
scale=iw*min(N/iw\,N/ih):ih*min(N/iw\,N/ih):flags=lanczos
```

Or equivalently (simpler):

```
scale=N:N:force_original_aspect_ratio=decrease:flags=lanczos
```

The second form is more readable and directly expresses the intent: scale so
neither dimension exceeds N, preserve aspect ratio, never upscale. The
`force_original_aspect_ratio=decrease` option is standard ffmpeg. Use
`lanczos` for high-quality downscaling.

The scale filter is appended to the existing `select=...` filtergraph with
a comma separator:

```
-vf "select=gt(scene\,THRESHOLD),metadata=print:file=...,scale=N:N:force_original_aspect_ratio=decrease:flags=lanczos"
```

The first-frame fallback invocation (`-vframes 1 frames/frame_00001.ext`) also
needs the scale filter if `FrameMaxDim > 0`:

```bash
ffmpeg -i input -vf "scale=N:N:force_original_aspect_ratio=decrease:flags=lanczos" -vframes 1 frames/frame_00001.jpg
```

### `-frame-format jpg` — ffmpeg output extension and quality

The frame output pattern changes from `frames/frame_%05d.png` to
`frames/frame_%05d.jpg`. The `-q:v N` flag (where N = `config.FrameQuality`)
is added to the ffmpeg command for JPEG output. For PNG, `-q:v 2` remains.

### `-grid-rows` — tile filter

```go
rows := cfg.GridRows
if rows == 0 {
    rows = cfg.GridCols
}
tileFilter := fmt.Sprintf("tile=%dx%d:padding=2", cfg.GridCols, rows)
```

### Extension threading

The frame extension must be threaded from `Config` through:
1. `Extract(input, outDir, threshold, threads, maxDim, format, quality)` → write `frame_%05d.{ext}`
2. `Grid(framesDir, outDir, cols, rows, ext)` → read `frame_*.{ext}`, write `grid_%03d.png`
3. `manifest.BuildSceneRefs(scenes, frames, ext)` → frame paths in MANIFEST use `{ext}`
4. `manifest.VerifyFramePaths(outDir, scenes, ext)` → verify `.{ext}` files exist
5. `test/consumer_validate.go` → accepts any extension when checking frame paths

### MANIFEST population

`frame_format` is always written. `frame_max_dim` is written only when > 0.
`grid_rows` is written in the grid section only when `cfg.GridRows > 0`
(distinguishing "square because default" from "explicitly rectangular").

## Project Structure (additive)

```
specs/002-output-control/
├── plan.md              ← this file
├── research.md          ← 4 design decisions
├── spec.md              ← user stories, requirements, success criteria
├── contracts/
│   └── cli.md           ← updated CLI contract (new flags)
├── quickstart.md        ← 4 new validation scenarios
└── tasks.md             ← task list
```

No new source files. All changes are modifications to existing files plus
test additions.
