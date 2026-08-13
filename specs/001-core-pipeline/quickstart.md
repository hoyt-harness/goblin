# Quickstart Validation Guide: goblin Core Pipeline

**Date**: 2026-08-10
**Branch**: 001-core-pipeline

This guide describes how to validate that the core pipeline works end-to-end
after implementation. Each scenario maps to a user story from the spec.

---

## Prerequisites

All of the following must be on PATH:

```bash
ffprobe -version    # FFmpeg suite
ffmpeg -version
whisper-cli --version
goblin -version     # after build
```

A whisper model must be available:
```bash
# Verify model path responds
whisper-cli -m /d/models/whisper-models/ggml-large-v3-turbo.bin --version
```

A short test video with spoken audio is needed. A 30–90 second clip with at
least 2 visible scene changes is sufficient. Place it at a known path, e.g.:
`/d/Video/test/goblin-test.mp4`

---

## Scenario 1: Full Pipeline (P1 — Full Analysis)

```bash
goblin -model /d/models/whisper-models/ggml-large-v3-turbo.bin \
       goblin-test.mp4
```

**Expected**:
- Exit code 0
- Directory `goblin-test_goblin/` created containing:
  - `MANIFEST.json` — `schema_version: "1"`, `stages_run: ["probe","extract","transcribe"]`
  - `probe.json` — correct codec, resolution, duration
  - `transcript.json` — at least one segment with non-empty text
  - `frames/frame_00001.png` — valid PNG, non-empty
- Progress lines printed to stdout, no errors to stderr

**Validation check**: Read MANIFEST.json; confirm `scenes` length ≥ 1 and
`transcript_segments` on scene 0 is a non-empty array.

---

## Scenario 2: Probe Only (P2 — Technical Probe)

```bash
goblin -probe-only goblin-test.mp4
```

**Expected**:
- Exit code 0
- `goblin-test_goblin/` contains `MANIFEST.json` and `probe.json` only
- No `frames/` directory, no `transcript.json`
- `MANIFEST.json`: `stages_run: ["probe"]`, `transcript_path: ""`

---

## Scenario 3: No Transcript (P3 — Selective Stages)

```bash
goblin -no-transcript \
       -model /d/models/whisper-models/ggml-large-v3-turbo.bin \
       goblin-test.mp4
```

**Expected**:
- Exit code 0
- `frames/` directory present with at least one PNG
- No `transcript.json`
- `MANIFEST.json`: `stages_run: ["probe","extract"]`, `transcript_path: ""`
- `scenes[*].transcript_segments` all empty arrays

---

## Scenario 4: No Frames (P3 — Selective Stages)

```bash
goblin -no-frames \
       -model /d/models/whisper-models/ggml-large-v3-turbo.bin \
       goblin-test.mp4
```

**Expected**:
- Exit code 0
- `transcript.json` present with segments
- No `frames/` directory
- `MANIFEST.json`: `stages_run: ["probe","transcribe"]`

---

## Scenario 5: Overwrite Guard (Edge Case)

```bash
# First run succeeds
goblin -no-transcript goblin-test.mp4

# Second run without -overwrite must fail
goblin -no-transcript goblin-test.mp4
```

**Expected second run**:
- Exit code 4
- stderr: `goblin: error: output directory already exists: goblin-test_goblin/`
- Hint mentioning `-overwrite`

```bash
# With -overwrite, second run succeeds
goblin -no-transcript -overwrite goblin-test.mp4
```

---

## Scenario 6: Missing Tool (Edge Case)

Temporarily rename `ffprobe.exe` or remove it from PATH, then:

```bash
goblin goblin-test.mp4
```

**Expected**:
- Exit code 2 within 1 second
- stderr: `goblin: error: ffprobe not found on PATH`
- Hint line present

---

## Scenario 7: Grid Mode (P1 extension)

Use a longer video (> 8 scenes recommended):

```bash
goblin -grid -grid-cols 4 \
       -model /d/models/whisper-models/ggml-large-v3-turbo.bin \
       goblin-test.mp4
```

**Expected**:
- `grids/` directory present with at least one `grid_001.png`
- `MANIFEST.json`: `grid_mode: true`, `grid_cols: 4`
- Each `SceneEntry.grid` references the grid sheet it appears in

---

## Success Criteria Traceability

| Criterion | Validated by |
|---|---|
| SC-001: Claude identifies visual/audio from output | Scenario 1 (manual review of frames + transcript) |
| SC-002: No outbound network connections | Scenario 1 (verify with network monitor or air-gap) |
| SC-003: Editing tool can derive cut points | Scenario 1 (parse MANIFEST.json, confirm scene times + frame paths) |
| SC-004: Missing tool exits within 1 second | Scenario 6 |
| SC-005: Source file unchanged | Scenario 1 (hash source before/after) |
| SC-006: Works on any ffprobe-readable file | Additional: test with MKV, MOV, AVI variants |
