# Quickstart Validation Guide: goblin Output Control (002)

**Date**: 2026-08-13
**Branch**: `002-output-control`

These scenarios test the three new flags from 002. Run all seven existing
Quickstart Scenarios from `001-core-pipeline/quickstart.md` first to confirm
no regressions, then run these.

---

## Prerequisites

All 001 prerequisites apply. Additionally:

```bash
# Verify ffprobe can report dimensions (used to validate SC-001/SC-002)
ffprobe -v error -select_streams v:0 -show_entries stream=width,height \
  -of csv=p=0 test.mp4
```

A high-resolution source (1080p or above) is recommended for Scenario 1 to
confirm visible scaling.

---

## Scenario 8: Frame Dimension Cap

Use a 1080p or higher resolution source:

```bash
goblin -frame-max-dim 1280 -no-transcript goblin-test.mp4
```

**Expected**:
- Exit code 0
- All PNG files in `goblin-test_goblin/frames/` have longest edge ≤ 1280
- Verify with: `ffprobe -v error -select_streams v:0 \
  -show_entries stream=width,height -of csv=p=0 frames/frame_00001.png`
- For a 1920×1080 source: frames should be 1280×720
- For a 1080×1920 portrait source: frames should be 720×1280
- `MANIFEST.json` contains `"frame_max_dim": 1280`

**No upscale check**: Run with `-frame-max-dim 9999` on a 1080p source and
confirm frames are written at native 1920×1080, not upscaled.

---

## Scenario 9: Rectangular Grid

Use a source with > 24 scenes (longer video recommended):

```bash
goblin -grid -grid-cols 6 -grid-rows 3 -no-transcript goblin-test.mp4
```

**Expected**:
- Exit code 0
- `grids/grid_001.png` exists
- Grid image width ≈ 6 × frame_width; grid image height ≈ 3 × frame_height
- `MANIFEST.json` contains `"grid_cols": 6` and `"grid_rows": 3`
- Each scene's `grid` field references the correct grid file

**Square backward compat**: Run with only `-grid -grid-cols 4` (no `-grid-rows`)
and confirm pages are 4×4 as before, with no `grid_rows` field in MANIFEST.

---

## Scenario 10: JPEG Frame Format

```bash
goblin -frame-format jpg -no-transcript goblin-test.mp4
```

**Expected**:
- Exit code 0
- All files in `goblin-test_goblin/frames/` have `.jpg` extension
- No `.png` files in `frames/`
- `MANIFEST.json` scene `frame` paths end in `.jpg`
- `MANIFEST.json` contains `"frame_format": "jpg"`
- JPEG files are valid images (open in any image viewer)

**Size check**: Compare total size of `frames/` to a PNG run on the same
source. JPEG output should be substantially smaller (typically 5–15×).

**Quality override**: Run with `-frame-format jpg -frame-quality 10` and
confirm the flag is accepted without error (files will be lower quality but
still valid JPEG).

---

## Scenario 11: Combined Output Control

```bash
goblin -frame-max-dim 1280 -frame-format jpg -frame-quality 3 \
       -grid -grid-cols 6 -grid-rows 3 \
       -model /d/models/whisper-models/ggml-large-v3-turbo.bin \
       goblin-test.mp4
```

**Expected**:
- Exit code 0
- All frame files in `frames/` are `.jpg` with longest edge ≤ 1280
- Grid files in `grids/` are `.png` in a 6×3 layout
- `MANIFEST.json` contains:
  - `"frame_format": "jpg"`
  - `"frame_max_dim": 1280`
  - `"grid_cols": 6`
  - `"grid_rows": 3`
- Consumer validate script exits 0 on the output

**Consumer validate**: The existing `test/consumer_validate.go` must handle
`.jpg` frame paths — it should not hardcode `.png` in any path check.

---

## Regression Check

After implementing 002, all seven original Quickstart Scenarios must still
pass with no flags changed. Specifically:
- Scenario 1 (full pipeline): frames are still `.png`, no `frame_format`
  field in MANIFEST (or `"png"` if always written — check the spec)
- Scenario 7 (grid mode): grid pages remain 4×4 when no `-grid-rows` is
  specified

---

## Success Criteria Traceability

| Criterion | Validated by |
|---|---|
| SC-001: Dimension cap | Scenario 8 (ffprobe dimension check) |
| SC-002: Rectangular grid | Scenario 9 (grid image dimension check) |
| SC-003: JPEG size reduction | Scenario 10 (compare `frames/` sizes) |
| SC-004: No regressions | All seven original scenarios + default-flag runs |
| SC-005: Combined flags | Scenario 11 (consumer validate exits 0) |
