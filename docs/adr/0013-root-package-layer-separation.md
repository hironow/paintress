# 0013. Root Package Layer Separation

**Date:** 2026-02-26
**Status:** Accepted — correction: InitTracer moved to session (not cmd as stated below)

## Context

The paintress root package contains 4,347 source lines across 25+ files with no
structural separation between pure types/functions and I/O operations (filesystem,
subprocess, network). This flat structure makes it difficult to reason about
side effects and violates the types-only root convention established in
phonewave (0010), sightjack (0011-0012), and amadeus (0014).

### Layer count analysis

sightjack uses 4 layers (root / domain / eventsource / session), but paintress
differs in two ways:

1. **Go method constraint**: paintress's pure logic lives in type methods
   (`GradientGauge.Charge()`, `ReserveParty.CheckOutput()`, etc.). Go requires
   methods to be defined in the same package as the receiver type. Extracting a
   `domain` layer would force moving the types themselves, leaving root empty.

2. **No event sourcing**: paintress has no event store (unlike sightjack and
   amadeus). Adding an `eventsource` layer would be YAGNI.

phonewave uses 2 layers (root / internal/service) at 576 source lines.
paintress at 4,347 lines is closer in complexity to phonewave than to
sightjack (22,000 lines), and the 2-layer model fits naturally.

## Decision

Adopt a 2-layer architecture matching phonewave's pattern:

```
+-------------------+     +--------------------+     +-------------------+
| internal/cmd/     | --> | internal/session/  | --> | paintress (root)  |
| (cobra CLI)       |     | (I/O, subprocess,  |     | (types, pure fn,  |
|                   |     |  filesystem, net)   |     |  go:embed, logger) |
+-------------------+     +--------------------+     +-------------------+
```

### Root package retains

- Type definitions: `Config`, `DMail`, `Expedition`, `ExpeditionFlag`,
  `JournalEntry`, `Issue`, `Lumina`, `ReviewResult`, `PruneResult`,
  `DoctorCheck`, `ProjectConfig`, `RunSummary`, `ExpeditionReport`
- Pure state machines: `GradientGauge`, `ReserveParty`
- Pure functions: `Msg()`, `ParseReport()`, `FormatDMailForPrompt()`,
  `FlagPath()`, `ProjectConfigPath()`, `FormatSummaryJSON()`
- Interfaces: `Approver`, `Notifier`, `GitExecutor`
- go:embed assets: template files, skills directory
- Logger: `logger.go` (shared exception per S0001)

### internal/session/ receives

- All `Paintress` orchestration: `NewPaintress()`, `Run()`, worker loops
- Filesystem I/O: `ReadFlag`, `WriteFlag`, `WriteJournal`,
  `SendDMail`, `ScanInbox`, `LoadProjectConfig`, `SaveProjectConfig`
- Subprocess: `WorktreePool`, `DevServer`, `RunReview`
- Network: `FetchIssues` (Linear GraphQL), `InitTracer` (OTLP)
- fsnotify watchers: `watchInbox`, `watchFlag`
- Approval/notification implementations: `StdinApprover`, `CmdApprover`,
  `LocalNotifier`, `CmdNotifier`
- `ArchivePrune`, `RunDoctor`, `RunInitWithReader`

### Not applied

- **4-layer separation**: sightjack's domain and eventsource layers are
  unnecessary for paintress (see Context). Re-evaluate if paintress grows
  event sourcing or exceeds 10,000 source lines.

## Consequences

### Positive

- Clear I/O boundary — root package is safe to import without side effects
- Consistent with phonewave (0010), sightjack (0012), amadeus (0014)
- Pure types in root enable easier testing of logic without I/O mocks
- Each structural commit is independently valid (`just check` passes)

### Negative

- Cross-package references increase import verbosity (e.g.,
  `session.NewPaintress()` instead of `paintress.NewPaintress()`)
- Some type/function splits require careful dependency ordering

### Neutral

- `logger.go` remains in root per S0001 shared exception
- go:embed directives move with their consuming types to internal/session
  where the type itself moves (Go allows go:embed in any package)
- Telemetry tracer global stays in root; `InitTracer()` moves to cmd
