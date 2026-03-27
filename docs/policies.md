# Policy Engine

PolicyEngine dispatches domain events to registered handlers (best-effort, fire-and-forget).
Errors are logged (if logger is non-nil) but never propagated — `Dispatch()` always returns nil.

## Location

- Engine: `internal/usecase/policy.go`
- Handlers: `internal/usecase/policy_handlers.go`
- Policy definitions: `internal/domain/policy.go`
- Registration: `internal/usecase/expedition.go` → `registerExpeditionPolicies()`

## Event → Handler Mapping

| Policy Name | WHEN [EVENT] | THEN [COMMAND] | Side Effects |
|---|---|---|---|
| ExpeditionCompletedStageReport | expedition.completed | StageReport | Log (Info) + Desktop Notify + Metrics |
| InboxReceivedProcessFeedback | inbox.received | ProcessFeedback | Log (Debug) + Metrics |
| GradientChangedTriggerGommage | gradient.changed | TriggerGommage | Log (Info) + Desktop Notify + Metrics |
| DMailStagedFlushOutbox | dmail.staged | FlushOutbox | Log (Info) + Desktop Notify + Metrics |

## Gommage Recovery

When `consecutiveFailures >= 3`, the Gommage guard fires. Instead of always halting, the system classifies the failure streak and decides retry vs halt:

| Class | Detection | Recovery |
|-------|-----------|----------|
| `timeout` | "timeout" in reason | Switch model + cooldown + retry same issue |
| `rate_limit` | "rate_limit" marker in reason | Cooldown + retry same issue |
| `parse_error` | "parse_error" in reason | Inject Lumina hint + retry same issue |
| `blocker` | "blocker" in reason | Halt + escalate (traditional behavior) |
| `systematic` | No majority class (default) | Halt + escalate (traditional behavior) |

Recovery is capped at 2 retries per failure streak. After exhaustion, the system halts and escalates. The `RecoveryDecider` port interface (`usecase/port/port.go`) enables dependency injection and testing.

New events: `gommage.recovery`, `expedition.checkpoint`.

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

## Skeleton Handlers

InboxReceivedProcessFeedback is an observation-only placeholder
(Debug log + Metrics, no notification).
