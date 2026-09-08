# Feature Specification: URL Input Mode

**Feature Branch**: `003-url-input`

**Created**: 2026-09-08

**Status**: Draft

**Input**: Enable goblin to accept a YouTube (or other yt-dlp-supported) URL as its input argument, download the video locally, and process it through the existing pipeline identically to a local file. The downloaded video is retained after processing. Network access is opt-in and clearly flagged per Constitution Principle II.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Process a YouTube Tutorial by URL (Priority: P1)

A user wants to run goblin's full pipeline (frames + transcript + MANIFEST) on a YouTube tutorial video without manually downloading it first. They pass the video URL as the input argument; goblin downloads the video and produces the same output it would for a local file.

**Why this priority**: This is the entire feature. All other stories are variants or safeguards around this core capability.

**Independent Test**: Pass a public YouTube URL as goblin's input argument. Verify a MANIFEST directory is produced with frames/, transcript.json, and MANIFEST.json — identical structure to a local-file run.

**Acceptance Scenarios**:

1. **Given** yt-dlp is on PATH and a public YouTube URL is passed as the input argument, **When** goblin runs to completion, **Then** a MANIFEST directory exists containing MANIFEST.json, frames/, and transcript.json with content derived from the video.
2. **Given** a YouTube URL with a video longer than 60 minutes, **When** goblin runs, **Then** the full video is downloaded, processed, and the MANIFEST covers the complete duration.
3. **Given** the same URL is run twice, **When** the second run targets the same output directory, **Then** goblin re-downloads (or uses the existing file if present in the output dir) and overwrites the prior MANIFEST when `-overwrite` is set.

---

### User Story 2 — Quality-Capped Download (Priority: P2)

A user wants to control the video resolution downloaded to manage storage and download time. goblin defaults to a capped resolution sufficient for frame extraction quality, but the cap is overridable.

**Why this priority**: Tutorial videos at full 1080p are 2–3× larger than at 720p with no meaningful benefit for scene-change keyframe extraction. The default cap reduces storage use and download time for a common workflow (processing many tutorial videos).

**Independent Test**: Run goblin on a URL that has multiple quality variants. Verify the downloaded file's resolution matches the cap without any additional flags.

**Acceptance Scenarios**:

1. **Given** a YouTube URL with 1080p and 720p variants available and no quality flags set, **When** goblin downloads the video, **Then** the downloaded file is at most 720p.
2. **Given** `-max-height 1080` is passed, **When** goblin downloads, **Then** the downloaded file is at most 1080p.
3. **Given** `-max-height 0` is passed, **When** goblin downloads, **Then** yt-dlp selects the best available quality.

---

### User Story 3 — Clear Failure When yt-dlp Is Absent (Priority: P3)

A user runs goblin with a URL on a machine where yt-dlp is not installed. goblin detects this at startup and exits with a diagnostic that names the missing tool and how to resolve it — the same pattern as ffprobe and ffmpeg.

**Why this priority**: A silent failure or a cryptic exec error would leave the user with no path forward. A clear diagnostic is the minimum bar for a tool that introduces a new external dependency.

**Independent Test**: Remove yt-dlp from PATH and pass a URL. Verify goblin exits immediately with a non-zero code and a message that names yt-dlp as the missing dependency.

**Acceptance Scenarios**:

1. **Given** yt-dlp is not on PATH and a URL is the input argument, **When** goblin starts, **Then** it exits with a non-zero code and an error message identifying yt-dlp as the missing tool before any download or processing begins.
2. **Given** a `-ytdlp-path` flag points to a valid yt-dlp binary not on PATH, **When** goblin starts, **Then** it uses the specified path and proceeds normally.

---

### User Story 4 — MANIFEST Records Source URL (Priority: P3)

The output MANIFEST.json includes the source URL so a downstream consumer (Claude, an editing pipeline) knows where the video came from without inspecting file metadata.

**Why this priority**: Provenance is part of the MANIFEST contract per Constitution Principle III (machine-first output). A future session loading the MANIFEST should be able to re-fetch the video if needed.

