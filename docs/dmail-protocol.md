# D-Mail Protocol

D-Mail is the inter-tool communication protocol used by the paintress ecosystem (paintress, sightjack, amadeus, phonewave). Messages are Markdown files with YAML frontmatter, exchanged through directory-based mailboxes.

## Wire Format

A D-Mail is a Markdown file with Jekyll/Hugo-style YAML frontmatter delimiters (`---`):

```
---
name: pt-report-my-42_9e1d4f8a
kind: report
description: "Expedition #1 completed implement for MY-42"
issues:
    - MY-42
dmail-schema-version: "1"
---

# Expedition #1 Report: Fix login bug

- **Issue:** MY-42
- **Mission:** implement
- **Status:** success
- **PR:** https://github.com/org/repo/pull/123
```

### Frontmatter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique message identifier (e.g., `pt-report-my-42_9e1d4f8a`) |
| `kind` | string | Yes | Message type: `report`, `specification`, `design-feedback`, `implementation-feedback` |
| `description` | string | Yes | Human-readable summary |
| `issues` | string[] | No | Related Linear issue IDs |
| `severity` | string | No | `high` triggers the approval gate |
| `action` | string | No | Requested action (e.g., `implement`, `review`, `fix`) |
| `priority` | int | No | Priority level (0 = unset, higher = more urgent) |
| `dmail-schema-version` | string | No | Protocol version (currently `"1"`) |
| `metadata` | map | No | Arbitrary key-value pairs |
| `context` | object | No | Insight context attached to outbound D-Mails (ADR S0031) |

#### Context Field (S0031)

The optional `context` field carries insight summaries from the Insight Ledger, enabling downstream tools to receive accumulated knowledge without reading insight files directly.

```yaml
---
name: pt-report-my-42_9e1d4f8a
kind: report
description: "Expedition #1 completed implement for MY-42"
issues:
    - MY-42
dmail-schema-version: "1"
context:
  insights:
    - source: ".expedition/insights/lumina.md"
      summary: "auth CI flaky"
    - source: ".expedition/insights/gommage.md"
      summary: "timeout on large repos needs --timeout 2400"
---
```

The `context.insights` array contains `InsightSummary` objects with `source` (file path) and `summary` (human-readable text). When no insights exist, the `context` field is omitted entirely.

### Body

The body section after the closing `---` is optional Markdown content. The body is separated from the closing delimiter by a blank line.

## Schema Versioning

Every outbound D-Mail carries a `dmail-schema-version` field in its frontmatter. The version string is centralized in the Go constant `DMailSchemaVersion` (currently `"1"`).

- `NewReportDMail` sets `SchemaVersion: DMailSchemaVersion` at construction time
- `SendDMail` stamps `DMailSchemaVersion` if the caller omits it (safety net)
- The version must match `dmail-frontmatter.v1.schema.json` (maintained by phonewave)

Bump `DMailSchemaVersion` when the frontmatter format changes.

## Directories

| Directory | Git Status | Purpose |
|-----------|-----------|---------|
| `.expedition/inbox/` | Ignored | Incoming d-mails from external tools |
| `.expedition/outbox/` | Ignored | Outgoing d-mails for courier pickup |
| `.expedition/archive/` | Tracked | Processed d-mails (audit trail) |

## Lifecycle

```
[External tool]          paintress                   [External tool]
     |                      |                              |
     | writes to inbox/     |                              |
     |--------------------->|                              |
     |                      | ScanInbox()                  |
     |                      | triagePreFlightDMails()      |
     |                      |   escalate -> archive + emit |
     |                      |   resolve  -> archive + emit |
     |                      |   retry    -> pass or escalate|
     |                      | FormatDMailForPrompt()        |
     |                      | -> embedded in prompt         |
     |                      |                              |
     |                      | (expedition runs)            |
     |                      |                              |
     |                      | on success:                  |
     |                      |   NewReportDMail()           |
     |                      |   SendDMail() -> archive/ + outbox/
     |                      |   ArchiveInboxDMail() -> inbox/ to archive/
     |                      |                              |
     |                      |              reads outbox/   |
     |                      |----------------------------->|
```

### Pre-Flight D-Mail Triage

Before creating an expedition, `triagePreFlightDMails` processes the `action` field on each inbox D-Mail:

