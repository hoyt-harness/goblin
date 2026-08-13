# Research: goblin Output Control (002)

**Branch**: `002-output-control` | **Date**: 2026-08-13

Four design questions resolved before writing the plan.

---

## D001: Image token cost driver — pixel dimensions, not bytes

**Question**: Does `-frame-format jpg` (smaller files) reduce Claude's token
cost for reading frames, justifying it as a context-efficiency tool?

**Finding**: Image token cost is determined by pixel dimensions, not file size.
Claude tiles images into 750×750-pixel tiles; each tile costs a fixed number
of tokens regardless of whether the source is PNG or JPEG. A 1280×720 JPEG
and a 1280×720 PNG both produce 2 tiles and cost the same tokens. File size
is irrelevant to the API.

**Decision**: The real context-efficiency lever is `-frame-max-dim N`, which
reduces the number of tiles per image by shrinking the frame's pixel dimensions.
`-frame-format jpg` is a storage/transfer optimization only and is correctly
classified as P3. The spec's P1/P2/P3 ordering reflects this.

**Consequence**: `-frame-max-dim 1280` on a 4K source reduces a frame from
~24 tiles (~36,000 tokens) to ~2 tiles (~3,000 tokens) — a 12× reduction.
This makes full-pipeline analysis of long 4K video practical within a single
context window. JPEG at the same 1280px width saves disk but does not change
the token count.

---

## D002: WebP deferred — conflicting quality scale conventions

**Question**: Should 002 include `-frame-format webp` alongside JPEG?

**Finding**: ffmpeg's quality parameter (`-q:v`) has opposite semantics
for JPEG and WebP:
- JPEG (`mjpeg` encoder): `-q:v 1` is maximum quality, `-q:v 31` is minimum.
  Lower is better.
- WebP (`libwebp` encoder): `-q:v 0` is minimum quality, `-q:v 100` is maximum.
  Higher is better. This encoder also uses `-quality N` as the primary flag.

A shared `-frame-quality N` flag with a single documented range (1–31, lower
= better) cannot correctly apply to both codecs without codec-specific remapping,
which adds complexity and is a potential source of user confusion.

**Decision**: 002 ships JPEG only. The `-frame-format` flag accepts `png|jpg`.
WebP support will be a separate spec with its own quality flag (or with the
remapping logic explicitly designed and tested). `libwebp` is confirmed
present on the primary workstation — availability is not the constraint.

---

## D003: JPEG quality value and default

**Question**: What should the default JPEG quality be, and how is it expressed
in the flag?

**Finding**: ffmpeg's `-q:v` scale for JPEG (`mjpeg` codec) runs 1–31, where
1 is the highest quality (~95% equivalent) and 31 is the lowest (~20%).
Common defaults in video/photo tooling:
- Value 2 is what goblin currently uses for PNG frames (`-q:v 2`), which is
  approximately "very high quality" in the PNG case (where `-q:v` controls
  compression speed, not quality loss).
- Value 3 for JPEG is approximately 90% JPEG quality — visually lossless for
  scene recognition purposes, with typical file sizes 80–150 KB at 1280px
  width.

**Decision**: Default `-frame-quality 3`. Expose the flag using ffmpeg's
native 1–31 scale directly (no remapping), documented as "ffmpeg `-q:v` value
for JPEG; lower = higher quality, range 1–31." This is honest about what the
flag does and avoids a remapping layer. Users who need a specific quality level
can consult ffmpeg documentation.

Emit a warning if `-frame-quality` is set with `-frame-format png`, since the
flag has no effect for lossless PNG.

---

## D004: Grid rows default and tile filter construction

**Question**: How should `-grid-rows 0` (the default) interact with `-grid-cols`?
What is the tile filter expression?

**Finding**: The current ffmpeg tile filter call uses `tile=COLSxCOLS` (square
pages). The tile filter syntax is `tile=WxH` where W = columns and H = rows.

**Decision**: When `-grid-rows 0` (default), rows = cols, preserving current
square-page behavior exactly. When `-grid-rows N > 0`, rows = N. The tile
filter becomes:

```
rows := cfg.GridRows
if rows == 0 {
    rows = cfg.GridCols
}
// tile filter: tile=COLSxROWS
```

This means:
- `-grid -grid-cols 4` → 4×4 = 16 frames/page (unchanged)
- `-grid -grid-cols 8 -grid-rows 4` → 8×4 = 32 frames/page
- `-grid -grid-rows 3` → 4×3 = 12 frames/page (default cols=4 applies)

The same rows value is used in `SortedFrameNames()` / grid path computation
in `internal/manifest` to correctly associate frames with their grid page.

---

## Summary of Design Decisions

| # | Decision |
|---|---|
| D001 | `-frame-max-dim` is the context-efficiency lever; `-frame-format jpg` is storage/transfer only |
| D002 | WebP deferred from 002; JPEG only in this spec |
| D003 | Default `-frame-quality 3`; expose as native ffmpeg scale (1–31, lower=better) |
| D004 | `-grid-rows 0` = same as cols (square, backward compat); tile filter = `COLSxROWS` |
