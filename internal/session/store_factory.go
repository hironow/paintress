package session

import (
	"github.com/hironow/paintress"
	"github.com/hironow/paintress/internal/eventsource"
)

// NewEventStore creates an event store for the given events directory.
// cmd layer should use this instead of importing eventsource directly (ADR S0008).
func NewEventStore(eventsDir string) paintress.EventStore {
	return eventsource.NewFileEventStore(eventsDir)
}

// ListExpiredEventFiles returns .jsonl files older than the given days.
// cmd layer should use this instead of importing eventsource directly (ADR S0008).
func ListExpiredEventFiles(stateDir string, days int) ([]string, error) {
	return eventsource.ListExpiredEventFiles(stateDir, days)
}

// PruneEventFiles deletes the named .jsonl files from the events directory.
// cmd layer should use this instead of importing eventsource directly (ADR S0008).
func PruneEventFiles(stateDir string, files []string) ([]string, error) {
	return eventsource.PruneEventFiles(stateDir, files)
}