| Action | Behavior | Passed to Expedition |
|--------|----------|---------------------|
| `escalate` | Emit escalation event, archive immediately | No (consumed) |
| `resolve` | Emit resolved event, archive immediately | No (consumed) |
| `retry` | Track retry count; pass through if under max retries, escalate if exceeded | Yes (if under limit) |
| (none/other) | Pass through unchanged | Yes |

Triaged-out D-Mails (escalate, resolve, over-limit retry) are archived immediately during pre-flight, not after expedition completion. This prevents re-processing on the next scan.

### Ordering Guarantees

- `SendDMail` writes to **archive/ first**, then outbox/ (archive-first for durability)
- `ScanInbox` returns d-mails **sorted by filename** for deterministic ordering
- `ArchiveInboxDMail` uses `os.Rename` for atomic move; idempotent — returns nil only if source is gone AND destination already exists in archive (confirmed by `os.Stat`), errors on genuinely missing source

### Mid-Expedition Arrivals

D-mails arriving mid-expedition are detected by `watchInbox` (fsnotify) and logged, but NOT archived. They remain in inbox/ for the next expedition's `ScanInbox`.

- **`high` severity**: Triggers desktop notification via `Notifier` (no approval gate mid-expedition). Counted in `totalMidHighSeverity` and recorded in journal/flag.
- **Issue-matched**: If the d-mail's `issues` field matches the expedition's `current_issue`, it is collected for a `--continue` follow-up turn after the expedition completes.

## Function Map

| Function | File | Purpose |
|----------|------|---------|
| `ParseDMail` | `dmail.go` | Parse bytes into DMail struct |
| `DMail.Marshal` | `dmail.go` | Serialize DMail to wire format |
| `FormatDMailForPrompt` | `dmail.go` | Format d-mails for prompt injection |
| `NewReportDMail` | `dmail.go` | Create report d-mail from ExpeditionReport |
| `FilterHighSeverity` | `dmail.go` | Filter d-mails with severity=high |
| `SendDMail` | `internal/session/dmail.go` | Write to archive/ then outbox/ |
| `ScanInbox` | `internal/session/dmail.go` | Read all .md files from inbox/ |
| `ArchiveInboxDMail` | `internal/session/dmail.go` | Move inbox/ file to archive/ (idempotent if already archived) |
| `TriagePreFlightDMails` | `internal/usecase/preflight_triage.go` | Pre-flight action processing (escalate/resolve/retry) — delegated via `port.PreFlightTriager` |
| `InsightsDir` | `dmail.go` | Path to insights directory |
| `RunDir` | `dmail.go` | Path to run directory (SQLite, locks, logs) |
| `InsightWriter.Append` | `internal/session/insight_writer.go` | Append insight entry to ledger file (flock + atomic rename, idempotent) |
| `InsightWriter.Read` | `internal/session/insight_writer.go` | Parse insight ledger file |

## HIGH Severity Gate

When inbox contains `high` severity d-mails, paintress runs a pre-flight gate:

1. `Notifier.Notify()` — Desktop notification (fire-and-forget)
2. `Approver.RequestApproval()` — Blocking approval request

See [approval-contract.md](./approval-contract.md) for the three-way approval contract.

## Agent Skills (SKILL.md)

SKILL.md manifests in `.expedition/skills/` declare D-Mail capabilities following the [Agent Skills](https://agentskills.io) specification. These manifests enable phonewave to discover what each tool can send and receive.

### dmail-sendable

```yaml
---
name: dmail-sendable
description: Produces D-Mail report messages to outbox/ after expedition completion.
license: Apache-2.0
metadata:
  dmail-schema-version: "1"
  produces:
    - kind: report
---
```

### dmail-readable

```yaml
---
name: dmail-readable
description: Consumes D-Mail specifications and implementation-feedback from inbox/.
license: Apache-2.0
metadata:
  dmail-schema-version: "1"
  consumes:
    - kind: specification
    - kind: implementation-feedback
---
```

### SKILL.md Top-Level Fields

Per Agent Skills spec, only these top-level fields are valid: `name`, `description`, `license`, `compatibility`, `metadata`, `allowed-tools`. Tool-specific capabilities (`produces`, `consumes`) go under `metadata`.

### Schema Version in SKILL.md

The `dmail-schema-version` in `metadata` tells external tools which protocol version the skill supports. phonewave uses this to validate compatibility before routing d-mails.

### Template Management

SKILL.md files are embedded in the paintress binary via `go:embed` from `templates/skills/`. `ValidateContinent` copies them to `.expedition/skills/` on first run (creates if absent, does not overwrite existing files).
