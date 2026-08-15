# Research: goblin Output Control (002)

**Branch**: `002-output-control` | **Date**: 2026-08-13

Two design questions resolved before writing the plan.

---

## D001: Image token cost driver — pixel dimensions, not bytes

**Question**: Does image codec selection (JPEG vs PNG) reduce Claude's token
cost for reading frames, justifying JPEG support as a context-efficiency tool?

**Finding**: Image token cost is determined by pixel dimensions, not file size.
Claude tiles images into 750×750-pixel tiles; each tile costs a fixed number
of tokens regardless of whether the source is PNG or JPEG. A 1280×720 JPEG
and a 1280×720 PNG both produce 2 tiles and cost the same tokens. File size
is irrelevant to the API.

**Decision**: The context-efficiency lever is `-frame-max-dim N`, which reduces
the number of tiles per image by shrinking the frame's pixel dimensions.
JPEG support is a storage/transfer optimization that does not change Claude's
token cost — it is deferred to a future spec. 002 implements P1 and P2 only.

**Consequence**: `-frame-max-dim 1280` on a 4K source reduces a frame from
~24 tiles (~36,000 tokens) to ~2 tiles (~3,000 tokens) — a 12× reduction.
This makes full-pipeline analysis of long 4K video practical within a single
context window.

---

## D002: Grid rows default and tile filter construction

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

The same rows value is used in the grid path computation in `main.go` to
correctly associate frames with their grid page.

---

## Summary of Design Decisions

| # | Decision |
|---|---|
| D001 | `-frame-max-dim` is the context-efficiency lever; JPEG support deferred — it solves disk/transfer, not token cost |
| D002 | `-grid-rows 0` = same as cols (square, backward compat); tile filter = `COLSxROWS` |
