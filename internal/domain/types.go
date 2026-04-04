package domain

import (
	"time"
)


// shutdownKey is the context key for the outer (shutdown) context.
type shutdownKey struct{}

// ShutdownKey is used to embed the outer context in workCtx via context.WithValue.
// Commands retrieve it to get a context that survives workCtx cancellation.
var ShutdownKey = shutdownKey{}

// IndexEntry represents one line in the archive index JSONL file.
type IndexEntry struct {
	Timestamp string `json:"ts"`
	Operation string `json:"op"`
	Issue     string `json:"issue"`
	Status    string `json:"status"`
	Tool      string `json:"tool"`
	Path      string `json:"path"`
	Summary   string `json:"summary"`
}

// HandoverState captures in-progress work state when an operation is
// interrupted by a signal. The struct is pure data — no context, no I/O.
type HandoverState struct {
	Tool         string // "paintress"
	Operation    string // "expedition"
	Timestamp    time.Time
	InProgress   string            // Current task description
	Completed    []string          // What was done
	Remaining    []string          // What's left
	PartialState map[string]string // Tool-specific state (key=label, value=detail)
}

// Lumina represents a learned passive skill extracted from past expedition journals.
type Lumina struct {
	Pattern string // The learned pattern / lesson
	Source  string // Which journal(s) contributed
	Uses    int    // How many times this pattern appeared
}

// ProviderErrorKind classifies the type of provider error.
type ProviderErrorKind int

const (
	// ProviderErrorNone indicates no provider-level error (normal failure).
	ProviderErrorNone ProviderErrorKind = iota
	// ProviderErrorRateLimit indicates a rate limit was hit.
	ProviderErrorRateLimit
	// ProviderErrorServer indicates a server-side error (5xx).
	ProviderErrorServer
)

// ProviderErrorInfo holds the classified result of a provider error.
type ProviderErrorInfo struct {
	Kind    ProviderErrorKind
	ResetAt time.Time // parsed reset time (zero if unknown)
}

// IsTrip returns true if the error should trip a circuit breaker.
func (i ProviderErrorInfo) IsTrip() bool {
	return i.Kind != ProviderErrorNone
}

// Policy represents an implicit reactive rule: WHEN [EVENT] THEN [COMMAND].
// See ADR S0014 for the POLICY pattern reference.
type Policy struct {
	Name    string    // unique identifier for the policy
	Trigger EventType // domain event that activates this policy
	Action  string    // description of the resulting command
}

// Policies registers all known implicit policies in paintress.
// These document the existing reactive behaviors for future automation.
var Policies = []Policy{
	{Name: "ExpeditionCompletedStageReport", Trigger: EventExpeditionCompleted, Action: "StageReport"},
	{Name: "InboxReceivedProcessFeedback", Trigger: EventInboxReceived, Action: "ProcessFeedback"},
	{Name: "GradientChangedTriggerGommage", Trigger: EventGradientChanged, Action: "TriggerGommage"},
	{Name: "DMailStagedFlushOutbox", Trigger: EventDMailStaged, Action: "FlushOutbox"},
}
