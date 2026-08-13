<!-- Sync Impact Report
Version change: template → 1.0.0 (initial ratification)
Added sections: Core Principles (I–V), External Dependencies, Governance
Removed sections: none (template replaced)
TODOs: none
-->

# goblin Constitution

## Core Principles

### I. Analyze, Never Modify

goblin's sole output is derived data — keyframes, transcripts, manifests.
It MUST NOT open any source media file for writing under any code path.
A goblin invocation that alters the source file is a defect, not a feature.

Rationale: tools that analyze and tools that edit have incompatible failure
modes. Keeping analysis read-only makes goblin safe to run on irreplaceable
source material without ceremony.

### II. Local by Default

All processing MUST occur on-device. No media content, frames, or transcript
data leave the machine unless the user explicitly requests a network operation
(e.g., future URL-fetch support). Network access is opt-in, clearly flagged,
and never a side effect of normal operation.

Rationale: source media may be sensitive, unfinished, or under NDA. The
operator must always control data residency.

### III. Machine-First Output

The MANIFEST directory format — MANIFEST.json, frames/, transcript.json — is
designed for downstream tool consumption: Claude and editing pipelines
(DaVinci Resolve MCP, ffmpeg orchestration). Human readability is secondary.
Output MUST use JSON for structured data, stable relative paths for assets,
and ISO 8601 timestamps. Prose summaries are NOT part of the output format.

Rationale: goblin is middleware. Its consumers are programs and AI models,
not terminal readers. An output format optimized for humans imposes a parsing
cost on every downstream consumer.

### IV. Composable Pipeline

The processing stages — probe, extract, transcribe — MUST be independently
invocable and independently useful. A caller that wants only metadata (no
frames, no transcript) MUST be able to get exactly that. The full pipeline
is the default, not the only mode.

Each stage reads inputs, writes outputs, and exits. No stage retains state
between invocations. Outputs of one stage are valid inputs to another.

Rationale: composition is how Unix tools scale. A monolithic pipeline that
cannot be partially invoked cannot be tested, debugged, or reused at the
granularity of its components.

### V. Format-Agnostic Input

goblin MUST accept any container/codec combination that ffmpeg can process.
Format detection and negotiation are delegated entirely to ffprobe and ffmpeg.
goblin MUST NOT hardcode format assumptions, reject files by extension, or
special-case specific codecs in application logic.

Rationale: the space of media formats in production use is large and growing.
A tool that prescribes input format is a tool that fails on real-world files.

## External Dependencies

Required on PATH at runtime:
- `ffprobe` — media metadata analysis (part of the FFmpeg suite)
- `ffmpeg` — scene-detection keyframe extraction

Required for transcription (gracefully skipped if absent and transcription
not requested):
- `whisper` or compatible whisper.cpp CLI (`main`, `whisper-cli`) — timestamped
  speech-to-text transcription

Optional (used when available, skipped when not):
- `mediainfo` — supplementary container metadata beyond ffprobe
- `mkvextract` (MKVToolNix) — embedded subtitle track extraction; preferred
  over transcription when professional subtitles are present in the container

goblin MUST check for required tools at startup and fail immediately with
a clear diagnostic if any required tool is missing.

## Governance

This constitution is the highest-authority design document for goblin.
It supersedes README, inline comments, and implementation decisions when
they conflict.

Amendments require:
1. A rationale explaining why the principle no longer serves the project.
2. An updated `LAST_AMENDED_DATE` and incremented `CONSTITUTION_VERSION`.
3. Review by Hoyt before the amendment is committed.

The spec in `.specify/` is the implementation authority for each release.
The constitution governs what kinds of specs are acceptable.

**Version**: 1.0.0 | **Ratified**: 2026-08-10 | **Last Amended**: 2026-08-10
