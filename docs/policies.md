# Policy Engine

PolicyEngine dispatches domain events to registered handlers (best-effort, fire-and-forget).
Errors are logged (if logger is non-nil) but never propagated — `Dispatch()` always returns nil.

## Location

- Engine: `internal/usecase/policy.go` (implements `port.EventDispatcher`)
- Policy declarations: `internal/domain/types.go` → `var Policies` (declarative WHEN/THEN registry)
- Wiring: `internal/usecase/emitter.go` (EventStore persistence + dispatch)

## Post jun15 MCP pivot: declarative registry only

The headless expedition loop that executed these policies was retired with the
jun15 MCP pivot (ADR 0017/0018). **No handlers are registered in production code
today** — the `domain.Policies` registry documents the reactive intent, and the
reactions are driven by the human-initiated Claude Code session via the
`/expedition-next` skill and the paintress MCP tools.

| Policy Name | WHEN [EVENT] | THEN [COMMAND] | Executed by (post-pivot) |
|---|---|---|---|
| ExpeditionCompletedStageReport | expedition.completed | StageReport | Claude Code session (skill workflow; emission tool gap tracked in refs issue 0031) |
| InboxReceivedProcessFeedback | inbox.received | ProcessFeedback | Claude Code session (reads inbox D-Mails) |
| GradientChangedTriggerGommage | gradient.changed | TriggerGommage | Claude Code session (gauge read model) |
| DMailStagedFlushOutbox | dmail.staged | FlushOutbox | transactional outbox (stage → atomic flush) |

## Gommage classification (domain logic, still live)

`internal/domain/gommage_classifier.go` classifies a failure streak by majority
vote over reason keywords. The classification itself is pure domain logic and
remains live as a read model; the recovery *actions* below describe the
declarative intent that the session applies post-pivot.

| Class | Detection | Recovery intent |
|-------|-----------|-----------------|
| `timeout` | "timeout" in reason | Switch model + cooldown + retry same issue |
| `rate_limit` | "rate_limit" marker in reason | Cooldown + retry same issue |
| `parse_error` | "parse_error" in reason | Inject Lumina hint + retry same issue |
| `blocker` | "blocker" in reason | Halt + escalate |
| `systematic` | No majority class (default) | Halt + escalate |

Recovery is capped at 2 retries per failure streak. After exhaustion, halt and
escalate to the human. The `RecoveryDecider` port interface
(`usecase/port/port.go`) enables dependency injection and testing.

Events: `gommage.recovery`, `expedition.checkpoint`.

## Event Payload Format

| Event | Payload Type | Fields |
|---|---|---|
| expedition.completed | `domain.ExpeditionCompletedData` | `Expedition`, `Status` |
| gommage.triggered | `domain.GommageTriggeredData` | `Expedition`, `ConsecutiveFailures`, `Class`, `RecoveryAction`, `RetryNum` |
| gommage.recovery | `domain.GommageRecoveryData` | `Expedition`, `Class`, `Action`, `RetryNum`, `Cooldown` |
| expedition.checkpoint | `domain.ExpeditionCheckpointData` | `Expedition`, `Phase`, `WorkDir`, `CommitCount` |
| inbox.received | (none) | uses `event.Type` |
| gradient.changed | (none) | uses `event.Type` |
| dmail.staged | (none) | uses `event.Type` |

## Dispatch Guarantee

Best-effort (at-most-once). Handler failures are silently logged.
No retry, no dead-letter queue, no error propagation to callers.
