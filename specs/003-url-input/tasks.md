# Tasks: URL Input Mode

**Feature**: 003-url-input | **Date**: 2026-09-08
**Input**: specs/003-url-input/plan.md, spec.md, data-model.md, contracts/, research.md

---

## Phase 1: Setup

No project structure changes required — Go packages are created by writing files. No tasks.

---

## Phase 2: Foundational (Manifest struct change)

**Purpose**: Add `SourceURL` to the Manifest struct. Required before US4 can set the field. Safe to do first — `omitempty` means no change to existing MANIFEST.json output.

- [ ] T001 Add `SourceURL string \`json:"source_url,omitempty"\`` field to Manifest struct in `internal/manifest/manifest.go`
- [ ] T002 [P] Add source_url test cases to `internal/manifest/manifest_test.go`: verify field present when non-empty, absent when zero value

**Checkpoint**: Manifest struct updated; existing tests still pass; `source_url` appears in JSON only when set.

---

## Phase 3: User Story 1 — Full Pipeline from URL (Priority: P1) 🎯 MVP

**Goal**: `goblin <youtube-url> -output <dir>` runs the full pipeline and produces a complete MANIFEST.

**Independent Test**: Pass a public YouTube URL with `-output`. Verify MANIFEST directory produced with frames/, transcript.json, MANIFEST.json.

- [ ] T003 [US1] Create `internal/download/download.go` with `buildFormatString(maxHeight int) string` and `Download(url, outDir, ytdlpPath string, maxHeight int) (string, error)` — default maxHeight 720; use `--print after_move:filepath` to capture output path; wrap yt-dlp stderr on non-zero exit
- [ ] T004 [P] [US1] Create `internal/download/download_test.go` with `TestBuildFormatString_Default` — verifies height<=720 filter appears in format string when maxHeight=720
- [ ] T005 [US1] Add `isURL` detection (`strings.HasPrefix(inputFile, "http://") || strings.HasPrefix(inputFile, "https://")`) and `-output` validation (error + exit 1 if `cfg.Output == ""` when `isURL`) to `cmd/goblin/main.go`
- [ ] T006 [US1] Add yt-dlp to `checkTools()` in `cmd/goblin/main.go` when `isURL`: `exec.LookPath("yt-dlp")` with hint `Install from https://github.com/yt-dlp/yt-dlp`; skip this check when input is a local file
- [ ] T007 [US1] Insert download stage into `run()` in `cmd/goblin/main.go`: after `setupOutputDir`, before probe — call `download.Download(inputFile, cfg.Output, "", cfg.MaxHeight)`; on error print `goblin: error: download failed: <err>` and return 2; reassign `inputFile` and `absInput` to downloaded path
- [ ] T008 [US1] Update `flag.Usage` in `cmd/goblin/main.go` to show `FILE|URL` instead of `FILE`
- [ ] T009 [P] [US1] Add `TestURLDetection` to `cmd/goblin/main_test.go` — unit test (no network): verify `strings.HasPrefix` logic correctly identifies URL vs file path inputs; at minimum test `https://`, `http://`, local path, and UNC path variants

**Checkpoint**: `goblin <url> -output <dir>` downloads the video and produces a complete MANIFEST. Local file runs unchanged.

---

## Phase 4: User Story 2 — Quality-Capped Download (Priority: P2)

**Goal**: `-max-height N` controls download resolution; default is 720p.

**Independent Test**: Run goblin on a URL with and without `-max-height`; verify downloaded file resolution via ffprobe matches cap.

- [ ] T010 [US2] Add `MaxHeight int` to `Config` struct, register `-max-height` flag (default 720) in `cmd/goblin/main.go`; wire `cfg.MaxHeight` to `download.Download` call replacing the hardcoded `720` from T007
- [ ] T011 [P] [US2] Extend `internal/download/download_test.go`: add `TestBuildFormatString_WithCap` (height<=N appears for any positive N) and `TestBuildFormatString_NoCap` (no height filter when maxHeight=0)

**Checkpoint**: `-max-height 1080` downloads at 1080p; `-max-height 0` removes cap; default 720 unchanged.

---

## Phase 5: User Story 3 — Clear Diagnostic When yt-dlp Absent (Priority: P3)

