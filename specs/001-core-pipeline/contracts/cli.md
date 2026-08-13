# Contract: goblin CLI Interface

**Version**: 1.0.0
**Binary**: `goblin`

---

## Invocation

```
goblin [flags] FILE
```

`FILE` is the only positional argument. It is required. Invoking goblin with
no arguments, or with `--help` / `-h`, prints usage to stdout and exits 0.

---

## Flags

| Flag | Type | Default | Env override | Description |
|---|---|---|---|---|
| `-output DIR` | string | `<FILE_basename>_goblin/` next to FILE | — | Output directory path |
| `-threshold N` | float | `0.4` | — | Scene detection sensitivity (0.0 = every frame, 1.0 = no changes) |
| `-probe-only` | bool | false | — | Run probe stage only |
| `-no-frames` | bool | false | — | Skip frame extraction |
| `-no-transcript` | bool | false | — | Skip transcription |
| `-prefer-whisper` | bool | false | — | Run whisper even when embedded subtitles exist |
| `-overwrite` | bool | false | — | Overwrite existing output directory |
| `-whisper-cmd CMD` | string | `whisper-cli` | `GOBLIN_WHISPER_CMD` | Whisper binary name or full path |
| `-model PATH` | string | — | `GOBLIN_WHISPER_MODEL` | Whisper model file path (required for transcription) |
| `-grid` | bool | false | — | Produce contact sheet grid images |
| `-grid-cols N` | int | `4` | — | Columns per grid sheet |
| `-frame-warn N` | int | `50` | — | Warn if output frame count exceeds N |
| `-threads N` | int | `0` | — | Thread count hint for ffmpeg and whisper (0 = let tools choose) |
| `-quiet` | bool | false | — | Suppress progress lines; print errors only |
| `-version` | bool | false | — | Print version string and exit 0 |

Flag conflicts:
- `-probe-only` overrides `-no-frames` and `-no-transcript` (implied by probe-only)
- `-no-frames -no-transcript` together is valid (probe + manifest only)

---

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success — output directory written |
| `1` | Usage error — bad flags, missing FILE argument |
| `2` | Tool error — required external tool missing or failed |
| `3` | Input error — FILE not found, not readable, or ffprobe cannot parse it |
| `4` | Output error — output directory exists without `-overwrite`, or write failure |

---

## Standard Streams

**stdout**: Progress lines when `-quiet` is false. One line per stage start/end.
Format: `goblin: [stage] message`. Example:
```
goblin: probe  interview.mp4 (01:23:45, h264/aac, 1920x1080)
goblin: extract  42 scenes detected (threshold 0.40)
goblin: transcribe  large-v3-turbo, CUDA device 0
goblin: done  interview_goblin/ (42 frames, 387 segments)
```

**stderr**: Errors and warnings only. Always written regardless of `-quiet`.
Format: `goblin: error: message` or `goblin: warning: message`.

**No JSON to stdout.** All structured output goes to files in the output
directory. stdout is for human-readable progress only.

---

## Environment Variables

| Variable | Equivalent flag | Description |
|---|---|---|
| `GOBLIN_WHISPER_CMD` | `-whisper-cmd` | Whisper binary; flag takes precedence |
| `GOBLIN_WHISPER_MODEL` | `-model` | Model path; flag takes precedence |

---

## Tool Dependency Checks

At startup (before reading FILE), goblin verifies:

1. `ffprobe` is on PATH
2. `ffmpeg` is on PATH (skipped if `-probe-only`)
3. `whisper-cli` (or `GOBLIN_WHISPER_CMD` value) is on PATH (skipped if
   `-no-transcript` or `-probe-only`)

On failure, exits 2 with a message naming the missing tool and a hint:
```
goblin: error: ffprobe not found on PATH
  Install the FFmpeg suite: https://ffmpeg.org/download.html
```
