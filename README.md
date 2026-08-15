# goblin

Goblin is a media analysis pipeline for Claude. It extracts scene-change
keyframes and a timestamped transcript from a local video or audio file,
then writes a structured MANIFEST directory that Claude and editing tools
can consume directly.

## Prerequisites

- [ffprobe / ffmpeg](https://ffmpeg.org/) — frame extraction and audio processing
- [whisper.cpp](https://github.com/ggerganov/whisper.cpp) — speech-to-text
  (`whisper-cli` on PATH, or set `GOBLIN_WHISPER_CMD`)
- [mkvextract](https://mkvtoolnix.download/) (optional) — embedded subtitle
  extraction from MKV files

## Install

```sh
make build
# or
go build -o bin/goblin ./cmd/goblin
```

Pre-built binaries for Windows, Linux, and macOS are attached to each
[GitHub release](https://github.com/hoyt-harness/goblin/releases).

Cross-compile all platforms:
```sh
make build-all
```

## Usage

```sh
# Full pipeline (frames + transcript)
goblin -model /path/to/ggml-large-v3-turbo.bin video.mp4

# Probe only (metadata, no extraction)
goblin -probe-only video.mp4

# Skip transcription
goblin -no-transcript video.mp4

# Use custom whisper binary
GOBLIN_WHISPER_CMD=/usr/local/bin/main goblin -model /path/to/model.bin video.mp4
```

## Output

```
video_goblin/
├── MANIFEST.json    — top-level index with scene list and cross-links
├── probe.json       — technical metadata (codec, resolution, duration)
├── transcript.json  — timestamped transcript segments
└── frames/
    ├── frame_00001.png
    └── ...
```

See `USING.md` for the full flag reference and worked examples, including
a strategy guide for AI agents. The guide is also embedded in the binary:

```sh
goblin -guide
```