**Goal**: Missing yt-dlp gives a named diagnostic immediately; `-ytdlp-path` overrides PATH lookup.

**Independent Test**: Remove yt-dlp from PATH; pass a URL; verify exit non-zero with message naming yt-dlp before any output directory is created.

- [ ] T012 [US3] Add `YtDlpPath string` to `Config`, register `-ytdlp-path` flag in `cmd/goblin/main.go`; update `checkTools()` to resolve binary as `cfg.YtDlpPath` when non-empty, else `"yt-dlp"`; wire `cfg.YtDlpPath` to `download.Download` call replacing the `""` from T007
- [ ] T013 [P] [US3] Add `TestDownload_BinaryNotFound` to `internal/download/download_test.go` — call `Download` with an invalid binary path; verify error returned is non-nil and contains a useful message

**Checkpoint**: `goblin <url>` without yt-dlp on PATH exits immediately with named diagnostic; `-ytdlp-path /full/path/yt-dlp.exe` overrides PATH lookup and succeeds.

---

## Phase 6: User Story 4 — MANIFEST Records Source URL (Priority: P3)

**Goal**: `MANIFEST.json` contains `"source_url"` when input was a URL; field absent for local-file runs.

**Independent Test**: Run goblin on a URL; `cat MANIFEST.json | python3 -c "import json,sys; print(json.load(sys.stdin).get('source_url'))"` returns the original URL.

- [ ] T014 [US4] Set `m.SourceURL = inputFile` (the original URL string) when `isURL` in `cmd/goblin/main.go`, before `manifest.WriteManifest`; depends on T001 (Manifest struct field)
- [ ] T015 [P] [US4] Add `TestManifestSourceURL` to `internal/manifest/manifest_test.go`: verify `source_url` appears in marshalled JSON when `SourceURL` is set; verify field absent when zero value (regression on local-file behavior)

**Checkpoint**: MANIFEST.json contains `source_url` for URL runs; `source_url` absent for local-file runs.

---

## Phase 7: Polish & Cross-Cutting

- [ ] T016 [P] Add "URL Input" section to `USING.md`: invocation pattern, `-max-height` and `-ytdlp-path` flags, `-output` requirement, yt-dlp maintenance cadence note (updates required when YouTube changes endpoints)
- [ ] T017 Run `make test` from `D:/Engineering/goblin` — all existing + new tests must pass; fix any failures before proceeding
- [ ] T018 Run `positronikal-check` — must be clean; address any findings
- [ ] T019 Run `make build-all` — all 4 platform targets must compile without error

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 2 (Foundational)**: No dependencies — start immediately
- **Phase 3 (US1)**: No dependency on Phase 2 — can start in parallel with Phase 2
- **Phase 4 (US2)**: Depends on T003 (download.go exists), T010 extends it
- **Phase 5 (US3)**: Depends on T006 (checkTools pattern established), T012 extends it
- **Phase 6 (US4)**: Depends on T001 (Manifest struct field) and T007 (isURL flag in run())
- **Phase 7 (Polish)**: Depends on all prior phases complete

### Within-Phase Dependencies

| Task | Depends on |
|---|---|
| T005 | — |
| T006 | T005 (isURL variable must exist) |
| T007 | T003 (download package), T005 (isURL), T006 (checkTools yt-dlp check) |
| T008 | T005 |
| T010 | T007 (replaces hardcoded value) |
| T012 | T006 (checkTools pattern) |
| T014 | T001 (struct field), T007 (isURL flag) |
| T017 | T001–T016 all complete |
| T018 | T017 |
| T019 | T018 |

### Parallel Opportunities

Tasks marked [P] touch different files and have no cross-dependencies:
- T002, T004, T009 can run in parallel (different test files)
- T011, T013, T015 can run in parallel (different test files)
- T016 (USING.md) can run any time after T007

---

## Implementation Strategy

### MVP (US1 only — Phases 2+3)

1. Complete T001–T002 (Manifest struct)
2. Complete T003–T009 (US1 core)
3. **Validate**: run `goblin <url> -output <dir>` and verify full MANIFEST produced
4. Then add US2 → US3 → US4 incrementally

### Full delivery order

Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6 → Phase 7

US2–US4 are independent of each other once US1 is done; they can be done in any order.
