package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestWaitForDMail_ArrivalReturnsDMail(t *testing.T) {
	// given
	ch := make(chan domain.DMail, 1)
	sent := domain.DMail{Name: "test-dmail", Kind: "feedback"}
	ch <- sent

	// when
	got, err := session.WaitForDMail(context.Background(), ch, time.Minute, &domain.NopLogger{})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil DMail when D-Mail is on channel")
	}
	if got.Name != sent.Name {
		t.Errorf("expected Name=%q, got %q", sent.Name, got.Name)
	}
	if got.Kind != sent.Kind {
		t.Errorf("expected Kind=%q, got %q", sent.Kind, got.Kind)
	}
}

func TestWaitForDMail_TimeoutReturnsNil(t *testing.T) {
	// given
	ch := make(chan domain.DMail)

	// when
	got, err := session.WaitForDMail(context.Background(), ch, 10*time.Millisecond, &domain.NopLogger{})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil on timeout")
	}
}

func TestWaitForDMail_CancelReturnsNil(t *testing.T) {
	// given
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan domain.DMail)
	cancel()

	// when
	got, err := session.WaitForDMail(ctx, ch, time.Minute, &domain.NopLogger{})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil on context cancel")
	}
}

func TestWaitForDMail_ClosedChannelReturnsNil(t *testing.T) {
	// given
	ch := make(chan domain.DMail)
	close(ch)

	// when
	got, err := session.WaitForDMail(context.Background(), ch, time.Minute, &domain.NopLogger{})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil on closed channel")
	}
}

func TestWaitForDMail_ZeroTimeout_UsesMaxWaitDuration(t *testing.T) {
	// given — timeout=0 should use maxWaitDuration safety cap, not block forever
	restore := session.ExportSetMaxWaitDuration(20 * time.Millisecond)
	t.Cleanup(restore)
	ch := make(chan domain.DMail) // no D-Mail will arrive

	// when
	start := time.Now()
	got, err := session.WaitForDMail(context.Background(), ch, 0, &domain.NopLogger{})
	elapsed := time.Since(start)

	// then — should return within safety cap, not hang
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil on safety cap timeout")
	}
	if elapsed > 1*time.Second {
		t.Errorf("expected quick return via safety cap, took %s", elapsed)
	}
}

func TestWaitForDMail_PreservesAllFields(t *testing.T) {
	// given — verify that metadata, severity, and other fields survive the round-trip
	ch := make(chan domain.DMail, 1)
	sent := domain.DMail{
		Name:     "feedback-round-2",
		Kind:     "feedback",
		Severity: "high",
		Metadata: map[string]string{
			"feedback_round": "2",
			"issue_id":       "ENG-123",
		},
	}
	ch <- sent

	// when
	got, err := session.WaitForDMail(context.Background(), ch, time.Minute, &domain.NopLogger{})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil DMail")
	}
	if got.Severity != "high" {
		t.Errorf("expected Severity=high, got %q", got.Severity)
	}
	if got.Metadata["feedback_round"] != "2" {
		t.Errorf("expected feedback_round=2, got %q", got.Metadata["feedback_round"])
	}
	if got.Metadata["issue_id"] != "ENG-123" {
		t.Errorf("expected issue_id=ENG-123, got %q", got.Metadata["issue_id"])
	}
}
