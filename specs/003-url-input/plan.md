# Implementation Plan: URL Input Mode

**Branch**: `003-url-input` | **Date**: 2026-09-08 | **Spec**: [spec.md](spec.md)

---

## Summary

Enable goblin to accept a YouTube (or any yt-dlp-supported) URL as its input argument. When a URL is detected, goblin downloads the video into the output directory using yt-dlp, then processes it through the existing pipeline identically to a local file. Adds `-ytdlp-path` and `-max-height` flags. Adds `source_url` to MANIFEST.json for URL-sourced runs. Local-file behavior is unchanged.

---

## Technical Context

**Language/Version**: Go 1.22+ (matches current goblin go.mod)

**Primary Dependencies**: yt-dlp (external binary, PATH-resolved); existing: ffprobe, ffmpeg, whisper-cli

**Storage**: Output directory (user-specified via `-output`, required for URL input)

**Testing**: `go test ./...` via `make test`; new tests in `internal/download/` and `cmd/goblin/main_test.go`

**Target Platform**: Windows 11 (primary); Linux (post-migration target); cross-compiled via `make build-all`

**Project Type**: CLI binary

**Performance Goals**: Download time dominated by yt-dlp + network; goblin processing time unchanged from local-file equivalent

**Constraints**: No breaking changes to existing MANIFEST.json schema for local-file runs; yt-dlp check must occur before output directory creation

**Scale/Scope**: Single-video URLs; playlist URLs explicitly out of scope for this spec

---

## Constitution Check

**Principle I (Analyze, Never Modify)**: PASS. goblin reads the downloaded video; yt-dlp writes it. goblin never opens the downloaded file for writing.

**Principle II (Local by Default)**: PASS. URL input is explicitly opt-in (user must pass a URL). Network access occurs only when a URL is the input argument. Local file processing path is unchanged.

**Principle III (Machine-First Output)**: PASS. `source_url` is a new MANIFEST.json field using the existing JSON structure. No prose output added.

**Principle IV (Composable Pipeline)**: PASS. Download is implemented as a new `internal/download/` package with a single exported function — independently invocable and testable. Existing stages (probe, extract, transcribe) unchanged.

**Principle V (Format-Agnostic Input)**: PASS. The downloaded file is passed to the existing ffprobe/ffmpeg pipeline unchanged. goblin does not inspect the format of the downloaded file itself.

No violations. Complexity Tracking table not required.

---

## Project Structure

### Documentation (this feature)

```
specs/003-url-input/
├── plan.md              ← this file
├── research.md          ← Phase 0 complete
├── data-model.md        ← Phase 1 complete
├── quickstart.md        ← Phase 1 complete
├── contracts/
│   └── cli-flags.md     ← Phase 1 complete
└── tasks.md             ← Phase 2 output (/speckit-tasks)
```

### Source Code Changes

```
internal/download/          ← NEW package
├── download.go             ← Download() function
└── download_test.go        ← format string + error path tests

cmd/goblin/
└── main.go                 ← URL detection, new flags, download stage, SourceURL

internal/manifest/
└── manifest.go             ← Add SourceURL field to Manifest struct
```

No new files in probe/, extract/, transcribe/, grid/ — those stages are unchanged.

---

## Implementation Sequence

### Stage A: `internal/download` package

1. Create `internal/download/download.go`
2. Implement `buildFormatString(maxHeight int) string` — constructs yt-dlp `-f` argument
3. Implement `Download(url, outDir, ytdlpPath string, maxHeight int) (string, error)`:
   - Resolve binary: `ytdlpPath` if set, otherwise `"yt-dlp"`
   - Build format string via `buildFormatString`
   - Construct yt-dlp args: `-f <fmt> -o "<outDir>/%(title)s.%(ext)s" --no-playlist --print after_move:filepath <url>`
   - Run via `exec.Command`, capture stdout for the filepath, stderr for error context
   - On non-zero exit: return `fmt.Errorf("yt-dlp: %s", stderr)`
   - On success: parse stdout for the downloaded filepath, verify it exists, return abs path
4. Create `internal/download/download_test.go`:
   - `TestBuildFormatString_WithCap` — verifies height filter in format string
   - `TestBuildFormatString_NoCap` — verifies no height filter when maxHeight=0
   - `TestDownload_BinaryNotFound` — verifies clean error when binary missing

### Stage B: Manifest struct

5. Add `SourceURL string \`json:"source_url,omitempty"\`` to `Manifest` struct in `internal/manifest/manifest.go`
6. Update `manifest_test.go`: verify `source_url` appears in marshalled JSON when set; verify absent when zero value

### Stage C: `cmd/goblin/main.go` wiring

7. Add `YtDlpPath string` and `MaxHeight int` to `Config` struct
8. Register `-ytdlp-path` and `-max-height` flags (default `MaxHeight = 720`)
9. After positional arg parse: set `isURL := strings.HasPrefix(inputFile, "http://") || strings.HasPrefix(inputFile, "https://")`
10. When `isURL`:
    - If `cfg.Output == ""`: print error `-output is required when input is a URL`, return 1
    - Skip `os.Stat` check
    - Add yt-dlp to `checkTools` list (binary name = `cfg.YtDlpPath` or `"yt-dlp"`)
11. After output dir setup, when `isURL`: call `download.Download(inputFile, cfg.Output, cfg.YtDlpPath, cfg.MaxHeight)`:
    - On error: print `goblin: error: download failed: <err>`, return 2
    - On success: reassign `inputFile` and `absInput` to the downloaded file path
12. Set `m.SourceURL = inputFile` (original URL) when `isURL`, before writing manifest
13. Update `flag.Usage` to show `FILE|URL` instead of `FILE`
14. Update `main_test.go`: add `TestURLDetection` (unit test, no network)

### Stage D: `make test` and compliance

15. Run `make test` — all existing tests must pass; new tests must pass
16. Run `positronikal-check` — expect clean
17. Build for all platforms: `make build-all`

---

## USING.md Update

Add a "URL Input" section after the existing flag reference documenting:
- `goblin "https://..." -output <dir>` as the invocation
- `-max-height` and `-ytdlp-path` flags
- Requirement that `-output` is specified
- Note on yt-dlp maintenance cadence (updates when YouTube changes internal endpoints)
