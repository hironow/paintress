package domain

import (
	"fmt"
	"time"
)

// AggregateTypeExpedition is the aggregate type for expedition events.
const AggregateTypeExpedition = "expedition"

// ExpeditionAggregate owns expedition lifecycle state and produces events.
// It tracks consecutive failures for gommage decisions and gradient state.
type ExpeditionAggregate struct {
	consecutiveFailures int
	escalationFired     bool
	recoveryAttempts    int
	seqNr               uint64
}

// NewExpeditionAggregate creates an empty ExpeditionAggregate.
func NewExpeditionAggregate() *ExpeditionAggregate {
	return &ExpeditionAggregate{}
}

// nextEvent creates an event tagged with expedition aggregate identity.
func (a *ExpeditionAggregate) nextEvent(eventType EventType, data any, now time.Time) (Event, error) {
	a.seqNr++
	ev, err := NewEvent(eventType, data, now)
	if err != nil {
		return ev, err
	}
	ev.AggregateType = AggregateTypeExpedition
	ev.SeqNr = a.seqNr
	return ev, nil
}

// ConsecutiveFailures returns the current consecutive failure count.
func (a *ExpeditionAggregate) ConsecutiveFailures() int {
	return a.consecutiveFailures
}

// StartExpedition produces an expedition.started event.
func (a *ExpeditionAggregate) StartExpedition(expedition, worker int, model string, now time.Time) (Event, error) {
	return a.nextEvent(EventExpeditionStarted, ExpeditionStartedData{
		Expedition: expedition,
		Worker:     worker,
		Model:      model,
	}, now)
}

// ValidExpeditionStatus reports whether s is a recognized expedition status.
func ValidExpeditionStatus(s string) bool {
	switch s {
	case "success", "failed", "parse_error", "skipped":
		return true
	}
	return false
}

// CompleteExpedition produces events for an expedition result.
// On success, consecutive failures are reset. On failure, they increment.
// Returns the expedition.completed event plus a gradient.changed event if applicable.
func (a *ExpeditionAggregate) CompleteExpedition(expedition int, status, issueID, bugsFound string, now time.Time) ([]Event, error) {
	if !ValidExpeditionStatus(status) {
		return nil, fmt.Errorf("unrecognized expedition status: %q", status)
	}
	completedEvent, err := NewEvent(EventExpeditionCompleted, ExpeditionCompletedData{
		Expedition: expedition,
		Status:     status,
		IssueID:    issueID,
		BugsFound:  bugsFound,
	}, now)
	if err != nil {
		return nil, err
	}
	events := []Event{completedEvent}

	switch status {
	case "success":
		a.consecutiveFailures = 0
		a.escalationFired = false
	case "failed", "parse_error":
		a.consecutiveFailures++
	case "skipped":
		// skipped does not affect consecutive failures
	}

	return events, nil
}

// ShouldEscalate returns true if escalation should fire: consecutive failures
// have reached the threshold AND escalation has not yet fired for this streak.
// Once it returns true, subsequent calls return false until a success resets the streak.
func (a *ExpeditionAggregate) ShouldEscalate(threshold int) bool {
	if a.consecutiveFailures >= threshold && !a.escalationFired {
		a.escalationFired = true
		return true
	}
	return false
}

// ShouldGommage returns true if consecutive failures have reached the threshold.
func (a *ExpeditionAggregate) ShouldGommage(threshold int) bool {
	return a.consecutiveFailures >= threshold
}

// RecordGommage produces a gommage.triggered event.
func (a *ExpeditionAggregate) RecordGommage(expedition int, now time.Time) (Event, error) {
	return a.nextEvent(EventGommageTriggered, GommageTriggeredData{
		Expedition:          expedition,
		ConsecutiveFailures: a.consecutiveFailures,
	}, now)
}

// RecordGradientChange produces a gradient.changed event.
func (a *ExpeditionAggregate) RecordGradientChange(level int, operator string, now time.Time) (Event, error) {
	return a.nextEvent(EventGradientChanged, GradientChangedData{
		Level:    level,
		Operator: operator,
	}, now)
}

