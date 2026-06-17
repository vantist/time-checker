---
attempts: 1
---

## Attempt 1 — FIXED

Root cause: Claude Code Stop hook stdin has no token data. Token fields live in `transcript_path` JSONL under `message.usage` on assistant entries.

Fix: `cmd/tt/record.go` — added `TranscriptPath` to `hookPayload`, `extractTokensFromTranscript()` reads JSONL and returns last assistant usage as flat JSON, wired into `resolveResponseInput`.

