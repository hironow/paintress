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
	events, err := agg.CompleteExpedition(1, "success", "ISS-123", "", "", "", time.Now().UTC())

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
	events, err := agg.CompleteExpedition(1, "failed", "", "", "", "", time.Now().UTC())

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
		agg.CompleteExpedition(i+1, "failed", "", "", "", "", now)
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
		agg.CompleteExpedition(i+1, "failed", "", "", "", "", now)
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
		agg.CompleteExpedition(i+1, "failed", "", "", "", "", now)
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
		name              string
		status            string
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
			agg.CompleteExpedition(1, "failed", "", "", "", "", now)
			before := agg.ConsecutiveFailures()

			// when
			events, err := agg.CompleteExpedition(2, tt.status, "", "", "", "", now)

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

func TestExpeditionAggregate_CompleteExpedition_RejectsUnknownStatus(t *testing.T) {
	// given
	agg := domain.NewExpeditionAggregate()

	// when
	events, err := agg.CompleteExpedition(1, "typo_status", "", "", "", "", time.Now().UTC())

	// then
	if err == nil {
		t.Fatal("expected error for unknown status, got nil")
	}
	if events != nil {
		t.Errorf("expected nil events, got %v", events)
	}
	if agg.ConsecutiveFailures() != 0 {
		t.Errorf("expected consecutive failures 0, got %d", agg.ConsecutiveFailures())
	}
}

func TestValidExpeditionStatus(t *testing.T) {
	known := []string{"success", "failed", "parse_error", "skipped"}
	for _, s := range known {
		if !domain.ValidExpeditionStatus(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	unknown := []string{"typo", "", "timeout", "SUCCESS"}
	for _, s := range unknown {
		if domain.ValidExpeditionStatus(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestExpeditionAggregate_SuccessResetsFailures(t *testing.T) {
	// given: 2 consecutive failures then a success
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()
	agg.CompleteExpedition(1, "failed", "", "", "", "", now)
	agg.CompleteExpedition(2, "failed", "", "", "", "", now)
	agg.CompleteExpedition(3, "success", "ISS-1", "", "", "", now)

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

func TestShouldEscalate_FiresOncePerStreak(t *testing.T) {
	// given: 3 consecutive failures
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()
	for i := range 3 {
		agg.CompleteExpedition(i+1, "failed", "", "", "", "", now)
	}

	// when / then: first call returns true, second returns false
	if !agg.ShouldEscalate(3) {
		t.Fatal("expected ShouldEscalate=true on first call")
	}
	if agg.ShouldEscalate(3) {
		t.Fatal("expected ShouldEscalate=false on second call (already fired)")
	}
}

func TestShouldEscalate_ResetsOnSuccess(t *testing.T) {
	// given: escalation fired, then success, then new failure streak
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()
	for i := range 3 {
		agg.CompleteExpedition(i+1, "failed", "", "", "", "", now)
	}
	agg.ShouldEscalate(3) // fires
	agg.CompleteExpedition(4, "success", "ISS-1", "", "", "", now)

	// when: new failure streak reaches threshold
	for i := range 3 {
		agg.CompleteExpedition(5+i, "failed", "", "", "", "", now)
	}

	// then: should fire again
	if !agg.ShouldEscalate(3) {
		t.Fatal("expected ShouldEscalate=true after success reset")
	}
}

func TestShouldEscalate_BelowThreshold(t *testing.T) {
	// given: 2 failures (below threshold of 3)
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()
	for i := range 2 {
		agg.CompleteExpedition(i+1, "failed", "", "", "", "", now)
	}

	// when / then
	if agg.ShouldEscalate(3) {
		t.Fatal("expected ShouldEscalate=false below threshold")
	}
}

func TestDecideRecovery_RetryOnTimeout(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"timeout", "timeout", "timeout"}
	d := agg.DecideRecovery(reasons)
	if d.RecoveryKind != domain.RecoveryRetry {
		t.Errorf("expected retry, got %s", d.RecoveryKind)
	}
	if d.Class != domain.GommageClassTimeout {
		t.Errorf("expected timeout class, got %s", d.Class)
	}
	if d.RetryNum != 1 {
		t.Errorf("expected retryNum=1, got %d", d.RetryNum)
	}
}

func TestDecideRecovery_RetryOnRateLimit(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"rate_limit: 429", "rate_limit: 429", "rate_limit: 429"}
	d := agg.DecideRecovery(reasons)
	if d.RecoveryKind != domain.RecoveryRetry {
		t.Errorf("expected retry, got %s", d.RecoveryKind)
	}
	if d.Class != domain.GommageClassRateLimit {
		t.Errorf("expected rate_limit, got %s", d.Class)
	}
}

func TestDecideRecovery_HaltOnBlocker(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"blocker: stuck", "blocker: stuck", "blocker: stuck"}
	d := agg.DecideRecovery(reasons)
	if d.RecoveryKind != domain.RecoveryHalt {
		t.Errorf("expected halt for blocker, got %s", d.RecoveryKind)
	}
}

func TestDecideRecovery_HaltAfterMaxRetries(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"timeout", "timeout", "timeout"}
	agg.DecideRecovery(reasons)      // retry 1
	agg.DecideRecovery(reasons)      // retry 2
	d := agg.DecideRecovery(reasons) // should halt
	if d.RecoveryKind != domain.RecoveryHalt {
		t.Errorf("expected halt after max retries, got %s", d.RecoveryKind)
	}
}

func TestDecideRecovery_ResetOnSuccess(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"timeout", "timeout", "timeout"}
	agg.DecideRecovery(reasons)      // retry 1
	agg.ResetRecovery()              // success happened
	d := agg.DecideRecovery(reasons) // should be retry 1 again
	if d.RetryNum != 1 {
		t.Errorf("expected retryNum=1 after reset, got %d", d.RetryNum)
	}
}

func TestExpeditionAggregate_RecordEscalated_DotCase(t *testing.T) {
	// given
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()

	// when
	ev, err := agg.RecordEscalated("stall-001", []string{"ISS-1"}, now)

	// then
	if err != nil {
		t.Fatalf("RecordEscalated error: %v", err)
	}
	if ev.Type != domain.EventEscalated {
		t.Errorf("event type = %q, want %q", ev.Type, domain.EventEscalated)
	}
	if string(ev.Type) != "issue.escalated" {
		t.Errorf("event type string = %q, want %q", string(ev.Type), "issue.escalated")
	}
}

func TestExpeditionAggregate_RecordResolved_DotCase(t *testing.T) {
	// given
	agg := domain.NewExpeditionAggregate()
	now := time.Now().UTC()

	// when
	ev, err := agg.RecordResolved("resolve-001", []string{"ISS-2"}, now)

	// then
	if err != nil {
		t.Fatalf("RecordResolved error: %v", err)
	}
	if ev.Type != domain.EventResolved {
		t.Errorf("event type = %q, want %q", ev.Type, domain.EventResolved)
	}
	if string(ev.Type) != "issue.resolved" {
		t.Errorf("event type string = %q, want %q", string(ev.Type), "issue.resolved")
	}
}
