package session_test

import (
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
