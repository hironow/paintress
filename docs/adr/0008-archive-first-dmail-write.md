# 0008. Archive-First D-Mail Write Pattern

**Date:** 2026-02-23
**Status:** Accepted

## Context

D-Mail messages are written to two directories: `outbox/` (for phonewave
courier pickup and delivery) and `archive/` (for permanent audit trail, git
tracked). If outbox/ is written first and the process crashes before archive/
is written, the message is delivered but has no permanent record. If archive/
is written first and outbox/ fails, the permanent record exists but delivery
does not happen — a recoverable situation since the courier can retry.

Additionally, with `--workers > 1`, multiple workers may attempt to archive
the same inbox D-Mail simultaneously via `os.Rename`, causing the second
worker to fail with `ENOENT`.

## Decision

Adopt archive-first write ordering and idempotent archive operations:

1. **`SendDMail`** (`dmail.go`): Writes to `archive/` first, then `outbox/`.
   The permanent audit record is prioritized over courier delivery. If the
   outbox write fails, the archive still preserves the message content.

2. **`ArchiveInboxDMail`** (`dmail.go`): Uses `os.Rename` for atomic move from
   `inbox/` to `archive/`. When `os.Rename` returns `fs.ErrNotExist`, a
   three-way check determines the outcome:
   - Source gone AND archive destination exists (`os.Stat` confirms) → return
     nil (another worker already archived it, idempotent)
   - Source gone AND archive destination also missing → return error (genuinely
     missing file, not a concurrent archive)
   - `os.Stat` on archive destination fails → return error (technical failure)

3. **Git tracking**: `archive/` is git tracked (audit trail survives repository
   clones). `outbox/` and `inbox/` are gitignored (transient, consumed by
   external tools).

## Consequences

### Positive

- Permanent record survives even if outbox write or process crashes mid-operation
- Concurrent workers archiving the same D-Mail is handled gracefully (idempotent)
- Three-way stat check distinguishes "already archived" from "genuinely missing"

### Negative

- `archive/` git history grows with each processed D-Mail (mitigated by
  `archive-prune` CLI subcommand for periodic cleanup)

### Neutral

- `SendDMail` stamps `DMailSchemaVersion` as a safety net if the caller omits it
- Archive-first ordering is a durability guarantee, not a consistency guarantee —
  a message may exist in archive without being delivered
