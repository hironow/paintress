package usecase

// white-box-reason: policy internals: tests unexported PolicyEngine constructor and Dispatch

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestPolicyEngine_Dispatch_NoHandlers(t *testing.T) {
	// given
	engine := NewPolicyEngine(nil)
	ev, err := domain.NewEvent(domain.EventExpeditionStarted, domain.ExpeditionStartedData{
		Expedition: 1,
		Worker:     0,
		Model:      "claude-sonnet-4-6",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	dispatchErr := engine.Dispatch(context.Background(), ev)

	// then
	if dispatchErr != nil {
		t.Fatalf("expected no error, got: %v", dispatchErr)
	}
}

func TestPolicyEngine_RegisterAndFire(t *testing.T) {
	// given
	engine := NewPolicyEngine(nil)
	var fired bool
	engine.Register(domain.EventExpeditionCompleted, func(ctx context.Context, ev domain.Event) error {
		fired = true
		return nil
	})
	ev, err := domain.NewEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: 1,
		Status:     "success",
		IssueID:    "ISS-123",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	dispatchErr := engine.Dispatch(context.Background(), ev)

	// then
	if dispatchErr != nil {
		t.Fatalf("expected no error, got: %v", dispatchErr)
	}
	if !fired {
		t.Fatal("expected handler to fire")
	}
}

func TestPolicyEngine_MultipleHandlers(t *testing.T) {
	// given
	engine := NewPolicyEngine(nil)
	var count int
	for range 3 {
		engine.Register(domain.EventGradientChanged, func(ctx context.Context, ev domain.Event) error {
			count++
			return nil
		})
	}
	ev, err := domain.NewEvent(domain.EventGradientChanged, domain.GradientChangedData{
		Level:    3,
		Operator: "+1",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	dispatchErr := engine.Dispatch(context.Background(), ev)

	// then
	if dispatchErr != nil {
		t.Fatalf("expected no error, got: %v", dispatchErr)
	}
	if count != 3 {
		t.Fatalf("expected 3 handlers to fire, got %d", count)
	}
}

func TestPolicyEngine_HandlerError_BestEffort(t *testing.T) {
	// given: two handlers — first fails, second succeeds
	engine := NewPolicyEngine(nil)
	var secondFired bool
	engine.Register(domain.EventGommageTriggered, func(ctx context.Context, ev domain.Event) error {
		return fmt.Errorf("handler failed")
	})
	engine.Register(domain.EventGommageTriggered, func(ctx context.Context, ev domain.Event) error {
		secondFired = true
		return nil
	})
	ev, err := domain.NewEvent(domain.EventGommageTriggered, domain.GommageTriggeredData{
		Expedition:          5,
		ConsecutiveFailures: 3,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	dispatchErr := engine.Dispatch(context.Background(), ev)

	// then: best-effort — error swallowed, all handlers execute, nil returned
	if dispatchErr != nil {
		t.Fatalf("expected nil (best-effort), got: %v", dispatchErr)
	}
	if !secondFired {
		t.Fatal("second handler should fire even after first handler error")
	}
}

func TestPolicyEngine_UnmatchedEventType(t *testing.T) {
	// given: register for expedition.completed only
	engine := NewPolicyEngine(nil)
	var fired bool
	engine.Register(domain.EventExpeditionCompleted, func(ctx context.Context, ev domain.Event) error {
		fired = true
		return nil
	})
	ev, err := domain.NewEvent(domain.EventDMailStaged, domain.DMailStagedData{
		Name: "test.dmail",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	dispatchErr := engine.Dispatch(context.Background(), ev)

	// then
	if dispatchErr != nil {
		t.Fatalf("expected no error, got: %v", dispatchErr)
	}
	if fired {
		t.Fatal("handler should not fire for unmatched event type")
	}
}
