package eventsource

import (
	"fmt"
	"sync"

	"github.com/hironow/paintress/internal/domain"
)

// eventStore is the package-local interface for SessionRecorder's store
// dependency. Kept unexported to avoid importing port from eventsource
// (prohibited by semgrep). FileEventStore satisfies this via duck typing.
type eventStore interface {
	Append(events ...domain.Event) (domain.AppendResult, error)
	LoadAll() ([]domain.Event, domain.LoadResult, error)
}

// SessionRecorder wraps a FileEventStore with automatic CorrelationID/CausationID
// assignment. Safe for concurrent use within a single process.
type SessionRecorder struct {
	store     eventStore
	sessionID string
	prevID    string
	mu        sync.Mutex
}

// NewSessionRecorder creates a SessionRecorder for the given session.
// Loads existing events to resume the CausationID chain from the last recorded event.
func NewSessionRecorder(store eventStore, sessionID string) (*SessionRecorder, error) {
	events, _, err := store.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("new session recorder: %w", err)
	}
	// Resume CausationID chain from the last event of the SAME session only.
	// Without this filter, a new session's first event would incorrectly
	// point its CausationID at the previous session's last event.
	var prevID string
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].CorrelationID == sessionID {
			prevID = events[i].ID
			break
		}
	}
	return &SessionRecorder{
		store:     store,
		sessionID: sessionID,
		prevID:    prevID,
	}, nil
}

// Record appends an event, setting CorrelationID and CausationID automatically.
func (r *SessionRecorder) Record(ev domain.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ev.CorrelationID = r.sessionID
	if r.prevID != "" {
		ev.CausationID = r.prevID
	}
	if _, err := r.store.Append(ev); err != nil {
		return err
	}
	r.prevID = ev.ID
	return nil
}
