# Implementation Plan: goblin Output Control (002)

**Branch**: `002-output-control` | **Date**: 2026-08-13 | **Spec**: [spec.md](spec.md)

## Summary

Two new flags that give the operator control over the form of goblin's output
without altering the analysis pipeline: `-frame-max-dim N` caps frame pixel
dimensions (the primary context-efficiency lever), and `-grid-rows N` enables
rectangular grid pages. No new Go package dependencies. The existing MANIFEST
schema gains two optional fields; all existing consumers continue to work.

## Technical Context

**Language/Version**: Go 1.26.5 — unchanged from v0.1.0.

**Dependencies**: stdlib only. No new Go package dependencies. Runtime tools:
ffmpeg gains one new flag combination (`scale` filter for `-frame-max-dim`;
`tile=COLSxROWS` already used).

**Testing**: `go test ./...` + integration tests in `test/`.

**Target Platform**: Windows x64 (primary). No Windows-specific code introduced.

**Constraints**: No CGO. No new external Go packages. Backward compatible —
all v0.1.0 invocations without new flags produce identical output.

## Constitution Check

| Principle | Status | Notes |
|---|---|---|
| I. Analyze, Never Modify | ✓ PASS | No changes to source file handling |
| II. Local by Default | ✓ PASS | No network calls added |
| III. Machine-First Output | ✓ PASS | JSON output unchanged; frames remain PNG |
| IV. Composable Pipeline | ✓ PASS | New flags are orthogonal to stage selection |
| V. Format-Agnostic Input | ✓ PASS | Input side entirely unchanged |

No violations. No complexity justification required.

## MANIFEST Schema Additions (additive only)

Two new optional top-level fields added to MANIFEST.json when the corresponding
flag is non-default. Existing consumers that ignore unknown fields continue to
work unchanged.

```json
{
  "schema_version": "1",
  "frame_max_dim": 1280,    // new: omitted when 0 (no cap)
  "grid_rows": 3,           // new: omitted when 0 (square pages)
  ...
}
```

Both fields are omitted from MANIFEST when their values equal the defaults (0),
to avoid polluting the output with uninformative zero values.

## Source Locations Requiring Change

**`internal/extract/extract.go`**: Add `scale` filter to the `-vf` filtergraph
when `maxDim > 0`. Applies to both the scene-detection invocation and the
first-frame fallback invocation. No extension changes — frames remain `.png`.

**`internal/grid/grid.go`**: Accept `rows int` parameter. Compute effective
rows (`if rows == 0 { rows = cols }`). Update tile filter from `tile=COLSxCOLS`
to `tile=COLSxROWS`. No other changes needed.

**`internal/manifest/manifest.go`**: Add `FrameMaxDim int` and `GridRows int`
fields to the `Manifest` struct.

**`cmd/goblin/main.go`**: Add `FrameMaxDim int` and `GridRows int` to `Config`;
wire flags; pass values through to `Extract()` and `Grid()` calls; populate
MANIFEST fields.

## Key Implementation Notes

### `-frame-max-dim` — ffmpeg scale filter

When `config.FrameMaxDim > 0`, append to the `-vf` filtergraph:

```
scale=N:N:force_original_aspect_ratio=decrease:flags=lanczos
```

The scale filter is appended to the existing `select=...` filtergraph with
a comma separator:

```
-vf "select=gt(scene\,THRESHOLD),metadata=print:file=...,scale=N:N:force_original_aspect_ratio=decrease:flags=lanczos"
```

The first-frame fallback invocation also needs the scale filter if `FrameMaxDim > 0`:

```bash
ffmpeg -i input -vf "scale=N:N:force_original_aspect_ratio=decrease:flags=lanczos" -vframes 1 frames/frame_00001.png
```

`force_original_aspect_ratio=decrease` ensures only downscaling occurs.
`lanczos` provides high-quality resampling.

### `-grid-rows` — tile filter

```go
rows := cfg.GridRows
if rows == 0 {
    rows = cfg.GridCols
}
tileFilter := fmt.Sprintf("tile=%dx%d:padding=2", cfg.GridCols, rows)
```

The page size for the grid→scene assignment in `main.go` must use the same
effective rows value:

```go
pageSize := cfg.GridCols * effectiveRows
```

## Project Structure (additive)

```
specs/002-output-control/
├── plan.md              ← this file
├── research.md          ← 2 design decisions
├── spec.md              ← user stories, requirements, success criteria
├── contracts/
│   └── cli.md           ← updated CLI contract (new flags)
├── quickstart.md        ← 2 new validation scenarios
└── tasks.md             ← task list
```

No new source files. All changes are modifications to existing files plus
test additions.
