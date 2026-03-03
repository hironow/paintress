package domain

import (
	"context"
	"encoding/json"
	"fmt"

	paintress "github.com/hironow/paintress"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RecordExpedition increments the paintress.expedition.total OTel counter.
func RecordExpedition(ctx context.Context, status string) {
	c, _ := paintress.Meter.Int64Counter("paintress.expedition.total",
		metric.WithDescription("Total expedition completions"),
	)
	c.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("status", status),
		),
	)
}

// RecordEventEmitError increments the paintress.event.emit_error.total OTel counter.
func RecordEventEmitError(ctx context.Context, eventType string) {
	c, _ := paintress.Meter.Int64Counter("paintress.event.emit_error.total",
		metric.WithDescription("Count of event emission failures"),
	)
	c.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("event_type", eventType),
		),
	)
}

// SuccessRate calculates the success rate from a list of events.
// It considers only EventExpeditionCompleted events, counting "success"
// vs "failed" outcomes. "skipped" events are excluded from the denominator.
// Returns 0.0 if there are no relevant events.
func SuccessRate(events []Event) float64 {
	var success, total int
	for _, ev := range events {
		if ev.Type != EventExpeditionCompleted {
			continue
		}
		var data ExpeditionCompletedData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			continue
		}
		if data.Status == "skipped" {
			continue
		}
		total++
		if data.Status == "success" {
			success++
		}
	}
	if total == 0 {
		return 0.0
	}
	return float64(success) / float64(total)
}

// FormatSuccessRate formats a success rate as a human-readable string.
// Returns "no events" when total is 0.
func FormatSuccessRate(rate float64, success, total int) string {
	if total == 0 {
		return "no events"
	}
	return fmt.Sprintf("%.1f%% (%d/%d)", rate*100, success, total)
}
