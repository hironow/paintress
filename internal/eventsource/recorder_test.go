package eventsource

// white-box-reason: tests SessionRecorder internals (prevID chaining, session-scoped causation) that require access to unexported eventStore interface

import (
	"context"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestSessionRecorder_SetsCausationChain(t *testing.T) {
	dir := t.TempDir()
	store := NewFileEventStore(dir, &domain.NopLogger{})
	rec, err := NewSessionRecorder(context.Background(), store, "session-1")
	if err != nil {
		t.Fatal(err)
	}

	ev1, _ := domain.NewEvent(domain.EventExpeditionStarted, map[string]string{}, time.Now())
	if err := rec.Record(context.Background(), ev1); err != nil {
		t.Fatal(err)
	}

	ev2, _ := domain.NewEvent(domain.EventExpeditionCompleted, map[string]string{}, time.Now())
	if err := rec.Record(context.Background(), ev2); err != nil {
		t.Fatal(err)
	}

	events, _, _ := store.LoadAll(context.Background())
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].CorrelationID != "session-1" {
		t.Errorf("event[0] CorrelationID = %q, want session-1", events[0].CorrelationID)
	}
	if events[0].CausationID != "" {
		t.Errorf("event[0] CausationID should be empty, got %q", events[0].CausationID)
	}
	if events[1].CorrelationID != "session-1" {
		t.Errorf("event[1] CorrelationID = %q, want session-1", events[1].CorrelationID)
	}
	if events[1].CausationID != events[0].ID {
		t.Errorf("event[1] CausationID = %q, want %q", events[1].CausationID, events[0].ID)
	}
}

func TestSessionRecorder_ResumesPrevID(t *testing.T) {
	dir := t.TempDir()
	store := NewFileEventStore(dir, &domain.NopLogger{})

	// First recorder writes one event with session "s1"
	rec1, _ := NewSessionRecorder(context.Background(), store, "s1")
	ev1, _ := domain.NewEvent(domain.EventExpeditionStarted, map[string]string{}, time.Now())
	rec1.Record(context.Background(), ev1)

	// Second recorder with SAME session ID should resume CausationID chain
	rec2, _ := NewSessionRecorder(context.Background(), store, "s1")
	ev2, _ := domain.NewEvent(domain.EventExpeditionCompleted, map[string]string{}, time.Now())
	rec2.Record(context.Background(), ev2)

	events, _, _ := store.LoadAll(context.Background())
	if events[1].CausationID != events[0].ID {
		t.Errorf("resumed recorder should chain: CausationID = %q, want %q",
			events[1].CausationID, events[0].ID)
	}
}

func TestSessionRecorder_DifferentSession_NoCausation(t *testing.T) {
	dir := t.TempDir()
	store := NewFileEventStore(dir, &domain.NopLogger{})

	rec1, _ := NewSessionRecorder(context.Background(), store, "s1")
	ev1, _ := domain.NewEvent(domain.EventExpeditionStarted, map[string]string{}, time.Now())
	rec1.Record(context.Background(), ev1)

	// Different session should NOT chain to s1's event
	rec2, _ := NewSessionRecorder(context.Background(), store, "s2")
	ev2, _ := domain.NewEvent(domain.EventExpeditionCompleted, map[string]string{}, time.Now())
	rec2.Record(context.Background(), ev2)

	events, _, _ := store.LoadAll(context.Background())
	if events[1].CausationID != "" {
		t.Errorf("different session should have empty CausationID, got %q", events[1].CausationID)
	}
}
