# Feature Specification: goblin Output Control

**Feature Branch**: `002-output-control`

**Created**: 2026-08-13

**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Frame Dimension Cap for Context Efficiency (Priority: P1)

As the AI consumer of goblin output, I need to cap the pixel dimensions of
extracted frames so that high-resolution source video does not generate images
that exceed my practical visual context budget. A 4K source produces frames of
~3840×2160 pixels, which costs roughly 20 tiles (~30,000 tokens) per image
at current image pricing. Scene recognition works reliably at 1280px wide
(2 tiles, ~3,000 tokens). Without a dimension cap, a 30-minute 4K video with
50 scene changes can consume more than a million tokens in frames alone, making
full-pipeline analysis impractical.

**Why this priority**: Token cost is the primary constraint on how much video
goblin can make visible to me per context window. Codec selection (JPEG vs PNG)
affects only disk and transfer size — token cost is determined entirely by
pixel dimensions. This flag is the highest-leverage output control in 002.

**Independent Test**: Can be fully tested by running goblin on a high-resolution
source with `-frame-max-dim 1280` and verifying that every extracted frame's
longer edge is ≤ 1280 pixels.

**Acceptance Scenarios**:

1. **Given** a video source with 4K (3840×2160) frames, **When** goblin is
   invoked with `-frame-max-dim 1280`, **Then** every frame in `frames/` has
   a longest edge ≤ 1280 pixels and aspect ratio is preserved (e.g. 1280×720).

2. **Given** a portrait-oriented video with 1080×1920 frames, **When** goblin
   is invoked with `-frame-max-dim 1280`, **Then** frames are scaled to
   720×1280 (height remains the long edge, capped at 1280).

3. **Given** goblin invoked with `-frame-max-dim 0` (the default), **When**
   any source resolution is used, **Then** frame dimensions are identical to
   the source — no scaling is applied.

4. **Given** a source whose longest edge is already ≤ the configured max,
   **When** goblin runs with that limit, **Then** frames are not upscaled —
   only downscaling occurs.

---

### User Story 2 — Rectangular Grid Pages (Priority: P2)

As the AI consumer of goblin grid output, I need grid contact sheets whose
aspect ratio matches the source video's frame aspect ratio. Current square
grids (N×N) waste space when analyzing widescreen content: a 4×4 grid of
16:9 frames produces a roughly square sheet with large horizontal gaps, while
a 8×3 grid of the same 24 frames fills the sheet efficiently and lets me scan
more content per image.

**Why this priority**: Grid mode is already implemented. Adding `-grid-rows`
is a small, isolated change to one ffmpeg tile filter call and requires no
external dependencies. It directly improves how grid pages are laid out for
the most common source format (widescreen video). Most useful in combination
with `-frame-max-dim`: small frames + rectangular layout maximizes frames
visible per context image.

**Independent Test**: Can be fully tested by running goblin with
`-grid -grid-cols 8 -grid-rows 4` and verifying that grid pages contain
up to 32 frames in an 8-wide, 4-tall layout.

**Acceptance Scenarios**:

1. **Given** goblin invoked with `-grid -grid-cols 8 -grid-rows 4` on a
   source with > 32 scenes, **When** grid output is produced, **Then** each
   full grid page contains exactly 32 frames in an 8-column, 4-row layout.

2. **Given** goblin invoked with only `-grid -grid-cols 4` (no `-grid-rows`),
   **When** grid output is produced, **Then** pages remain 4×4 (16 frames) —
   backward compatibility preserved.

3. **Given** goblin invoked with `-grid -grid-rows 3` (no explicit `-grid-cols`),
   **When** grid output is produced, **Then** the default 4 columns applies,
   yielding 4×3 = 12 frames per page.

4. **Given** a source with fewer frames than one full page, **When** grid is
   produced, **Then** the partial last page is written and referenced in
   MANIFEST — no frames are dropped.

---

### Edge Cases

- What happens when `-frame-max-dim` is set and `-grid` is also enabled? The
  grid reads from the dimension-capped frames. Grid output dimensions will be
  smaller than if no cap were applied; this is correct and expected behavior.
- What happens when `-grid-rows 1 -grid-cols 1`? Each frame gets its own
  "grid" page — functionally equivalent to no grid, but still valid. goblin
  MUST accept this.
- What happens when `-frame-max-dim` is larger than the source resolution?
  goblin MUST NOT upscale. Frames are written at native resolution.
- What happens when both `-grid-cols` and `-grid-rows` are 0? This cannot
  happen in practice (both have positive defaults and are validated), but if
  reached, goblin exits with a clear diagnostic.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: goblin MUST accept `-frame-max-dim N` (integer ≥ 0) to cap the
  longest edge of extracted frames. When N > 0, frames are scaled so the
  longest edge = min(source_longest_edge, N), preserving aspect ratio. When
  N = 0 (default), no scaling is applied.
- **FR-002**: goblin MUST NOT upscale frames. If the source's longest edge is
  already ≤ N, frames are written at native resolution.
- **FR-003**: goblin MUST accept `-grid-rows N` (integer ≥ 0) to set the
  number of rows per grid page independently of the number of columns. When
  N = 0 (default), rows = cols (square pages, preserving current behavior).
- **FR-004**: All existing Quickstart Scenarios (1–7) MUST continue to pass
  without modification when none of the new flags are specified.

### Key Entities

- **Frame max dimension**: An integer cap on the longest edge of an extracted
  frame PNG. Applies at the ffmpeg scale filter level, not post-extraction.
- **Grid rows**: The number of rows per grid page. Independent of `grid-cols`.
  When 0, rows = cols (square pages).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A 4K source with `-frame-max-dim 1280` produces frames whose
  longest edge is ≤ 1280 pixels, verified by reading image dimensions with
  ffprobe or equivalent.
- **SC-002**: A grid produced with `-grid-cols 8 -grid-rows 4` contains
  exactly 8 columns and 4 rows per full page, verifiable by inspecting grid
  image dimensions (width = 8 × frame_width, height = 4 × frame_height,
  approximately).
- **SC-003**: All seven existing Quickstart Scenarios exit 0 with no new flags
  specified after 002 is implemented — no regressions.
- **SC-004**: A full pipeline with `-frame-max-dim 1280 -grid -grid-cols 6
  -grid-rows 3` exits 0 and produces a valid MANIFEST.json containing
  `frame_max_dim` and `grid_rows` fields, with all referenced frame and grid
  files existing on disk.

## Assumptions

- The source of truth for image token cost is pixel dimensions, not file bytes.
  JPEG and PNG at the same pixel dimensions cost the same number of tokens
  when read by Claude. This is why `-frame-max-dim` is the sole output control
  implemented in 002; JPEG support addresses a different (disk/transfer) problem
  and is deferred to a future spec.
- ffmpeg's `scale` filter with `force_original_aspect_ratio=decrease` correctly
  handles the no-upscale constraint without additional application logic.
- Batch processing (multiple input files), URL input, and stream selection
  remain future extensions beyond 002 scope.
