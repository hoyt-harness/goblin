# Tasks: goblin Core Pipeline

**Branch**: `001-core-pipeline` | **Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

**Format**: `[ID] [P?] [Story?] Description — file path`
- **[P]**: Parallelizable — different files, no incomplete dependencies
- **[US#]**: Maps to User Story # from spec.md

---

## Phase 1: Setup

**Purpose**: Project initialization, module structure, tooling.

- [x] T001 Initialize Go module — `go.mod` (`module github.com/hoyt-harness/goblin`, `go 1.26.5`)
- [x] T002 Create directory structure per plan.md — `cmd/goblin/`, `internal/probe/`, `internal/extract/`, `internal/transcribe/`, `internal/manifest/`, `internal/grid/`, `test/fixtures/`, `docs/`, `bin/`, `var/`
- [x] T003 [P] Write `.gitignore` — binary artifacts (`bin/`), temp files (`.audio.wav`, `.frame-meta`), `var/`, OS noise
- [x] T004 [P] Write `.gitattributes` — `*.go text eol=lf`, `*.sh text eol=lf`, `/hooks/* text eol=lf`
- [x] T005 [P] Update `hooks/ci-check.sh` for Go — `go build ./...`, `go test ./...`, `goimports -l`, `positronikal-check .`
- [x] T006 [P] Scaffold `README.md` — name, one-line description, prerequisites (ffprobe, ffmpeg, whisper-cli), install, minimal usage example
- [x] T007 [P] Write test fixture generator in `test/fixtures/gen.sh` — uses `ffmpeg -lavfi` to produce a synthetic 30s video (`test.mp4`) with 3 scene changes and spoken-content tone; checked in script, generated file gitignored

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure all stages depend on. No user story work begins until this phase is complete.

- [x] T008 Define `Config` struct in `cmd/goblin/main.go` — all flags (output, threshold, probe-only, no-frames, no-transcript, prefer-whisper, overwrite, whisper-cmd, model, grid, grid-cols, frame-warn, threads, quiet, version) plus env var fields
- [x] T009 Implement flag parsing in `cmd/goblin/main.go` — stdlib `flag` package; resolve `GOBLIN_WHISPER_CMD` and `GOBLIN_WHISPER_MODEL` env vars with flag-overrides-env precedence; `-help` prints usage to stdout, exits 0; `-version` prints version string, exits 0
- [x] T010 Implement tool presence checker in `cmd/goblin/main.go` — `exec.LookPath` for `ffprobe` and `ffmpeg` (always); `whisper-cli` / `GOBLIN_WHISPER_CMD` (unless `-no-transcript` or `-probe-only`); exits 2 with named diagnostic and hint on missing tool
- [x] T011 Implement output directory manager in `cmd/goblin/main.go` — default naming (`<basename>_goblin/` next to input), existence check (exit 4 without `-overwrite`), `os.MkdirAll` on success, `frames/` subdirectory creation
- [x] T012 Implement positional argument validation in `cmd/goblin/main.go` — exactly one FILE arg required (exit 1 on zero or >1); `os.Stat` to confirm FILE exists and is readable (exit 3 on failure)

**Checkpoint**: `go build ./...` passes; `goblin --version` and `goblin --help` work; missing-tool exit tested manually.

---

## Phase 3: User Story 1 — Full Visual and Audio Analysis (P1) 🎯 MVP

**Goal**: `goblin -model MODEL video.mp4` produces a complete MANIFEST directory from which Claude can identify visual content and quote spoken lines with timestamps.

**Independent Test**: Quickstart Scenario 1 — run goblin on a test video, confirm MANIFEST.json + frames/ + transcript.json present with valid content.

### internal/probe

- [x] T013 [P] [US1] Define `ProbeResult` and `SubtitleTrack` types in `internal/probe/probe.go` — fields per data-model.md; exported types
- [x] T014 [US1] Implement `Probe(path string) (*ProbeResult, error)` in `internal/probe/probe.go` — `exec.Command("ffprobe", "-print_format", "json", "-show_streams", "-show_format", path)`; parse JSON into `ProbeResult`; populate all fields including subtitle track list
- [x] T015 [P] [US1] Write `internal/probe/probe_test.go` — unit tests using fixture: correct codec, duration within 0.5s, subtitle track list, error on non-existent file

### internal/extract

- [x] T016 [P] [US1] Define `Scene` and `Frame` types in `internal/extract/extract.go` — fields per data-model.md
- [x] T017 [US1] Implement `Extract(input string, outDir string, threshold float64) ([]Scene, []Frame, error)` in `internal/extract/extract.go` — invokes `ffmpeg -i input -vf "select=gt(scene\,THRESHOLD),metadata=print:file=OUTDIR/.frame-meta" -vsync vfr -q:v 2 OUTDIR/frames/frame_%05d.png`; captures stderr for error reporting; returns on non-zero exit
- [x] T018 [US1] Implement `.frame-meta` parser in `internal/extract/extract.go` — reads key=value blocks (blank-line separated), extracts `pts_time` per frame, associates with `frame_%05d.png` filenames, deletes `.frame-meta` on success
- [x] T019 [US1] Implement "at least one frame" invariant in `internal/extract/extract.go` — if ffmpeg produces zero output frames (no scene changes), extract frame at t=0 via `ffmpeg -i input -vframes 1 OUTDIR/frames/frame_00001.png`; populate Scene covering full file duration; set `SceneScore: 0`
- [x] T020 [P] [US1] Write `internal/extract/extract_test.go` — unit tests using fixture: ≥1 frame produced, frame files exist on disk, scenes contiguous, `end_s` of last scene ≤ duration + 0.5s

### internal/transcribe

- [x] T021 [P] [US1] Define `TranscriptSegment` type in `internal/transcribe/transcribe.go`
- [x] T022 [US1] Implement audio extraction in `internal/transcribe/transcribe.go` — `ffmpeg -i input -vn -acodec pcm_s16le -ar 16000 -ac 1 OUTDIR/.audio.wav`; defer delete of `.audio.wav`
- [x] T023 [US1] Implement whisper subprocess invocation in `internal/transcribe/transcribe.go` — `whisper-cli -m MODEL -f .audio.wav -oj -ojf -of OUTDIR/transcript --no-prints`; captures stderr; returns on non-zero exit
- [x] T024 [US1] Implement whisper JSON normalization in `internal/transcribe/transcribe.go` — parse whisper's `transcript.json` output; map `segments[].start` / `.end` / `.text` → `[]TranscriptSegment`; trim whitespace from text; set `Source: "whisper"` in output
- [x] T025 [P] [US1] Write `internal/transcribe/transcribe_test.go` — unit tests: audio extraction produces `.wav`, normalization maps whisper segments correctly, temp file deleted after run

### internal/manifest

- [x] T026 [P] [US1] Define `Manifest` and `SceneRef` types in `internal/manifest/manifest.go` — fields per data-model.md; `SchemaVersion: "1"` constant
- [x] T027 [US1] Implement scene↔transcript cross-linking in `internal/manifest/manifest.go` — for each `Scene`, collect indices of `TranscriptSegments` where `seg.StartS < scene.EndS && seg.EndS > scene.StartS`; populate `SceneRef.TranscriptSegments`
- [x] T028 [US1] Implement `WriteManifest(outDir string, m *Manifest) error` in `internal/manifest/manifest.go` — marshals Manifest to indented JSON; writes `MANIFEST.json`
- [x] T029 [P] [US1] Implement `WriteProbe(outDir string, r *probe.ProbeResult) error` in `internal/manifest/manifest.go` — writes `probe.json`
- [x] T030 [P] [US1] Implement `WriteTranscript(outDir string, segs []transcribe.TranscriptSegment, meta TranscriptMeta) error` in `internal/manifest/manifest.go` — writes `transcript.json` in goblin schema (source, model, language, segments)
- [x] T031 [P] [US1] Write `internal/manifest/manifest_test.go` — unit tests: cross-linking correct for overlapping/non-overlapping segments, all relative paths present in output, schema_version field = "1"

### cmd/goblin/main.go — Full Pipeline Wire-Up

- [x] T032 [US1] Wire full pipeline in `cmd/goblin/main.go` — call probe → extract → transcribe → manifest in sequence; pass results between stages; populate `StagesRun` slice; collect warnings
- [x] T033 [US1] Implement stdout progress lines per CLI contract — `goblin: probe ...`, `goblin: extract ...`, `goblin: transcribe ...`, `goblin: done ...`; respect `-quiet` flag (suppress progress, never suppress errors)
- [x] T034 [US1] Implement stderr error formatting — `goblin: error: MESSAGE` format; all exit-code paths covered

**Checkpoint**: Quickstart Scenario 1 passes. `goblin -model /d/models/whisper-models/ggml-large-v3-turbo.bin test.mp4` exits 0, produces valid MANIFEST directory.

---

## Phase 4: User Story 2 — Technical Media Probe (P2)

**Goal**: `-probe-only` mode exits after probe + probe.json, no frames extracted, no transcription attempted.

**Independent Test**: Quickstart Scenario 2 — output contains only MANIFEST.json and probe.json; stages_run = ["probe"].

- [x] T035 [US2] Implement `-probe-only` stage gate in `cmd/goblin/main.go` — after probe completes, if `-probe-only`: write probe.json + MANIFEST.json with stages_run=["probe"], exit 0; skip extract/transcribe/grid entirely
- [x] T036 [US2] Confirm probe.json is always written regardless of other flags — verify in manifest writer that probe.json write is unconditional

**Checkpoint**: Quickstart Scenario 2 passes. Output directory contains exactly MANIFEST.json and probe.json, nothing else.

---

## Phase 5: User Story 3 — Selective Stage Invocation (P2)

**Goal**: `-no-frames`, `-no-transcript`, and embedded-subtitle preference all work correctly and independently.

**Independent Test**: Quickstart Scenarios 3, 4, and Scenario involving audio-only / video-only input.

- [x] T037 [US3] Implement `-no-frames` gate in `cmd/goblin/main.go` — skip `internal/extract`; `scenes` in MANIFEST is empty array; `frames/` not created; stages_run excludes "extract"
- [x] T038 [US3] Implement `-no-transcript` gate in `cmd/goblin/main.go` — skip `internal/transcribe`; `transcript_path` empty in MANIFEST; transcript.json not written; stages_run excludes "transcribe"
- [x] T039 [US3] Handle audio-only input — if `ProbeResult.HasVideo == false`: skip extract stage automatically; emit `goblin: warning: no video stream, skipping frame extraction`; note in MANIFEST warnings
- [x] T040 [US3] Handle video-only input — if `ProbeResult.HasAudio == false`: skip transcribe stage automatically; emit warning; note in MANIFEST warnings
- [x] T041 [US3] Implement embedded subtitle detection in `internal/transcribe/subtitle.go` — check mkvextract on PATH via `exec.LookPath`; check `ProbeResult.SubtitleTracks` non-empty; return bool `ShouldExtractSubtitles(probe, config)`
- [x] T042 [US3] Implement mkvextract subtitle extraction in `internal/transcribe/subtitle.go` — `mkvextract tracks input INDEX:OUTDIR/.subtitles.srt`; handle SRT parsing → `[]TranscriptSegment`; set `Source: "subtitle"` in transcript meta
- [x] T043 [US3] Implement `-prefer-whisper` override in `cmd/goblin/main.go` — when set, bypass subtitle detection and run whisper regardless
- [x] T044 [P] [US3] Write `internal/transcribe/subtitle_test.go` — unit tests: SRT parsing, segment normalization, source field set correctly

**Checkpoint**: Quickstart Scenarios 3 and 4 pass. Audio-only and video-only inputs handled without error.

---

## Phase 6: User Story 4 — Editing Pipeline Handoff (P3)

**Goal**: MANIFEST.json is sufficient for a downstream editing tool to derive scene cut points and caption strings without transformation or additional parsing.

**Independent Test**: Quickstart Scenario 4 — consumer validation script reads MANIFEST.json and derives cut list + captions using only what MANIFEST references.

- [x] T045 [US4] Write consumer validation script in `test/consumer_validate.go` — reads MANIFEST.json, produces ordered list of `{scene_index, start_s, end_s, frame_path, captions: [text, ...]}` for each scene; exits 0 only if all scene frames exist on disk and all referenced transcript segment indices are in bounds
- [x] T046 [US4] Implement post-write verification pass in `internal/manifest/manifest.go` — after writing MANIFEST.json, verify every relative path referenced in scenes[].frame exists on disk; add warning to MANIFEST if any are missing (should never happen, but detects partial writes)
- [x] T047 [US4] Verify `schema_version: "1"` is present and correct in all JSON output files — add assertion to manifest unit tests

**Checkpoint**: Quickstart Scenario 4 passes. Consumer script exits 0 on full-pipeline output.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Grid mode, edge case hardening, frame-count warning, documentation, full quickstart validation.

- [x] T048 [P] Implement `Grid(framesDir string, outDir string, cols int) ([]string, error)` in `internal/grid/grid.go` — `ffmpeg -framerate 1 -pattern_type glob -i "frames/frame_*.png" -vf "tile=COLSxROWS:padding=2" -frames:v 1 grids/grid_%03d.png`; return list of grid file relative paths
- [x] T049 Integrate grid mode in `cmd/goblin/main.go` — after extract stage when `-grid` set: call `internal/grid.Grid()`; create `grids/` subdir; populate `SceneRef.Grid` field in manifest builder
- [x] T050 [P] Write `internal/grid/grid_test.go` — unit tests: grid produces at least one PNG, MANIFEST grid fields populated
- [x] T051 Implement `-frame-warn` threshold check in `cmd/goblin/main.go` — after extract, if `len(frames) > config.FrameWarn`: write to stderr `goblin: warning: N frames exceed warn threshold T; consider --grid`; add to MANIFEST warnings
- [x] T052 [P] Implement `-threads` flag forwarding — pass `-threads N` to ffmpeg invocations and `-t N` to whisper-cli when N > 0
- [x] T053 Implement overwrite guard integration test — verify exit 4 + helpful message on second run without `-overwrite`; verify `-overwrite` succeeds (Quickstart Scenario 5)
- [x] T054 Implement missing-tool integration test — verify exit 2 within expected time window (Quickstart Scenario 6)
- [x] T055 [P] Run `goimports -w .` across all packages; confirm no formatting drift
- [x] T056 [P] Run `positronikal-check .` and resolve any findings
- [x] T057 [P] Add SPDX license headers to all `.go` files — `SPDX-License-Identifier: GPL-3.0-or-later`
- [x] T058 Update `CLAUDE.md` — build command (`go build -o bin/goblin ./cmd/goblin`), flag reference summary, test fixture instructions
- [x] T059 [P] Write `USING.md` — end-user guide: prerequisites, install, five worked examples (full pipeline, probe-only, no-transcript, grid mode, subtitle extraction)
- [x] T060 Run all Quickstart Scenarios 1–7 from `quickstart.md`; confirm each exits with expected code and produces expected output structure

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Requires Phase 1 — blocks all user story phases
- **Phase 3 (US1 Full Pipeline)**: Requires Phase 2 — MVP; all later phases depend on it
- **Phase 4 (US2 Probe-Only)**: Requires Phase 3 foundation in main.go — but probe package is already built
- **Phase 5 (US3 Selective)**: Requires Phase 3 — gates are additions to existing dispatch logic
- **Phase 6 (US4 Handoff)**: Requires Phase 3 — cross-linking already implemented; this phase validates it
- **Phase 7 (Polish)**: Requires Phases 3–6 complete

### Within Phase 3

```
T013, T016, T021, T026 [P] — type definitions, all independent
     ↓
T014 (probe impl)     T017 (extract impl)     T022+T023 (transcribe impl)
T015 (probe tests)    T018 (meta parser)       T024 (normalization)
                      T019 (first-frame inv.)  T025 (transcribe tests)
                      T020 (extract tests)
     ↓
T027 (cross-linking)  ← depends on Scene + TranscriptSegment types
T028-T031 (manifest writers + tests)
     ↓
T032 (wire-up main.go)
T033, T034 (progress + error output)
```

### Parallel Opportunities

- All [P]-marked tasks within a phase can run concurrently
- Type-definition tasks (T013, T016, T021, T026) all parallel in Phase 3
- Test files (T015, T020, T025, T031) parallel with each other after their impl tasks complete
- Phase 7 polish tasks (T055–T059) largely parallel

---

## Implementation Strategy

### MVP Scope (Phases 1–3 only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: Full Pipeline (US1)
4. **STOP and VALIDATE**: Run Quickstart Scenario 1 on a real video
5. Tag v0.1.0-alpha and get Claude to consume the output

### Incremental Delivery

- Phase 3 done → `goblin` can analyse any video end-to-end (MVP)
- Phase 4 done → probe-only mode available
- Phase 5 done → selective stages + subtitle extraction
- Phase 6 done → editing pipeline handoff validated by consumer script
- Phase 7 done → grid mode, full hardening, v0.1.0 release

---

## Task Count Summary

| Phase | Tasks | Notes |
|---|---|---|
| Phase 1 Setup | T001–T007 | 7 tasks |
| Phase 2 Foundational | T008–T012 | 5 tasks |
| Phase 3 US1 Full Pipeline | T013–T034 | 22 tasks |
| Phase 4 US2 Probe-Only | T035–T036 | 2 tasks |
| Phase 5 US3 Selective Stages | T037–T044 | 8 tasks |
| Phase 6 US4 Handoff | T045–T047 | 3 tasks |
| Phase 7 Polish | T048–T060 | 13 tasks |
| **Total** | | **60 tasks** |
