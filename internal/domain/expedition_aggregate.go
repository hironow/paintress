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
	case "failure":
		a.consecutiveFailures++
	case "skip":
		// skip does not affect consecutive failures
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
