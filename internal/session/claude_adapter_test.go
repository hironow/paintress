package session_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestClaudeAdapter_HasTimeoutSec(t *testing.T) {
	// given
	adapter := &session.ClaudeAdapter{
		ClaudeCmd:  "claude",
		Model:      "opus",
		TimeoutSec: 1980,
		Logger:     &domain.NopLogger{},
	}

	// then
	if adapter.TimeoutSec != 1980 {
		t.Errorf("TimeoutSec: got %d, want 1980", adapter.TimeoutSec)
	}
}

// TestClaudeAdapter_RunDetailedReturnsErrMCPPivotDeprecated is the
// canonical assertion that ClaudeAdapter.RunDetailed short-circuits
// with session.ErrMCPPivotDeprecated post jun15 MCP pivot
// (refs/issues/0027 + 0028 §4.2 residue cleanup). The previous
// streambus_wiring_test.go suite (TestStreamBusWiring_*) was retired
// because its premise (live `claude --print` invocation producing
// stream-json) no longer holds.
func TestClaudeAdapter_RunDetailedReturnsErrMCPPivotDeprecated(t *testing.T) {
	// given
	adapter := &session.ClaudeAdapter{
		ClaudeCmd: "claude",
		Model:     "opus",
		Logger:    &domain.NopLogger{},
		ToolName:  "paintress",
	}

	// when
	_, err := adapter.RunDetailed(context.Background(), "test prompt", io.Discard)

	// then
	if !errors.Is(err, session.ErrMCPPivotDeprecated) {
		t.Errorf("RunDetailed() error = %v, want ErrMCPPivotDeprecated", err)
	}
}
