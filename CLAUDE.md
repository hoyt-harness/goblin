# CLAUDE.md — goblin

Goblin is a media analysis pipeline that extracts keyframes and timestamped
transcripts from video/audio files and emits a structured MANIFEST directory
for Claude and editing-tool consumption.

Language: Go 1.26.5. Single binary distribution. No runtime Go dependencies
beyond external CLI tools (ffprobe, ffmpeg, whisper.cpp or equivalent).

## Build

```sh
make build
# or
go build -ldflags "-X main.version=$(git describe --tags --abbrev=0)" -o bin/goblin ./cmd/goblin
```

## Test

```sh
go test ./...
bash hooks/ci-check.sh
```

Generate test fixture: `bash test/fixtures/gen.sh`

## Key files

- `cmd/goblin/main.go` — flag parsing, tool checks, pipeline orchestration
- `internal/probe/` — ffprobe subprocess; returns ProbeResult
- `internal/extract/` — ffmpeg scene detection and frame extraction
- `internal/transcribe/` — whisper subprocess and SRT subtitle extraction
- `internal/manifest/` — writes MANIFEST.json, probe.json, transcript.json
- `specs/001-core-pipeline/` — full spec-kit (spec, plan, tasks, contracts)

## Known limitations

- Scene 0 starts at first detected change (not necessarily t=0) — content
  before the first scene change has no representative frame

Vault documentation: `C:\Users\hoyth\Obsidian\Positronikal\03-OPERATIONS\Engineering\`

Standards: `D:\Engineering\PositronikalCodingStandards\standards\`
Coding Bible: `D:\Engineering\_references\CODING_BIBLE.md`
