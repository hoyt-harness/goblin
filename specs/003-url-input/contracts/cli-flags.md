# CLI Contract: URL Input Mode flags

**Feature**: 003-url-input
**Date**: 2026-09-08

---

## New flags

```
-ytdlp-path string
    Path to yt-dlp binary (default: PATH lookup for "yt-dlp")

-max-height int
    Cap video height in pixels for download; 0 = best available (default: 720)
```

## Changed behaviour: positional FILE argument

The positional FILE argument now also accepts an `http://` or `https://` URL.

```
goblin [flags] FILE|URL
```

When a URL is given:
- `-output` is required (error if absent).
- yt-dlp must be on PATH or specified via `-ytdlp-path`.
- The video is downloaded into the output directory before the pipeline runs.
- All existing flags (`-no-frames`, `-no-transcript`, `-grid`, `-frame-max-dim`, etc.) apply to the downloaded video identically to a local file.

## MANIFEST.json contract addition

```json
{
  "source_url": "https://www.youtube.com/watch?v=EXAMPLE"
}
```

- Field is present and non-empty only when input was a URL.
- Field is absent from MANIFEST.json for local-file runs (no breaking change).
- Value is the original URL string as passed by the user, verbatim.

## Error cases

| Condition | Exit code | Stderr message |
|---|---|---|
| URL input, yt-dlp not on PATH | 2 | `goblin: error: yt-dlp not found on PATH\n  Install from https://github.com/yt-dlp/yt-dlp` |
| URL input, `-output` not set | 1 | `goblin: error: -output is required when input is a URL` |
| yt-dlp download fails | 2 | `goblin: error: download failed: <yt-dlp stderr>` |
| Downloaded file not found after yt-dlp | 2 | `goblin: error: download produced no output file` |
