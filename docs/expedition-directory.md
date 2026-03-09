# `.expedition/` Directory Structure

Paintress manages all expedition state under `<Continent>/.expedition/`.
This document describes what each directory/file does, who creates it, and how it flows into the expedition prompt.

## Directory Tree

```
.expedition/
  .gitignore            # auto-managed by ValidateContinent
  config.yaml           # project-scoped settings (Linear team/project)
  journal/
    001.md              # expedition #1 journal entry
    002.md              # expedition #2 journal entry
    ...
  context/
    *.md                # user-placed context files (injected into prompt)
  skills/
    dmail-sendable/
      SKILL.md          # Agent Skills spec manifest (produces: report)
    dmail-readable/
      SKILL.md          # Agent Skills spec manifest (consumes: specification, implementation-feedback)
  inbox/                # incoming d-mails (specifications, implementation-feedback from sightjack/amadeus)
    *.md
  outbox/               # outgoing d-mails (reports)
    *.md
  archive/              # processed d-mails (inbox moves here after expedition)
    *.md
  events/               # append-only event store (JSONL)
    YYYY-MM-DD.jsonl
  .run/                 # ephemeral runtime data
    flag.md             # consolidated checkpoint (written at exit from per-worker max)
    logs/
      paintress-YYYYMMDD.log
      dev-server.log
      expedition-NNN-prompt.md
      expedition-NNN-output.txt
    worktrees/          # WorktreePool managed directories
      worker-001/
        .expedition/
          .run/
            flag.md     # per-worker checkpoint (exclusive access, no mutex)
      worker-002/
        .expedition/
          .run/
            flag.md     # per-worker checkpoint
      ...
```

## Git Tracking Rules

`.expedition/.gitignore` (auto-managed by `ValidateContinent`):

```
.run/
inbox/
outbox/
.otel.env
events/
```

