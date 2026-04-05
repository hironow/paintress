package policy_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/policy"
)

func TestDeterminePreFlightDecision(t *testing.T) {
	tests := []struct {
		name       string
		dmail      domain.DMail
		retryCount int
		maxRetries int
		want       policy.PreFlightDecision
	}{
		{
			name: "escalate consumes",
			dmail: domain.DMail{
				Action: "escalate",
			},
			want: policy.PreFlightDecision{Escalate: true},
		},
		{
			name: "resolve consumes",
			dmail: domain.DMail{
				Action: "resolve",
			},
			want: policy.PreFlightDecision{Resolve: true},
		},
		{
			name: "retry without issues passes through",
			dmail: domain.DMail{
				Action: "retry",
			},
			maxRetries: 3,
			want:       policy.PreFlightDecision{PassThrough: true},
		},
		{
			name: "retry under limit tracks retry",
			dmail: domain.DMail{
				Action: "retry",
				Issues: []string{"MY-10"},
			},
			retryCount: 2,
			maxRetries: 3,
			want:       policy.PreFlightDecision{PassThrough: true, TrackRetry: true},
		},
		{
			name: "retry over limit escalates",
			dmail: domain.DMail{
				Action: "retry",
				Issues: []string{"MY-10"},
			},
			retryCount: 4,
			maxRetries: 3,
			want:       policy.PreFlightDecision{Escalate: true},
		},
		{
			name: "unknown action passes through",
			dmail: domain.DMail{
				Action: "future-action",
			},
			want: policy.PreFlightDecision{PassThrough: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.DeterminePreFlightDecision(tt.dmail, tt.retryCount, tt.maxRetries)
			if got != tt.want {
				t.Fatalf("DeterminePreFlightDecision() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
