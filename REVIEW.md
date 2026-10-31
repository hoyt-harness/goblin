# Code Review Notes — goblin

## What failure looks like here
A media analysis run that completes without error but produces a MANIFEST.json
with missing, mis-timed, or zero-count scene entries — silent data corruption
from a whisper transcription or yt-dlp interaction that failed mid-run.

## Key invariants the reviewer should check
- yt-dlp URL input mode validates the URL and uses the video ID as the output
  filename (not the title, which can contain Unicode path-breaking characters)
- Whisper subprocess errors are surfaced as MANIFEST entries, not silently dropped
- MANIFEST.json schema is written atomically (temp-write then rename)
- Scene change detection threshold is configurable but has a reasonable default

## Paths to skip
- `var/` — health reports, not source

## Finding standards
- Every finding must cite a specific file and line number
- Cap nits at 3 per review pass; flag the class, not every instance
