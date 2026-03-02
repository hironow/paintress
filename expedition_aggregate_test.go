package paintress_test

import (
	"testing"
	"time"

	"github.com/hironow/paintress"
)

func TestExpeditionAggregate_StartExpedition(t *testing.T) {
	// given
	agg := paintress.NewExpeditionAggregate()

	// when
	ev, err := agg.StartExpedition(1, 0, "claude-sonnet-4-6", time.Now().UTC())

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != paintress.EventExpeditionStarted {
		t.Errorf("expected type %s, got %s", paintress.EventExpeditionStarted, ev.Type)
	}
}

func TestExpeditionAggregate_CompleteExpedition_Success(t *testing.T) {
	// given
	agg := paintress.NewExpeditionAggregate()

	// when
	events, err := agg.CompleteExpedition(1, "success", "ISS-123", "", time.Now().UTC())

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should produce expedition.completed + gradient.changed events
	if len(events) < 1 {
		t.Fatalf("expected at least 1 event, got %d", len(events))
	}
	if events[0].Type != paintress.EventExpeditionCompleted {
		t.Errorf("expected type %s, got %s", paintress.EventExpeditionCompleted, events[0].Type)
	}
	// consecutive failures should be 0 after success
	if agg.ConsecutiveFailures() != 0 {
		t.Errorf("expected consecutive failures 0, got %d", agg.ConsecutiveFailures())
	}
}

func TestExpeditionAggregate_CompleteExpedition_Failure(t *testing.T) {
	// given
	agg := paintress.NewExpeditionAggregate()

	// when
	events, err := agg.CompleteExpedition(1, "failure", "", "", time.Now().UTC())

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) < 1 {
		t.Fatal("expected at least 1 event")
	}
	// consecutive failures should be 1 after failure
	if agg.ConsecutiveFailures() != 1 {
		t.Errorf("expected consecutive failures 1, got %d", agg.ConsecutiveFailures())
	}
}

func TestExpeditionAggregate_ShouldGommage(t *testing.T) {
	// given: 3 consecutive failures (threshold)
	agg := paintress.NewExpeditionAggregate()
	now := time.Now().UTC()
	for i := range 3 {
		agg.CompleteExpedition(i+1, "failure", "", "", now)
	}

	// when
	shouldStop := agg.ShouldGommage(3)

	// then
	if !shouldStop {
		t.Error("expected ShouldGommage=true after 3 failures")
	}
}

func TestExpeditionAggregate_ShouldGommage_BelowThreshold(t *testing.T) {
	// given: 2 consecutive failures (below threshold of 3)
	agg := paintress.NewExpeditionAggregate()
	now := time.Now().UTC()
	for i := range 2 {
		agg.CompleteExpedition(i+1, "failure", "", "", now)
	}

	// when
	shouldStop := agg.ShouldGommage(3)

	// then
	if shouldStop {
		t.Error("expected ShouldGommage=false when below threshold")
	}
}

func TestExpeditionAggregate_GommageEvent(t *testing.T) {
	// given: consecutive failures at threshold
	agg := paintress.NewExpeditionAggregate()
	now := time.Now().UTC()
	for i := range 3 {
		agg.CompleteExpedition(i+1, "failure", "", "", now)
	}

	// when
	ev, err := agg.RecordGommage(4, now)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != paintress.EventGommageTriggered {
		t.Errorf("expected type %s, got %s", paintress.EventGommageTriggered, ev.Type)
	}
}

func TestExpeditionAggregate_SuccessResetsFailures(t *testing.T) {
	// given: 2 consecutive failures then a success
	agg := paintress.NewExpeditionAggregate()
	now := time.Now().UTC()
	agg.CompleteExpedition(1, "failure", "", "", now)
	agg.CompleteExpedition(2, "failure", "", "", now)
	agg.CompleteExpedition(3, "success", "ISS-1", "", now)

	// when
	shouldStop := agg.ShouldGommage(3)

	// then: success resets counter
	if shouldStop {
		t.Error("expected ShouldGommage=false after success reset")
	}
	if agg.ConsecutiveFailures() != 0 {
		t.Errorf("expected consecutive failures 0 after success, got %d", agg.ConsecutiveFailures())
	}
}
