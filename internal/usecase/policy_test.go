package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hironow/paintress"
)

func TestPolicyEngine_Dispatch_NoHandlers(t *testing.T) {
	// given
	engine := NewPolicyEngine(nil)
	ev, err := paintress.NewEvent(paintress.EventExpeditionStarted, paintress.ExpeditionStartedData{
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
	engine.Register(paintress.EventExpeditionCompleted, func(ctx context.Context, ev paintress.Event) error {
		fired = true
		return nil
	})
	ev, err := paintress.NewEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{
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
		engine.Register(paintress.EventGradientChanged, func(ctx context.Context, ev paintress.Event) error {
			count++
			return nil
		})
	}
	ev, err := paintress.NewEvent(paintress.EventGradientChanged, paintress.GradientChangedData{
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

func TestPolicyEngine_HandlerError(t *testing.T) {
	// given
	engine := NewPolicyEngine(nil)
	engine.Register(paintress.EventGommageTriggered, func(ctx context.Context, ev paintress.Event) error {
		return fmt.Errorf("handler failed")
	})
	ev, err := paintress.NewEvent(paintress.EventGommageTriggered, paintress.GommageTriggeredData{
		Expedition:          5,
		ConsecutiveFailures: 3,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	dispatchErr := engine.Dispatch(context.Background(), ev)

	// then
	if dispatchErr == nil {
		t.Fatal("expected error from handler")
	}
}

func TestPolicyEngine_UnmatchedEventType(t *testing.T) {
	// given: register for expedition.completed only
	engine := NewPolicyEngine(nil)
	var fired bool
	engine.Register(paintress.EventExpeditionCompleted, func(ctx context.Context, ev paintress.Event) error {
		fired = true
		return nil
	})
	ev, err := paintress.NewEvent(paintress.EventDMailStaged, paintress.DMailStagedData{
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
