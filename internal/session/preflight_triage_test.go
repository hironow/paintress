package session

// white-box-reason: session internals: tests unexported triagePreFlightDMails method

import (
	"context"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// countingEmitter wraps NopExpeditionEventEmitter and counts EmitEscalated
// and EmitRetryAttempted calls for assertion in triage tests.
type countingEmitter struct {
	port.NopExpeditionEventEmitter
	escalatedCount     int
	retryAttemptedCount int
}

func (c *countingEmitter) EmitEscalated(_ string, _ []string, _ time.Time) error {
	c.escalatedCount++
	return nil
}

func (c *countingEmitter) EmitRetryAttempted(_ string, _ int, _ time.Time) error {
	c.retryAttemptedCount++
	return nil
}

func TestTriagePreFlightDMails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		dmails             []domain.DMail
		maxRetries         int
		preloadRetries     int // how many times to pre-call Track before triage
		preloadIssues      []string
		wantRemaining      int
		wantEscalated      int
		wantRetryAttempted int
	}{
		{
			name: "no action passes through",
			dmails: []domain.DMail{
				{Name: "info-1", Kind: "report", Description: "status update"},
			},
			maxRetries:    3,
			wantRemaining: 1,
		},
		{
			name:          "empty inbox returns zero",
			dmails:        []domain.DMail{},
			maxRetries:    3,
			wantRemaining: 0,
		},
		{
			name: "escalate removes dmail and emits event",
			dmails: []domain.DMail{
				{Name: "esc-1", Kind: "feedback", Description: "critical", Action: "escalate", Issues: []string{"MY-10"}},
			},
			maxRetries:    3,
			wantRemaining: 0,
			wantEscalated: 1,
		},
		{
			name: "resolve removes dmail",
			dmails: []domain.DMail{
				{Name: "res-1", Kind: "feedback", Description: "fixed", Action: "resolve", Issues: []string{"MY-20"}},
			},
			maxRetries:    3,
			wantRemaining: 0,
		},
		{
			name: "retry with issues under limit keeps dmail",
			dmails: []domain.DMail{
				{Name: "retry-1", Kind: "feedback", Description: "flaky", Action: "retry", Issues: []string{"MY-30"}},
			},
			maxRetries:         3,
			wantRemaining:      1,
			wantRetryAttempted: 1,
		},
		{
			name: "retry with issues at max promotes to escalation",
			dmails: []domain.DMail{
				{Name: "retry-max", Kind: "feedback", Description: "stuck", Action: "retry", Issues: []string{"MY-40"}},
			},
			maxRetries:     3,
			preloadRetries: 3, // Track called 3 times before; triage call will be the 4th > maxRetries
			preloadIssues:  []string{"MY-40"},
			wantRemaining:  0,
			wantEscalated:  1,
		},
		{
			name: "retry without issues passes through",
			dmails: []domain.DMail{
				{Name: "retry-noissue", Kind: "feedback", Description: "generic retry", Action: "retry"},
			},
			maxRetries:    3,
			wantRemaining: 1,
		},
		{
			name: "mixed actions filter correctly",
			dmails: []domain.DMail{
				{Name: "esc-mix", Kind: "feedback", Description: "escalate me", Action: "escalate", Issues: []string{"MY-50"}},
				{Name: "pass-mix", Kind: "report", Description: "just info"},
				{Name: "res-mix", Kind: "feedback", Description: "resolved", Action: "resolve", Issues: []string{"MY-51"}},
				{Name: "retry-mix", Kind: "feedback", Description: "try again", Action: "retry", Issues: []string{"MY-52"}},
			},
			maxRetries:         3,
			wantRemaining:      2, // pass-mix + retry-mix
			wantEscalated:      1, // esc-mix
			wantRetryAttempted: 1, // retry-mix
		},
		{
			name: "unknown action passes through",
			dmails: []domain.DMail{
				{Name: "unknown-1", Kind: "report", Description: "future action", Action: "future-action"},
			},
			maxRetries:    3,
			wantRemaining: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			emitter := &countingEmitter{}
			tracker := domain.NewRetryTracker()

			// preload retries to simulate previous attempts
			for range tt.preloadRetries {
				tracker.Track(tt.preloadIssues)
			}

			p := &Paintress{
				Emitter:      emitter,
				config:       domain.Config{MaxRetries: tt.maxRetries},
				Logger:       &domain.NopLogger{},
				retryTracker: tracker,
			}

			// when
			result := p.triagePreFlightDMails(context.Background(), tt.dmails)

			// then
			if got := len(result); got != tt.wantRemaining {
				t.Errorf("remaining D-Mails = %d, want %d", got, tt.wantRemaining)
			}
			if emitter.escalatedCount != tt.wantEscalated {
				t.Errorf("escalated count = %d, want %d", emitter.escalatedCount, tt.wantEscalated)
			}
			if emitter.retryAttemptedCount != tt.wantRetryAttempted {
				t.Errorf("retryAttempted count = %d, want %d", emitter.retryAttemptedCount, tt.wantRetryAttempted)
			}
		})
	}
}
