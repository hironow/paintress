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
| ExpeditionCompletedStageReport | expedition.completed | StageReport | Log (Info) + Desktop notification (5s timeout) |
| InboxReceivedProcessFeedback | inbox.received | ProcessFeedback | Log (Debug) |
| GradientChangedTriggerGommage | gradient.changed | TriggerGommage | Log (Debug) |
| DMailStagedFlushOutbox | dmail.staged | FlushOutbox | Log (Debug) |

## Event Payload Format

| Event | Payload Type | Fields |
|---|---|---|
| expedition.completed | `domain.ExpeditionCompletedData` | `Expedition`, `Status` |
| inbox.received | (none) | uses `event.Type` |
| gradient.changed | (none) | uses `event.Type` |
| dmail.staged | (none) | uses `event.Type` |

## Dispatch Guarantee

Best-effort (at-most-once). Handler failures are silently logged.
No retry, no dead-letter queue, no error propagation to callers.

## Skeleton Handlers

InboxReceivedProcessFeedback, GradientChangedTriggerGommage,
and DMailStagedFlushOutbox are logging-only placeholders.
