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
