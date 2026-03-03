package domain_test

import (
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestExpeditionAggregate_StartExpedition(t *testing.T) {
	// given
	agg := domain.NewExpeditionAggregate()

	// when
	ev, err := agg.StartExpedition(1, 0, "claude-sonnet-4-6", time.Now().UTC())

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != domain.EventExpeditionStarted {
		t.Errorf("expected type %s, got %s", domain.EventExpeditionStarted, ev.Type)
	}
}

func TestExpeditionAggregate_CompleteExpedition_Success(t *testing.T) {
	// given
	agg := domain.NewExpeditionAggregate()

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
	if events[0].Type != domain.EventExpeditionCompleted {
		t.Errorf("expected type %s, got %s", domain.EventExpeditionCompleted, events[0].Type)
	}
	// consecutive failures should be 0 after success
	if agg.ConsecutiveFailures() != 0 {
		t.Errorf("expected consecutive failures 0, got %d", agg.ConsecutiveFailures())
	}
}

func TestExpeditionAggregate_CompleteExpedition_Failure(t *testing.T) {
	// given
	agg := domain.NewExpeditionAggregate()

	// when
	events, err := agg.CompleteExpedition(1, "failed", "", "", time.Now().UTC())

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
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()
	for i := range 3 {
		agg.CompleteExpedition(i+1, "failed", "", "", now)
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
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()
	for i := range 2 {
		agg.CompleteExpedition(i+1, "failed", "", "", now)
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
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()
	for i := range 3 {
		agg.CompleteExpedition(i+1, "failed", "", "", now)
	}

	// when
	ev, err := agg.RecordGommage(4, now)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != domain.EventGommageTriggered {
		t.Errorf("expected type %s, got %s", domain.EventGommageTriggered, ev.Type)
	}
}

func TestExpeditionAggregate_StatusVocabulary(t *testing.T) {
	tests := []struct {
		name             string
		status           string
		wantFailuresDelta int // 0 = no change, 1 = increment
	}{
		{"success resets failures", "success", 0},
		{"failed increments failures", "failed", 1},
		{"parse_error increments failures", "parse_error", 1},
		{"skipped does not affect failures", "skipped", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: aggregate with 1 pre-existing failure
			agg := domain.NewExpeditionAggregate()
			now := time.Now().UTC()
			agg.CompleteExpedition(1, "failed", "", "", now)
			before := agg.ConsecutiveFailures()

			// when
			events, err := agg.CompleteExpedition(2, tt.status, "", "", now)

			// then
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) < 1 {
				t.Fatal("expected at least 1 event")
			}
			if events[0].Type != domain.EventExpeditionCompleted {
				t.Errorf("expected type %s, got %s", domain.EventExpeditionCompleted, events[0].Type)
			}

			after := agg.ConsecutiveFailures()
			switch tt.status {
			case "success":
				if after != 0 {
					t.Errorf("success should reset failures to 0, got %d", after)
				}
			default:
				wantAfter := before + tt.wantFailuresDelta
				if after != wantAfter {
					t.Errorf("status %q: expected failures %d, got %d", tt.status, wantAfter, after)
				}
			}
		})
	}
}

func TestExpeditionAggregate_SuccessResetsFailures(t *testing.T) {
	// given: 2 consecutive failures then a success
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()
	agg.CompleteExpedition(1, "failed", "", "", now)
	agg.CompleteExpedition(2, "failed", "", "", now)
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
