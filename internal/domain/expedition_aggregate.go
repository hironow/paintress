package domain

import "time"

// ExpeditionAggregate owns expedition lifecycle state and produces events.
// It tracks consecutive failures for gommage decisions and gradient state.
type ExpeditionAggregate struct {
	consecutiveFailures int
}

// NewExpeditionAggregate creates an empty ExpeditionAggregate.
func NewExpeditionAggregate() *ExpeditionAggregate {
	return &ExpeditionAggregate{}
}

// ConsecutiveFailures returns the current consecutive failure count.
func (a *ExpeditionAggregate) ConsecutiveFailures() int {
	return a.consecutiveFailures
}

// StartExpedition produces an expedition.started event.
func (a *ExpeditionAggregate) StartExpedition(expedition, worker int, model string, now time.Time) (Event, error) {
	return NewEvent(EventExpeditionStarted, ExpeditionStartedData{
		Expedition: expedition,
		Worker:     worker,
		Model:      model,
	}, now)
}

// CompleteExpedition produces events for an expedition result.
// On success, consecutive failures are reset. On failure, they increment.
// Returns the expedition.completed event plus a gradient.changed event if applicable.
func (a *ExpeditionAggregate) CompleteExpedition(expedition int, status, issueID, bugsFound string, now time.Time) ([]Event, error) {
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
	case "failed", "parse_error":
		a.consecutiveFailures++
	case "skipped":
		// skipped does not affect consecutive failures
	}

	return events, nil
}

// ShouldGommage returns true if consecutive failures have reached the threshold.
func (a *ExpeditionAggregate) ShouldGommage(threshold int) bool {
	return a.consecutiveFailures >= threshold
}

// RecordGommage produces a gommage.triggered event.
func (a *ExpeditionAggregate) RecordGommage(expedition int, now time.Time) (Event, error) {
	return NewEvent(EventGommageTriggered, GommageTriggeredData{
		Expedition:          expedition,
		ConsecutiveFailures: a.consecutiveFailures,
	}, now)
}

// RecordGradientChange produces a gradient.changed event.
func (a *ExpeditionAggregate) RecordGradientChange(level int, operator string, now time.Time) (Event, error) {
	return NewEvent(EventGradientChanged, GradientChangedData{
		Level:    level,
		Operator: operator,
	}, now)
}

// RecordInboxReceived produces an inbox.received event.
func (a *ExpeditionAggregate) RecordInboxReceived(name, severity string, now time.Time) (Event, error) {
	return NewEvent(EventInboxReceived, InboxReceivedData{
		Name:     name,
		Severity: severity,
	}, now)
}

// RecordRetryAttempted produces a retry.attempted event.
func (a *ExpeditionAggregate) RecordRetryAttempted(dmailKey string, attempt int, now time.Time) (Event, error) {
	return NewEvent(EventRetryAttempted, RetryAttemptedData{
		DMail:   dmailKey,
		Attempt: attempt,
	}, now)
}

// RecordEscalated produces an escalated event.
func (a *ExpeditionAggregate) RecordEscalated(dmailName string, issues []string, now time.Time) (Event, error) {
	return NewEvent(EventEscalated, EscalatedData{
		DMail:  dmailName,
		Issues: issues,
	}, now)
}

// RecordResolved produces a resolved event.
func (a *ExpeditionAggregate) RecordResolved(dmailName string, issues []string, now time.Time) (Event, error) {
	return NewEvent(EventResolved, ResolvedData{
		DMail:  dmailName,
		Issues: issues,
	}, now)
}

// RecordDMailStaged produces a dmail.staged event.
func (a *ExpeditionAggregate) RecordDMailStaged(name string, now time.Time) (Event, error) {
	return NewEvent(EventDMailStaged, DMailStagedData{Name: name}, now)
}

// RecordDMailFlushed produces a dmail.flushed event.
func (a *ExpeditionAggregate) RecordDMailFlushed(count int, now time.Time) (Event, error) {
	return NewEvent(EventDMailFlushed, DMailFlushedData{Count: count}, now)
}

// RecordDMailArchived produces a dmail.archived event.
func (a *ExpeditionAggregate) RecordDMailArchived(name string, now time.Time) (Event, error) {
	return NewEvent(EventDMailArchived, DMailArchivedData{Name: name}, now)
}
