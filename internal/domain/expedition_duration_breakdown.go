package domain

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// ExpeditionDurationBreakdown records the timing of distinct phases within
// a single expedition run, enabling fine-grained telemetry.
type ExpeditionDurationBreakdown struct {
	// PromptBuildDuration is the time spent in BuildPrompt() before Claude is invoked.
	PromptBuildDuration time.Duration
	// InvokeDuration is the wall-clock time from claude.invoke span start to end.
	InvokeDuration time.Duration
}

// Total returns the sum of all phase durations.
func (b ExpeditionDurationBreakdown) Total() time.Duration {
	return b.PromptBuildDuration + b.InvokeDuration
}

// SpanAttributes returns OpenTelemetry span attributes for the breakdown,
// expressing each duration in milliseconds for compatibility with dashboards.
func (b ExpeditionDurationBreakdown) SpanAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int64("expedition.prompt_build_ms", b.PromptBuildDuration.Milliseconds()),
		attribute.Int64("expedition.invoke_ms", b.InvokeDuration.Milliseconds()),
	}
}