| Path | Git Status | Reason |
|------|-----------|--------|
| `journal/` | Tracked | Permanent knowledge; Lumina extraction source |
| `context/` | Tracked | User-curated guidance for expeditions |
| `skills/` | Tracked | Agent Skills spec manifests for phonewave discovery (see [dmail-protocol.md](./dmail-protocol.md#agent-skills-skillmd)) |
| `config.yaml` | Tracked | Project-level configuration |
| `archive/` | Tracked | Audit trail of processed d-mails |
| `events/` | Ignored | Append-only event store (JSONL, expedition events) |
| `.run/` | Ignored | Ephemeral runtime state (logs, flag, worktrees) |
| `inbox/` | Ignored | Transient; consumed and archived per expedition |
| `outbox/` | Ignored | Transient; courier picks up and delivers |

## Prompt Injection Map

The expedition prompt template embeds some content inline and references other files by path for Claude Code to read on its own.

### Inline in Prompt (embedded by paintress)

| Section | Source | Builder |
|---------|--------|---------|
| Lumina | `journal/*.md` | `ScanJournalsForLumina` -> `FormatLuminaForPrompt` |
| Gradient Gauge | Runtime state | `GradientGauge.FormatForPrompt` |
| Reserve Party | Runtime state | `ReserveParty.FormatForPrompt` |
| Context | `context/*.md` | `ReadContextFiles` -> concatenation |
| D-Mail Inbox | `inbox/*.md` | `ScanInbox` -> `FormatDMailForPrompt` |
| Mission | Hardcoded text | `MissionText()` |
| Linear Scope | `config.yaml` | `LoadProjectConfig` |
| Environment | CLI flags | Direct field mapping |

### File Path References (Claude Code reads these itself)

| Reference in Template | Path | Who Creates | Purpose |
|----------------------|------|-------------|---------|
| `flag.md` | `.expedition/.run/flag.md` (Continent) or `{worktree}/.expedition/.run/flag.md` (per-worker) | paintress (`writeFlag`) + Claude Code (`current_issue`) | Per-worker checkpoint during execution; consolidated to Continent at exit via `reconcileFlags`. Claude Code writes `current_issue`/`current_title` to its own workDir during expedition |
| `CLAUDE.md` | `<Continent>/CLAUDE.md` | User | Repository conventions |
| `journal/` latest | `.expedition/journal/NNN.md` | paintress (WriteJournal) | Direct lesson from the most recent expedition |

### Dual-Use: Journals

Journals serve two purposes simultaneously:

1. **Pre-processed** -> Lumina extraction (aggregated patterns injected inline)
2. **Direct read** -> Template instructs Claude Code to read the latest entry

## D-Mail Lifecycle

```
[External tool]          paintress                   [External tool]
     |                      |                              |
     | writes to inbox/     |                              |
     |--------------------->|                              |
     |                      | ScanInbox()                  |
     |                      | FormatDMailForPrompt()        |
     |                      | -> embedded in prompt         |
     |                      |                              |
     |                      | (expedition runs)            |
     |                      |                              |
     |                      | on success:                  |
     |                      |   NewReportDMail()           |
     |                      |   SendDMail() -> outbox/ + archive/
     |                      |   ArchiveInboxDMail() -> inbox/ to archive/
     |                      |                              |
     |                      |              reads outbox/   |
     |                      |----------------------------->|
```

- **Pre-flight triage**: `triagePreFlightDMails` processes action fields (escalate/resolve/retry) before expedition creation. Triaged-out D-Mails are archived immediately.
- **inbox/** -> triage -> prompt injection -> **archive/** (after success)
- **report** -> **archive/** first, then **outbox/** (archive-first for durability)
- `SendDMail` and `ArchiveInboxDMail` are best-effort (LogWarn on failure, never block success)
- `ArchiveInboxDMail` is idempotent — returns nil only if source is gone AND destination already exists in archive (confirmed by `os.Stat`), errors on genuinely missing source
- `watchInbox` (fsnotify) performs an initial scan of existing files on startup, then monitors for Create/Write events
- D-mails arriving mid-expedition are logged but NOT archived — they stay in inbox/ for the next expedition's `ScanInbox`
- HIGH severity d-mails arriving mid-expedition trigger notification and are counted in `totalMidHighSeverity`
- Issue-matched d-mails arriving mid-expedition are collected for a `--continue` follow-up turn after the expedition completes

For full protocol details (wire format, schema versioning, function map), see [dmail-protocol.md](./dmail-protocol.md).

## File Creators

| File | Created By | When |
|------|-----------|------|
| `.expedition/` dirs | `ValidateContinent` | CLI startup |
| `.gitignore` | `ValidateContinent` | CLI startup (upgrades append missing entries) |
| `config.yaml` | User or `SaveProjectConfig` | Manual or programmatic |
| `journal/NNN.md` | `WriteJournal` | After each expedition (success, skip, or fail) |
| `context/*.md` | User | Manual placement |
| `skills/*/SKILL.md` | `ValidateContinent` | CLI startup (created from embedded templates, updated when template changes) |
| `events/YYYY-MM-DD.jsonl` | `ExpeditionEventEmitter` | During expedition lifecycle (append-only) |
| `inbox/*.md` | External tool (courier/sightjack) | Before expedition |
| `outbox/*.md` | `SendDMail` | After successful expedition |
| `archive/*.md` | `SendDMail` + `ArchiveInboxDMail` | After successful expedition |
| `.run/flag.md` (Continent) | `reconcileFlags` + `WriteFlag` (consolidation at exit) | After all workers complete |
| `{worktree}/.expedition/.run/flag.md` | `writeFlag` (per-worker) + Claude Code (`current_issue`) | Each expedition cycle |
| `.run/logs/*` | `NewPaintress` + `Expedition.Run` | Session startup / each expedition |
| `.run/worktrees/` | `WorktreePool.Init` | Session startup (if --workers > 0) |
