# Feature Specification: goblin Core Pipeline

**Feature Branch**: `001-core-pipeline`

**Created**: 2026-08-10

**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Full Visual and Audio Analysis (Priority: P1)

An operator points goblin at a local media file. goblin extracts scene-change
keyframes and a timestamped transcript, then writes a structured MANIFEST
directory. Claude reads the MANIFEST and can answer questions about what the
video shows and what was said, without having received the video file itself.

**Why this priority**: This is the primary use case — without it, goblin has
no reason to exist. Every other story depends on the core analysis working.

**Independent Test**: Can be fully tested by running goblin on any video file
with spoken content and verifying that Claude, given only the output directory,
can correctly describe visual scene changes and quote spoken lines with
accurate timestamps.

**Acceptance Scenarios**:

1. **Given** a valid local video file with video and audio tracks, **When**
   goblin is invoked with the file path, **Then** it produces an output
   directory containing MANIFEST.json, a frames/ subdirectory with at least
   one PNG per detected scene, and a transcript.json with timestamped segments.

2. **Given** goblin completes successfully, **When** the output directory is
   read by Claude, **Then** Claude can identify at least one specific visual
   element from a frame and quote at least one timestamped line of speech.

3. **Given** a video with no audio track, **When** goblin is invoked,
   **Then** it completes without error and omits transcript.json from the
   output, noting the absence in MANIFEST.json.

---

### User Story 2 — Technical Media Probe (Priority: P2)

An operator wants to know a file's technical characteristics — codec, resolution,
duration, track inventory, embedded subtitles — before committing to full
analysis. goblin returns structured metadata without extracting frames or
transcribing.

**Why this priority**: Probe mode is both a standalone utility and a
prerequisite for informed pipeline decisions. It enables the operator to decide
which subsequent stages are appropriate before spending GPU time on transcription.

**Independent Test**: Can be tested by running goblin in probe-only mode on
any media file and verifying the metadata output matches the file's known
properties (e.g., as reported by MediaInfo or ffprobe directly).

**Acceptance Scenarios**:

1. **Given** a local media file, **When** goblin is invoked in probe-only mode,
   **Then** it writes a probe.json to the output directory containing codec,
   duration, resolution, frame rate, audio track count, and subtitle track list.

2. **Given** a file with embedded subtitle tracks, **When** goblin is invoked
   in probe-only mode, **Then** probe.json lists each subtitle track by index,
   language code, and format.

3. **Given** a file that ffprobe cannot read, **When** goblin is invoked,
   **Then** it exits with a non-zero status and a diagnostic message naming the
   file and the underlying tool error.

---

### User Story 3 — Selective Stage Invocation (Priority: P2)

An operator needs only a subset of the pipeline. A silent video needs frames
but no transcript. A podcast needs a transcript but no frames. An already-
transcribed video needs neither — just probe metadata for editing tool handoff.
goblin runs only the requested stages.

**Why this priority**: Composability is a core principle. Forcing the full
pipeline on every invocation wastes time and makes goblin unusable for partial
workflows. This story unlocks the editing pipeline integration use case.

**Independent Test**: Can be tested by invoking goblin with individual stage
flags and verifying only the expected output files are produced.

**Acceptance Scenarios**:

1. **Given** a video file and a no-transcript flag, **When** goblin is invoked,
   **Then** the output contains frames/ and MANIFEST.json but no transcript.json.

2. **Given** an audio-only file and a no-frames flag, **When** goblin is
   invoked, **Then** the output contains transcript.json and MANIFEST.json
   but no frames/ directory.

3. **Given** a video with an embedded subtitle track and no no-transcript flag,
   **When** goblin is invoked and mkvextract is available, **Then** the
   embedded subtitles are extracted to transcript.json instead of running
   speech recognition.

---

### User Story 4 — Editing Pipeline Handoff (Priority: P3)

A downstream editing tool (such as a DaVinci Resolve MCP client) reads
goblin's MANIFEST output to understand scene boundaries and transcript cues,
then uses that information to drive editing decisions without re-analyzing
the source file.

**Why this priority**: This is the interoperability goal — MANIFEST is designed
as an EDL precursor. Without it, goblin is a useful but isolated tool rather
than the first stage of a broader pipeline.

**Independent Test**: Can be tested by writing a consumer script that reads
MANIFEST.json and produces a list of cut points and caption strings, verifying
that the data is present, correctly typed, and requires no transformation.

**Acceptance Scenarios**:

1. **Given** a completed goblin output directory, **When** a downstream tool
   reads MANIFEST.json, **Then** it can extract an ordered list of scenes,
   each with start time, end time, and the path to its representative frame,
   without making any assumptions about file layout beyond what MANIFEST.json
   states.

