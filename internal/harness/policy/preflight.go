package policy

import "github.com/hironow/paintress/internal/domain"

// PreFlightDecision captures the deterministic triage outcome for a single
// inbox D-Mail before side effects (emit/archive/pass-through) are applied.
type PreFlightDecision struct {
	PassThrough bool
	Escalate    bool
	Resolve     bool
	TrackRetry  bool
}

// DeterminePreFlightDecision decides how pre-flight triage should handle a
// D-Mail given the current retry count.
func DeterminePreFlightDecision(dm domain.DMail, retryCount, maxRetries int) PreFlightDecision {
	switch dm.Action {
	case "escalate":
		return PreFlightDecision{Escalate: true}
	case "resolve":
		return PreFlightDecision{Resolve: true}
	case "retry":
		if len(dm.Issues) == 0 {
			return PreFlightDecision{PassThrough: true}
		}
		if retryCount > maxRetries {
			return PreFlightDecision{Escalate: true}
		}
		return PreFlightDecision{PassThrough: true, TrackRetry: true}
	default:
		return PreFlightDecision{PassThrough: true}
	}
}
