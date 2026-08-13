# CLI Contract: goblin Output Control (002)

**Branch**: `002-output-control` | **Spec**: [../spec.md](../spec.md)

This document is additive to `specs/001-core-pipeline/contracts/cli.md`.
All existing flags, exit codes, and stdout/stderr conventions from 001 remain
unchanged. Only new flags are documented here.

---

## New Flags

| Flag | Type | Default | Purpose |
|---|---|---|---|
| `-frame-max-dim N` | int | `0` | Cap the longest edge of extracted frames in pixels. When 0, no scaling. Never upscales. |
| `-frame-format FORMAT` | string | `png` | Frame image codec. Accepted values: `png`, `jpg`. |
| `-frame-quality N` | int | `3` | ffmpeg `-q:v` value for JPEG encoding. Range 1–31; lower = higher quality. No effect with `-frame-format png`. |
| `-grid-rows N` | int | `0` | Rows per grid page. When 0, rows = cols (square pages, matching v0.1.0 behavior). |

---

## Flag Validation Rules

- `-frame-max-dim` must be ≥ 0. Negative values → exit 1 with diagnostic.
- `-frame-format` must be `png` or `jpg` (case-sensitive). Any other value →
  exit 1 with diagnostic listing accepted values.
- `-frame-quality` must be in range 1–31. Values outside this range → exit 1
  with diagnostic.
- `-frame-quality` with `-frame-format png` → warning to stderr, proceed.
- `-grid-rows` must be ≥ 0. Negative values → exit 1 with diagnostic.

---

## Stdout Progress Lines (additive)

No new progress lines are required. The existing `goblin: extract ...` progress
line may include dimension information when `-frame-max-dim` is active, but
this is informational only and its exact format is not contractual.

---

## MANIFEST.json Additions

Three new top-level fields appear when the corresponding flags are non-default:

```json
{
  "schema_version": "1",
  "frame_format": "jpg",
  "frame_max_dim": 1280,
  ...
  "grid_mode": true,
  "grid_cols": 6,
  "grid_rows": 3,
  ...
}
```

- `frame_format`: Always written. Values: `"png"` (default) or `"jpg"`.
- `frame_max_dim`: Written only when > 0. Integer, longest-edge cap in pixels.
- `grid_rows`: Written in MANIFEST when grid mode is active and `cfg.GridRows > 0`.
  Omitted (or consumers should treat absent as equal to `grid_cols`) when
  grid pages are square (default behavior).

---

## Frame Path Convention

When `-frame-format jpg` is active, all frame paths in MANIFEST.json use the
`.jpg` extension:

```json
{
  "scenes": [
    {
      "frame": "frames/frame_00001.jpg",
      ...
    }
  ]
}
```

Grid paths are always `.png` regardless of frame format:

```json
{
  "scenes": [
    {
      "frame": "frames/frame_00001.jpg",
      "grid": "grids/grid_001.png",
      ...
    }
  ]
}
```

---

## Exit Codes (unchanged from 001)

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Invalid arguments (including new flag validation failures) |
| 2 | Required tool not found on PATH |
| 3 | Source file not found or not readable |
| 4 | Output directory exists without `-overwrite` |