2. **Given** a MANIFEST.json with a linked transcript.json, **When** a
   downstream tool cross-references scene times with transcript segments,
   **Then** it can produce a caption-timed-to-scene mapping using only
   ISO 8601 timestamps.

---

### Edge Cases

- What happens when the video has no detectable scene changes? goblin MUST
  produce at least one frame (the first frame) and note in MANIFEST.json that
  no scene boundaries were detected.
- What happens when whisper-cli exits non-zero? goblin MUST report the failure
  with the tool's stderr output and exit non-zero itself. Partial transcripts
  MUST NOT be written as complete.
- What happens when the output directory already exists? goblin MUST fail with
  a clear error rather than silently overwriting prior results. An explicit
  overwrite flag is required.
- What happens when a media file has multiple video streams? goblin MUST
  default to the first video stream and document the selection in MANIFEST.json.
- What happens on a very long video (>2 hours)? Without --grid, the frame
  count may exceed Claude's context capacity; goblin MUST warn when the
  estimated frame count exceeds a configurable threshold.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: goblin MUST accept a local file path as the primary positional argument.
- **FR-002**: goblin MUST verify the presence of all required external tools
  (ffprobe, ffmpeg) at startup and exit within one second with a named
  diagnostic if any are missing.
- **FR-003**: goblin MUST probe the source file via ffprobe and record technical
  metadata (codec, duration, resolution, frame rate, audio tracks, subtitle
  tracks) in the output.
- **FR-004**: goblin MUST extract keyframes at detected scene boundaries using
  ffmpeg scene-change detection.
- **FR-005**: goblin MUST produce a timestamped transcript of spoken audio when
  transcription is not disabled and whisper-cli is available on PATH.
- **FR-006**: goblin MUST extract embedded subtitle tracks in preference to
  running speech recognition when a subtitle track is present and mkvextract
  is available.
- **FR-007**: goblin MUST write all output to a named directory: MANIFEST.json,
  frames/ (PNG keyframes), and transcript.json (when applicable).
- **FR-008**: goblin MUST NOT modify the source file under any circumstances.
- **FR-009**: goblin MUST support skipping individual stages via flags
  (probe-only, no-frames, no-transcript).
- **FR-010**: goblin MUST expose the scene detection sensitivity threshold as
  a configurable parameter with a documented default.
- **FR-011**: goblin MUST allow the whisper command name and model path to be
  specified via flag or environment variable, with no hardcoded default paths.
- **FR-012**: goblin MUST support grid mode, which packs multiple frames into
  contact sheets to reduce total output file count for long videos.
- **FR-013**: goblin MUST produce at least one frame regardless of scene
  detection results.
- **FR-014**: goblin MUST refuse to write to an existing output directory
  without an explicit overwrite flag.
- **FR-015**: goblin MUST warn when the estimated output frame count exceeds
  a configurable threshold that approximates Claude's visual context limit.

### Key Entities

- **Source file**: The input media asset; read-only; any ffmpeg-supported container.
- **Scene**: A contiguous time range with stable visual content; has a start
  time, end time, and one representative frame.
- **Frame**: A PNG image extracted at a scene boundary; has a timestamp and
  path relative to the output directory.
- **Transcript segment**: A time-bounded span of speech; has start time, end
  time, and text content.
- **MANIFEST**: The index document; references all other output artifacts by
  relative path; consumable without reading any other file first.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Given only goblin's output directory (no source file), Claude
  can correctly identify at least one specific visual element and quote at
  least one timestamped spoken line.
- **SC-002**: goblin completes without initiating any outbound network
  connections during analysis of a local file.
- **SC-003**: A downstream editing tool can read MANIFEST.json and derive an
  ordered list of scene cut points and caption strings without format
  transformation or additional file parsing beyond what MANIFEST references.
- **SC-004**: A missing required external tool produces a named error message
  within one second of invocation.
- **SC-005**: The source media file is byte-for-byte identical before and after
  goblin completes.
- **SC-006**: goblin produces usable output on any media file that ffprobe can
  read, regardless of container format or codec.

## Assumptions

- External tools (ffprobe, ffmpeg, whisper-cli) are installed and on PATH by
  the operator; goblin does not bundle, install, or download them.
- The operator supplies the Whisper model path; no model location is assumed.
- Output directories are on the local filesystem; goblin does not manage remote
  or cloud storage.
- Initial scope is local files only; URL and streaming source support are
  future extensions.
- When a container has multiple video or audio streams, goblin defaults to the
  first of each; stream selection is a future extension.
- Frame images are written as PNG; lossy formats are a future option.
- goblin is single-invocation, single-file; batch processing across multiple
  files is a future extension.
- The MANIFEST schema is versioned from v1; breaking schema changes increment
  the schema version field, allowing consumers to detect incompatibility.
