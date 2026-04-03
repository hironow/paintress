package session

// white-box-reason: tests executeRecovery method on Paintress struct and injectParseErrorLumina internal function

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
)

func newRecoveryTestPaintress(t *testing.T) *Paintress {
	t.Helper()
	continent := t.TempDir()
	// Create required dirs
	os.MkdirAll(filepath.Join(continent, domain.StateDir, ".run", "logs"), 0755)
	os.MkdirAll(filepath.Join(continent, domain.StateDir, "insights"), 0755)
	cfg := domain.Config{Continent: continent, Model: "opus"}
	logger := platform.NewLogger(io.Discard, false)
	return NewPaintress(cfg, logger, io.Discard, io.Discard, nil, nil, nil, domain.NewExpeditionAggregate())
}

func TestExecuteRecovery_RetryOnTimeout(t *testing.T) {
	p := newRecoveryTestPaintress(t)
	decision := domain.RecoveryDecision{
		RecoveryKind: domain.RecoveryRetry,
		Class:        domain.GommageClassTimeout,
		Cooldown:     1 * time.Millisecond,
		RetryNum:     1,
		MaxRetry:     2,
		KeepWorkDir:  true,
	}
	got := p.executeRecovery(context.Background(), decision, 1, nil)
	if !got {
		t.Error("expected true (retry) for timeout recovery")
	}
}

func TestExecuteRecovery_HaltOnSystematic(t *testing.T) {
	p := newRecoveryTestPaintress(t)
	decision := domain.RecoveryDecision{
		RecoveryKind: domain.RecoveryHalt,
		Class:        domain.GommageClassSystematic,
	}
	got := p.executeRecovery(context.Background(), decision, 1, nil)
	if got {
		t.Error("expected false (halt) for systematic")
	}
}

func TestExecuteRecovery_ContextCancelled(t *testing.T) {
	p := newRecoveryTestPaintress(t)
	decision := domain.RecoveryDecision{
		RecoveryKind: domain.RecoveryRetry,
		Class:        domain.GommageClassRateLimit,
		Cooldown:     10 * time.Second,
		RetryNum:     1,
		MaxRetry:     2,
		KeepWorkDir:  true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := p.executeRecovery(ctx, decision, 1, nil)
	if got {
		t.Error("expected false when context is cancelled")
	}
}

func TestInjectParseErrorLumina_WritesInsight(t *testing.T) {
	continent := t.TempDir()
	os.MkdirAll(filepath.Join(continent, domain.StateDir, "insights"), 0755)
	os.MkdirAll(filepath.Join(continent, domain.StateDir, ".run"), 0755)
	logger := platform.NewLogger(io.Discard, false)

	injectParseErrorLumina(continent, logger)

	// Check that lumina-recovery.md was created
	insightsDir := domain.InsightsDir(continent)
	entries, _ := os.ReadDir(insightsDir)
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), "lumina-recovery") {
			found = true
			data, _ := os.ReadFile(filepath.Join(insightsDir, e.Name()))
			if !strings.Contains(string(data), "parse-error-recovery") {
				t.Errorf("insight missing parse-error-recovery title, got: %s", string(data))
			}
			break
		}
	}
	if !found {
		t.Error("expected lumina-recovery insight file to be created")
	}
}

func TestExecuteRecovery_TimeoutSwitchesModel(t *testing.T) {
	p := newRecoveryTestPaintress(t)
	// Set up reserve with a fallback model
	p.reserve = harness.NewReserveParty("opus", []string{"sonnet"}, &domain.NopLogger{})

	decision := domain.RecoveryDecision{
		RecoveryKind: domain.RecoveryRetry,
		Class:        domain.GommageClassTimeout,
		Cooldown:     1 * time.Millisecond,
		RetryNum:     1,
		MaxRetry:     2,
		KeepWorkDir:  true,
	}
	p.executeRecovery(context.Background(), decision, 1, nil)

	if !p.reserve.IsOnReserve() {
		t.Error("expected model to be switched to reserve after timeout recovery")
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want bool
	}{
		{"rate_limit keyword", "rate_limit: 429 too many requests", true},
		{"429 status", "HTTP 429 response", true},
		{"quota exhausted", "API quota exceeded", true},
		{"too many requests", "Too Many Requests from API", true},
		{"merge conflict", "merge conflict in main.go", false},
		{"test failure", "exit status 1: tests failed", false},
		{"timeout", "context deadline exceeded", false},
		{"blocker", "PR is stuck due to CI failure", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRateLimitError(tt.msg)
			if got != tt.want {
				t.Errorf("isRateLimitError(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

// newRecoveryTestPaintress uses NopExpeditionEventEmitter via nil (falls back in constructor)
var _ port.RecoveryDecider = (*domain.ExpeditionAggregate)(nil) // compile-time interface check
