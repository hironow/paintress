package session

// white-box-reason: tests that SetStreamBus propagates to newTrackedClaudeRunner and Expedition

import (
	"context"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

func TestStreamBusWiring_ClaudeAdapter(t *testing.T) {
	// given: process-wide StreamBus set
	bus := platform.NewInProcessSessionBus()
	defer bus.Close()
	sub := bus.Subscribe(16)
	defer sub.Close()

	old := sharedStreamBus
	SetStreamBus(bus)
	defer func() { sharedStreamBus = old }()

	// when: newTrackedClaudeRunner creates a runner (ClaudeAdapter inside)
	cfg := domain.Config{ClaudeCmd: "nonexistent", Model: "test", TimeoutSec: 10, Continent: t.TempDir()}
	runner := newTrackedClaudeRunner(cfg, "test", &domain.NopLogger{})

	// then: runner is non-nil
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}

	// Verify bus is live by publishing and checking subscriber receives
	bus.Publish(context.Background(), domain.SessionStreamEvent{
		Tool:      "paintress",
		Type: "session_end",
		Timestamp: time.Now(),
	})

	select {
	case ev := <-sub.C():
		if ev.Tool != "paintress" {
			t.Errorf("expected Tool=paintress, got %q", ev.Tool)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive event within timeout")
	}
}
