# Research: URL Input Mode

**Feature**: 003-url-input
**Date**: 2026-09-08

---

## Decision: yt-dlp output path retrieval

**Decision**: Use `--print after_move:filepath` to capture the final local path after yt-dlp completes download and muxing.

**Rationale**: yt-dlp may rename or remux files after the initial download (e.g., merging best-video + best-audio streams). The output filename is not predictable from the input URL. `--print after_move:filepath` is the official yt-dlp mechanism for capturing the real final path — it prints to stdout after all post-processing is complete.

**Alternatives considered**: Templated `-o` output path + reconstructing the expected filename — rejected because yt-dlp's extension selection and title sanitization are non-trivial to replicate exactly.

---

## Decision: yt-dlp format string for quality capping

**Decision**: For `-max-height N` (N > 0):
```
bestvideo[height<=N][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=N]+bestaudio/best[height<=N]/best
```
For `-max-height 0` (no cap):
```
bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio/best
```

**Rationale**: The preference cascade tries for MP4 video + M4A audio first (produces a clean MP4 after ffmpeg mux), falls back to any container if that's unavailable, then falls back to the best single-file format. This maximises compatibility with ffprobe/ffmpeg downstream. `720` is the default cap per the spec.

**Alternatives considered**: Simpler `best[height<=N]` — rejected because it excludes separate video+audio streams, which have significantly higher quality at the same resolution for modern YouTube content. `720p` fixed format string — rejected because `-max-height N` is a user-configurable flag.

---

## Decision: Output directory requirement for URL input

**Decision**: When the input is a URL and `-output` is not set, goblin exits with an error requiring `-output` to be specified.

**Rationale**: The default output directory derivation (`<FILE_basename>_goblin/` next to the input file) has no sensible URL equivalent — the filename is unknown before download and there is no "next to" location. Requiring `-output` is honest and avoids creating mystery directories.

**Alternatives considered**: Derive a slug from the URL — rejected as complex and fragile (URL sanitization edge cases). Use `./download_goblin/` in cwd — rejected as too implicit; user should always know where output lands.

---

## Decision: Downloaded file location

**Decision**: Download the video into the output directory (`-output`). The downloaded file lives alongside MANIFEST.json, frames/, and transcript.json.

**Rationale**: Keeps the full run self-contained in one directory. The user can delete the directory to clean up. Consistent with goblin's existing principle that the output directory is the complete artifact of a run.

**Alternatives considered**: Download to a system temp directory, then pipeline processes from there — rejected because it creates two locations to track; also means cleanup requires two steps.

---

## Decision: yt-dlp not on PATH when URL input

**Decision**: Check for yt-dlp in `checkTools()` when the input is a URL, following the same pattern as ffprobe/ffmpeg checks. Use `exec.LookPath` with the resolved binary name (flag override or "yt-dlp"). Exit with error code 2 and a diagnostic naming the tool.

**Rationale**: Consistent with existing tool presence check pattern. Fail fast before any output directory is created.

**Alternatives considered**: Defer check to download stage — rejected because it would create an output directory before failing.

---

## Decision: New internal package vs. code in main

**Decision**: New `internal/download/` package exporting a `Download(url, outDir, ytdlpPath string, maxHeight int) (string, error)` function.

**Rationale**: Constitution Principle IV (composable pipeline, each stage independently invocable). Download is a distinct stage; other stages (probe, extract, transcribe) each have their own internal package. Consistency and testability.

**Alternatives considered**: Implement in `main.go` directly — rejected for testability and consistency. Implement as a function in `cmd/goblin/` — rejected; private to the binary, not independently testable.
