# Tasks: goblin Output Control (002)

**Branch**: `002-output-control` | **Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

**Format**: `[ID] [P?] [Story?] Description — file path`
- **[P]**: Parallelizable — different files, no incomplete dependencies
- **[US#]**: Maps to User Story # from spec.md

---

## Phase 1: Config and Flag Parsing

**Purpose**: Add new flags to `Config` struct and wire up validation. All later
phases depend on this.

- [ ] T001 Add `FrameMaxDim int`, `FrameFormat string`, `FrameQuality int`, `GridRows int` to `Config` struct in `cmd/goblin/main.go`
- [ ] T002 Wire new flags in `cmd/goblin/main.go`: `-frame-max-dim` (int, default 0), `-frame-format` (string, default "png"), `-frame-quality` (int, default 3), `-grid-rows` (int, default 0)
- [ ] T003 Validate new flags in `cmd/goblin/main.go`: frame-max-dim ≥ 0; frame-format ∈ {"png","jpg"} (exit 1 otherwise); frame-quality 1–31 (exit 1 otherwise); grid-rows ≥ 0; emit warning to stderr when frame-quality is set with frame-format=png

**Checkpoint**: `go build ./...` passes; `-frame-format xyz` exits 1 with diagnostic;
all four flags visible in `--help` output.

---

## Phase 2: internal/extract — Extension and Dimension Cap

**Purpose**: Thread frame format/quality and dimension cap through extraction.

- [ ] T004 [US1,US3] Update `Extract()` signature in `internal/extract/extract.go` to accept `ext string` (e.g. `"png"` or `"jpg"`) and `quality int` and `maxDim int`; replace all 4 `frame_%05d.png` literals with `fmt.Sprintf("frame_%%05d.%s", ext)` — `extract.go` lines 60, 134, 144, 162, 179
- [ ] T005 [US1] Update first-frame fallback path `frame_00001.png` → `fmt.Sprintf("frame_00001.%s", ext)` in `extract.go` lines 85 and 92
- [ ] T006 [US3] Add JPEG quality flag to ffmpeg frame-extraction invocation: when `ext == "jpg"`, replace `-q:v 2` with `-q:v <quality>`; PNG keeps `-q:v 2` unchanged
- [ ] T007 [US1] Add scale filter to ffmpeg frame-extraction invocation when `maxDim > 0`: append `scale=MAXDIM:MAXDIM:force_original_aspect_ratio=decrease:flags=lanczos` to the `-vf` filtergraph (both the scene-detection path and the first-frame fallback path)
- [ ] T008 [P] [US1,US3] Update `internal/extract/extract_test.go` to cover: JPEG extension in output paths; scale filter presence when maxDim > 0; no upscaling when maxDim > source dimension

**Checkpoint**: `go test ./internal/extract/...` passes; a manual run with
`-frame-format jpg` produces `.jpg` files; `-frame-max-dim 320` on a 1080p
source produces 320px-wide frames.

---

## Phase 3: internal/grid — Extension and Rows

**Purpose**: Thread frame extension and grid row count through grid generation.

- [ ] T009 [US2] Update `Grid()` signature in `internal/grid/grid.go` to accept `ext string` and `rows int`; compute effective rows: `if rows == 0 { rows = cols }`; update tile filter from `tile=COLSxCOLS` to `fmt.Sprintf("tile=%dx%d:padding=2", cols, rows)`
- [ ] T010 [US3] Update `Grid()` frame file filter: `strings.HasSuffix(e.Name(), ".png")` → `strings.HasSuffix(e.Name(), "."+ext)` — grid.go line 27
- [ ] T011 [US3] Update ffmpeg sequential input pattern in `Grid()`: `"frames/frame_%05d.png"` → `fmt.Sprintf("frames/frame_%%05d.%s", ext)` — grid.go line 54
- [ ] T012 [US3] Update post-grid frame path collectors in `Grid()` (line 71) and `SortedFrameNames()` (line 88): `.HasSuffix(".png")` → `.HasSuffix("."+ext)`
- [ ] T013 [P] [US2,US3] Update `internal/grid/grid_test.go` to cover: rectangular pages (non-zero rows ≠ cols); JPEG extension input (frames named `.jpg`); combined dim-cap + JPEG + rectangular grid

**Checkpoint**: `go test ./internal/grid/...` passes; a manual run with
`-grid -grid-cols 8 -grid-rows 3` produces 8×3 grid pages.

---

## Phase 4: main.go Wire-Up and MANIFEST Updates

**Purpose**: Connect new config values to extract/grid calls; update MANIFEST output.

- [ ] T014 [US1,US2,US3] Update Extract() call in `cmd/goblin/main.go` to pass `cfg.FrameFormat`, `cfg.FrameQuality`, `cfg.FrameMaxDim`
- [ ] T015 [US2,US3] Update Grid() call in `cmd/goblin/main.go` to pass `cfg.GridRows` and `cfg.FrameFormat` (ext)
- [ ] T016 [US1,US3] Add `FrameFormat` to `Manifest` struct in `internal/manifest/manifest.go` (always written); add `FrameMaxDim int` (written when > 0); add `GridRows int` to grid section (written when > 0 and grid mode active)
- [ ] T017 [US3] Update `VerifyFramePaths()` in `internal/manifest/manifest.go` to use the frame extension from manifest (not hardcoded `.png`)
- [ ] T018 [US3] Update `test/consumer_validate.go` to accept frame paths with any extension (not hardcoded `.png` suffix check)
- [ ] T019 [P] Update `internal/manifest/manifest_test.go`: assert `frame_format` field present; assert frame paths use configured extension; assert `frame_max_dim` absent when 0, present when > 0

**Checkpoint**: Full pipeline with `-frame-format jpg -grid -grid-cols 6 -grid-rows 3`
exits 0; MANIFEST.json contains the three new fields; consumer_validate exits 0.

---

## Phase 5: Documentation and Polish

- [ ] T020 [P] Update `CLAUDE.md` flag table: add `-frame-max-dim`, `-frame-format`, `-frame-quality`, `-grid-rows` rows
- [ ] T021 [P] Update `USING.md`: add worked examples for Scenario 8 (dimension cap), Scenario 10 (JPEG), Scenario 11 (combined); update grid mode section to document `-grid-rows`
- [ ] T022 Run `goimports -w .` across all modified packages; confirm no formatting drift
- [ ] T023 Run `positronikal-check .` and resolve any findings (expect SPDX header check on any new test files)
- [ ] T024 [P] Run all 11 Quickstart Scenarios (7 from 001 + 4 from 002); confirm each exits with expected code and produces expected output structure

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1**: No dependencies — start immediately
- **Phase 2**: Requires Phase 1 (Config struct with new fields)
- **Phase 3**: Requires Phase 1 (Config struct); independent of Phase 2
- **Phase 4**: Requires Phases 2 and 3 — wire-up depends on updated signatures
- **Phase 5**: Requires Phase 4 — documentation reflects final implementation

### Within Phase 2

```
T004 (Extract signature + extension) → T005 (fallback path)
                                     → T006 (JPEG quality flag)
                                     → T007 (scale filter)
T008 (tests) — parallel after T004–T007
```

### Within Phase 3

```
T009 (Grid signature + tile filter + rows)
T010, T011, T012 — all edit grid.go; can be done sequentially with T009
T013 (tests) — parallel after T009–T012
```

### Parallel Opportunities

- T008 and T013 (test updates in different packages) can run in parallel
- T020 and T021 (CLAUDE.md, USING.md) can run in parallel after Phase 4
- T022 and T023 (formatting/compliance checks) run after all code changes

---

## Task Count Summary

| Phase | Tasks | Notes |
|---|---|---|
| Phase 1 Config | T001–T003 | 3 tasks |
| Phase 2 Extract | T004–T008 | 5 tasks |
| Phase 3 Grid | T009–T013 | 5 tasks |
| Phase 4 Wire-Up | T014–T019 | 6 tasks |
| Phase 5 Polish | T020–T024 | 5 tasks |
| **Total** | | **24 tasks** |
