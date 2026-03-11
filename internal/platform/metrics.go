package platform

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RecordExpedition increments the paintress.expedition.total OTel counter.
func RecordExpedition(ctx context.Context, status string) {
	c, _ := Meter.Int64Counter("paintress.expedition.total",
		metric.WithDescription("Total expedition completions"),
	)
	c.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("status", SanitizeUTF8(status)),
		),
	)
}

// RecordEventEmitError increments the paintress.event.emit_error.total OTel counter.
func RecordEventEmitError(ctx context.Context, eventType string) {
	c, _ := Meter.Int64Counter("paintress.event.emit_error.total",
		metric.WithDescription("Count of event emission failures"),
	)
	c.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("event_type", SanitizeUTF8(eventType)),
		),
	)
}
