# Tasks: goblin Output Control (002)

**Branch**: `002-output-control` | **Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

**Format**: `[ID] [P?] [Story?] Description — file path`
- **[P]**: Parallelizable — different files, no incomplete dependencies
- **[US#]**: Maps to User Story # from spec.md

---

## Phase 1: Config and Flag Parsing

**Purpose**: Add new flags to `Config` struct and wire up validation. All later
phases depend on this.

- [ ] T001 Add `FrameMaxDim int` and `GridRows int` to `Config` struct in `cmd/goblin/main.go`
- [ ] T002 Wire new flags in `cmd/goblin/main.go`: `-frame-max-dim` (int, default 0), `-grid-rows` (int, default 0)
- [ ] T003 Validate new flags in `cmd/goblin/main.go`: frame-max-dim ≥ 0 (exit 1 otherwise); grid-rows ≥ 0 (exit 1 otherwise)

**Checkpoint**: `go build ./...` passes; `-frame-max-dim -1` exits 1 with
diagnostic; both flags visible in `--help` output.

---

## Phase 2: internal/extract — Dimension Cap

**Purpose**: Add scale filter to frame extraction when `-frame-max-dim` is set.

- [ ] T004 [US1] Update `Extract()` signature in `internal/extract/extract.go` to accept `maxDim int`
- [ ] T005 [US1] Add scale filter to the scene-detection ffmpeg invocation when `maxDim > 0`: append `,scale=MAXDIM:MAXDIM:force_original_aspect_ratio=decrease:flags=lanczos` to the `-vf` filtergraph
- [ ] T006 [US1] Add scale filter to the first-frame fallback ffmpeg invocation when `maxDim > 0`: add `-vf scale=MAXDIM:MAXDIM:force_original_aspect_ratio=decrease:flags=lanczos`
- [ ] T007 [P] [US1] Update `internal/extract/extract_test.go`: add test asserting scale filter is present in ffmpeg args when maxDim > 0; add test asserting no scale filter when maxDim == 0; add test asserting no upscaling when maxDim exceeds source dimension

**Checkpoint**: `go test ./internal/extract/...` passes; a manual run with
`-frame-max-dim 320` on a 1080p source produces 320px-wide frames verified
by ffprobe.

---

## Phase 3: internal/grid — Rows Parameter

**Purpose**: Thread grid row count through grid generation.

- [ ] T008 [US2] Update `Grid()` signature in `internal/grid/grid.go` to accept `rows int`; compute effective rows: `if rows == 0 { rows = cols }`; update tile filter from `tile=COLSxCOLS` to `fmt.Sprintf("tile=%dx%d:padding=2", cols, rows)`
- [ ] T009 [P] [US2] Update `internal/grid/grid_test.go` to cover: rectangular pages (rows ≠ cols); backward compat (rows == 0 → square pages)

**Checkpoint**: `go test ./internal/grid/...` passes; a manual run with
`-grid -grid-cols 8 -grid-rows 3` produces 8×3 grid pages.

---

## Phase 4: main.go Wire-Up and MANIFEST Updates

**Purpose**: Connect new config values to extract/grid calls; update MANIFEST output.

- [ ] T010 [US1,US2] Update `Extract()` call in `cmd/goblin/main.go` to pass `cfg.FrameMaxDim`
- [ ] T011 [US2] Update `Grid()` call in `cmd/goblin/main.go` to pass `cfg.GridRows`
- [ ] T012 [US2] Update the page-size calculation in grid→scene assignment in `cmd/goblin/main.go` to use effective rows (same `if rows == 0 { rows = cols }` logic as grid.go)
- [ ] T013 [US1] Add `FrameMaxDim int` to `Manifest` struct in `internal/manifest/manifest.go`; populate in `main.go` only when `cfg.FrameMaxDim > 0`; add `omitempty` json tag
- [ ] T014 [US2] Add `GridRows int` to `Manifest` struct in `internal/manifest/manifest.go`; populate in `main.go` only when `cfg.Grid && cfg.GridRows > 0`; add `omitempty` json tag
- [ ] T015 [P] Update `internal/manifest/manifest_test.go`: assert `frame_max_dim` absent when 0, present when > 0; assert `grid_rows` absent when 0 or square, present when explicitly set

**Checkpoint**: Full pipeline with `-frame-max-dim 1280 -grid -grid-cols 6 -grid-rows 3`
exits 0; MANIFEST.json contains both new fields; consumer_validate exits 0.

---

## Phase 5: Documentation and Polish

- [ ] T016 [P] Update `CLAUDE.md` flag table: add `-frame-max-dim` and `-grid-rows` rows
- [ ] T017 [P] Update `USING.md`: add worked examples for Scenario 8 (dimension cap) and Scenario 9 (rectangular grid); update grid mode section to document `-grid-rows`; add Scenario 10 (combined)
- [ ] T018 Run `goimports -w .` across all modified packages; confirm no formatting drift
- [ ] T019 Run `positronikal-check .` and resolve any findings
- [ ] T020 [P] Run all 10 Quickstart Scenarios (7 from 001 + 3 from 002); confirm each exits with expected code and produces expected output structure

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1**: No dependencies — start immediately
- **Phase 2**: Requires Phase 1 (Config struct with FrameMaxDim)
- **Phase 3**: Requires Phase 1 (Config struct with GridRows); independent of Phase 2
- **Phase 4**: Requires Phases 2 and 3 — wire-up depends on updated signatures
- **Phase 5**: Requires Phase 4 — documentation reflects final implementation

### Within Phase 2

```
T004 (Extract signature) → T005 (scale filter, scene path)
                         → T006 (scale filter, fallback path)
T007 (tests) — parallel after T004–T006
```

### Within Phase 3

```
T008 (Grid signature + rows)
T009 (tests) — parallel after T008
```

### Parallel Opportunities

- T007 and T009 (test updates in different packages) can run in parallel after their respective Phase 2/3 chains
- T016 and T017 (CLAUDE.md, USING.md) can run in parallel after Phase 4
- T018 and T019 (formatting/compliance checks) run after all code changes

---

## Task Count Summary

| Phase | Tasks | Notes |
|---|---|---|
| Phase 1 Config | T001–T003 | 3 tasks |
| Phase 2 Extract | T004–T007 | 4 tasks |
| Phase 3 Grid | T008–T009 | 2 tasks |
| Phase 4 Wire-Up | T010–T015 | 6 tasks |
| Phase 5 Polish | T016–T020 | 5 tasks |
| **Total** | | **20 tasks** |
