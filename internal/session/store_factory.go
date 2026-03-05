package session

import (
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/eventsource"
	"github.com/hironow/paintress/internal/usecase/port"
)

// NewEventStore creates an event store for the given state directory.
// eventsource is the event persistence adapter (AWS Event Sourcing pattern).
// cmd layer should use this instead of importing eventsource directly (ADR S0008).
func NewEventStore(stateDir string, logger domain.Logger) port.EventStore {
	return eventsource.NewFileEventStore(filepath.Join(stateDir, "events"), logger)
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
