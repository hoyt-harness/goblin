# Data Model: URL Input Mode

**Feature**: 003-url-input
**Date**: 2026-09-08

---

## Manifest struct changes (`internal/manifest/manifest.go`)

Add one new optional field to `Manifest`:

```go
type Manifest struct {
    // ... existing fields unchanged ...
    SourceURL string `json:"source_url,omitempty"` // NEW: set when input was a network URL
}
```

**Rules**:
- `source_url` is set to the original URL string when goblin was invoked with a URL input.
- `source_url` is omitted (zero value, `omitempty`) when goblin was invoked with a local file path. No change to existing MANIFEST.json output for local-file runs.
- The field is a raw URL string — no normalization, no redaction.

---

## Config struct changes (`cmd/goblin/main.go`)

Add two new fields:

```go
type Config struct {
    // ... existing fields unchanged ...
    YtDlpPath string // -ytdlp-path flag; empty = use PATH lookup
    MaxHeight int    // -max-height flag; 0 = no cap; default 720
}
```

---

## New package: `internal/download`

Single exported function:

```go
// Download fetches url using yt-dlp and writes the video into outDir.
// ytdlpPath is the yt-dlp binary path; empty string → PATH lookup ("yt-dlp").
// maxHeight caps the video height in pixels; 0 = no cap.
// Returns the absolute path of the downloaded file on success.
func Download(url, outDir, ytdlpPath string, maxHeight int) (string, error)
```

**Behaviour**:
- Constructs the yt-dlp format string from `maxHeight`.
- Calls yt-dlp with `-o "<outDir>/%(title)s.%(ext)s"` and `--print after_move:filepath`.
- Captures stdout to extract the final file path.
- On yt-dlp non-zero exit: wraps stderr in the returned error.
- Returns `(absPath, nil)` on success; `("", err)` on failure.

---

## URL detection logic (`cmd/goblin/main.go`)

```go
isURL := strings.HasPrefix(inputFile, "http://") || strings.HasPrefix(inputFile, "https://")
```

Applied after positional argument parsing, before `os.Stat`. When `isURL`:
- Skip `os.Stat` check.
- Require `-output` flag (error if empty).
- Add yt-dlp to tool checks.
- Insert download stage before probe.
- Set `m.SourceURL = inputFile`.