**Independent Test**: Run goblin on a URL. Verify MANIFEST.json contains a `source_url` field with the original URL value.

**Acceptance Scenarios**:

1. **Given** goblin processes a URL input, **When** the MANIFEST is written, **Then** MANIFEST.json contains `"source_url": "<the original URL>"`.
2. **Given** goblin processes a local file, **When** the MANIFEST is written, **Then** MANIFEST.json does not contain a `source_url` field (no regression on local-file behavior).

---

### Edge Cases

- What if the URL points to a private or age-restricted video? goblin should propagate yt-dlp's error clearly and exit with a non-zero code.
- What if the download is interrupted midway? The partial file should not be passed to the processing pipeline; goblin should exit with an error.
- What if the output directory already contains a file with the same name as the download target? Behavior should follow the existing `-overwrite` flag convention.
- What if yt-dlp downloads a format that ffprobe cannot analyze? goblin should propagate the ffprobe error — this is the same failure path as an unsupported local file.
- What if the URL is structurally valid but not a video (e.g., a YouTube playlist URL)? yt-dlp behavior for playlists varies; goblin should document that single-video URLs are the supported input, and playlist behavior is undefined.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: goblin MUST accept a URL beginning with `http://` or `https://` as its positional input argument, auto-detecting it as a network source rather than a local file path.
- **FR-002**: When a URL input is detected, goblin MUST invoke yt-dlp to download the video before passing it to the existing pipeline stages.
- **FR-003**: goblin MUST check for yt-dlp availability at startup when a URL is the input, and exit immediately with a clear diagnostic if yt-dlp is not found.
- **FR-004**: goblin MUST accept a `-ytdlp-path` flag that specifies the explicit path to the yt-dlp binary, overriding PATH lookup.
- **FR-005**: The default download quality MUST be capped at 720p height; a `-max-height N` flag MUST override the cap (0 = no cap).
- **FR-006**: The downloaded video file MUST be retained in the output directory after processing completes.
- **FR-007**: MANIFEST.json MUST include a `source_url` field containing the original URL when the input was a network source.
- **FR-008**: Local file input behavior MUST be unchanged — no regression on any existing flag or output format.
- **FR-009**: The download stage MUST surface yt-dlp errors (private video, network failure, unsupported URL) as goblin errors with the original yt-dlp message included.

### Key Entities

- **URL Input**: A network address (http/https) passed as goblin's positional input argument. Triggers the download stage before the existing pipeline.
- **Downloaded File**: The local video file produced by yt-dlp, stored in the output directory. Treated as a local file by all subsequent pipeline stages.
- **source_url**: A new MANIFEST.json field recording the network source of the media, present only for URL inputs.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can run `goblin <youtube-url> -o <dir>` and receive a complete MANIFEST in one command, with no manual download step.
- **SC-002**: The output MANIFEST produced from a URL input is structurally identical to one produced from the equivalent local file — same fields, same formats, same relative paths, with the addition of `source_url`.
- **SC-003**: A missing yt-dlp dependency produces a diagnostic within 1 second of goblin startup — before any download or processing begins.
- **SC-004**: Default 720p cap reduces download size by at least 40% compared to best-available quality for a typical 1080p tutorial video.
- **SC-005**: All existing goblin tests pass without modification after this feature is implemented.

---

## Assumptions

- yt-dlp is the download backend; youtube-dl (present on this machine) is superseded and not used.
- Single-video URLs are the supported input. Playlist URLs are undefined behavior for this spec.
- The output directory is writable and has sufficient space for the downloaded video before goblin is invoked. goblin does not pre-check available disk space.
- yt-dlp is configured to use ffmpeg (already on PATH) for muxing best-video + best-audio streams into a single file.
- The `-ytdlp-path` flag follows the same resolution pattern as goblin's existing `-model` flag for whisper.
- `source_url` is a top-level field in MANIFEST.json, consistent with the existing flat structure of that file.