// RecordInboxReceived produces an inbox.received event.
func (a *ExpeditionAggregate) RecordInboxReceived(name, severity string, now time.Time) (Event, error) {
	return a.nextEvent(EventInboxReceived, InboxReceivedData{
		Name:     name,
		Severity: severity,
	}, now)
}

// RecordRetryAttempted produces a retry.attempted event.
func (a *ExpeditionAggregate) RecordRetryAttempted(dmailKey string, attempt int, now time.Time) (Event, error) {
	return a.nextEvent(EventRetryAttempted, RetryAttemptedData{
		DMail:   dmailKey,
		Attempt: attempt,
	}, now)
}

// RecordEscalated produces an escalated event.
func (a *ExpeditionAggregate) RecordEscalated(dmailName string, issues []string, now time.Time) (Event, error) {
	return a.nextEvent(EventEscalated, EscalatedData{
		DMail:  dmailName,
		Issues: issues,
	}, now)
}

// RecordResolved produces a resolved event.
func (a *ExpeditionAggregate) RecordResolved(dmailName string, issues []string, now time.Time) (Event, error) {
	return a.nextEvent(EventResolved, ResolvedData{
		DMail:  dmailName,
		Issues: issues,
	}, now)
}

// RecordDMailStaged produces a dmail.staged event.
func (a *ExpeditionAggregate) RecordDMailStaged(name string, now time.Time) (Event, error) {
	return a.nextEvent(EventDMailStaged, DMailStagedData{Name: name}, now)
}

// RecordDMailFlushed produces a dmail.flushed event.
func (a *ExpeditionAggregate) RecordDMailFlushed(count int, now time.Time) (Event, error) {
	return a.nextEvent(EventDMailFlushed, DMailFlushedData{Count: count}, now)
}

// RecordDMailArchived produces a dmail.archived event.
func (a *ExpeditionAggregate) RecordDMailArchived(name string, now time.Time) (Event, error) {
	return a.nextEvent(EventDMailArchived, DMailArchivedData{Name: name}, now)
}

const maxRecoveryAttempts = 2

// DecideRecovery classifies the failure streak and decides retry vs halt.
// Tracks recovery attempts; after maxRecoveryAttempts, forces halt.
func (a *ExpeditionAggregate) DecideRecovery(reasons []string) RecoveryDecision {
	class := ClassifyGommage(reasons)
	switch class {
	case GommageClassTimeout, GommageClassRateLimit, GommageClassParseError:
		if a.recoveryAttempts >= maxRecoveryAttempts {
			return RecoveryDecision{Action: RecoveryHalt, Class: class}
		}
		a.recoveryAttempts++
		return RecoveryDecision{
			Action:      RecoveryRetry,
			Class:       class,
			Cooldown:    CooldownForClass(class, a.recoveryAttempts),
			RetryNum:    a.recoveryAttempts,
			MaxRetry:    maxRecoveryAttempts,
			KeepWorkDir: true,
		}
	default:
		return RecoveryDecision{Action: RecoveryHalt, Class: class}
	}
}

// ResetRecovery clears recovery attempts. Called when consecutiveFailures resets.
func (a *ExpeditionAggregate) ResetRecovery() {
	a.recoveryAttempts = 0
}

// RecordGommageRecovery produces a gommage.recovery event.
func (a *ExpeditionAggregate) RecordGommageRecovery(expedition int, class GommageClass, action string, retryNum int, cooldown string, now time.Time) (Event, error) {
	return a.nextEvent(EventGommageRecovery, GommageRecoveryData{
		Expedition: expedition,
		Class:      class,
		Action:     action,
		RetryNum:   retryNum,
		Cooldown:   cooldown,
	}, now)
}

// RecordCheckpoint produces an expedition.checkpoint event.
func (a *ExpeditionAggregate) RecordCheckpoint(expedition int, phase, workDir string, commitCount int, now time.Time) (Event, error) {
	return a.nextEvent(EventExpeditionCheckpoint, ExpeditionCheckpointData{
		Expedition:  expedition,
		Phase:       phase,
		WorkDir:     workDir,
		CommitCount: commitCount,
	}, now)
}
