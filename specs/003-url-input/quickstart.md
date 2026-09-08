# Quickstart: URL Input Mode validation

**Feature**: 003-url-input
**Date**: 2026-09-08

---

## Prerequisites

- goblin built from this branch
- yt-dlp on PATH (or use `-ytdlp-path`)
- ffmpeg, ffprobe on PATH
- whisper-cli on PATH with a model at `D:\models\whisper-models\ggml-large-v3-turbo.bin`
- Internet access

---

## Scenario 1: Full pipeline from a YouTube URL (P1)

```bash
goblin \
  -output D:/_generations/output/comfyui-basics_goblin \
  -model D:/models/whisper-models/ggml-large-v3-turbo.bin \
  -frame-max-dim 1280 \
  -grid \
  "https://www.youtube.com/watch?v=<COMFYUI_TUTORIAL_VIDEO_ID>"
```

**Expected outcome**:
- Progress lines: `goblin: download ...`, `goblin: probe ...`, `goblin: extract ...`, `goblin: transcribe ...`, `goblin: done ...`
- Output directory exists with: `MANIFEST.json`, `probe.json`, `transcript.json`, `frames/`, optionally `grid-*.jpg`
- `MANIFEST.json` contains `"source_url": "https://www.youtube.com/watch?v=<ID>"`
- `"source_url"` field absent from a local-file run (regression check)

---

## Scenario 2: Quality cap (P2)

```bash
goblin -output /tmp/test-720 "https://www.youtube.com/watch?v=<ID>"
# Check downloaded file resolution
ffprobe -v error -select_streams v:0 -show_entries stream=height \
  -of default=noprint_wrappers=1 /tmp/test-720/<downloaded>.mp4
```

**Expected**: `height=720` (or lower if 720p not available)

```bash
goblin -output /tmp/test-1080 -max-height 1080 "https://www.youtube.com/watch?v=<ID>"
ffprobe -v error -select_streams v:0 -show_entries stream=height \
  -of default=noprint_wrappers=1 /tmp/test-1080/<downloaded>.mp4
```

**Expected**: `height=1080` (or lower if 1080p not available)

---

## Scenario 3: Missing yt-dlp diagnostic (P3)

```bash
PATH_BACKUP="$PATH"
export PATH="/usr/bin"    # strip yt-dlp from PATH
goblin -output /tmp/test-missing "https://www.youtube.com/watch?v=<ID>"
echo "Exit: $?"
export PATH="$PATH_BACKUP"
```

**Expected**: exits non-zero; stderr contains `yt-dlp not found on PATH`; no output directory created.

---

## Scenario 4: Missing -output flag (P1 / contract)

```bash
goblin "https://www.youtube.com/watch?v=<ID>"
echo "Exit: $?"
```

**Expected**: exits 1; stderr contains `-output is required when input is a URL`; no download started.

---

## Regression: local file unchanged (P1 / FR-008)

```bash
goblin -output /tmp/local-test D:/path/to/existing.mp4
cat /tmp/local-test/MANIFEST.json | python3 -c "import json,sys; m=json.load(sys.stdin); print('source_url' in m)"
```

**Expected**: `False` — `source_url` absent from local-file MANIFEST.

---

## Test suite

```bash
cd D:/Engineering/goblin
make test
```

**Expected**: all tests pass; new tests in `internal/download/download_test.go` and `cmd/goblin/main_test.go` cover format string construction and URL detection.
