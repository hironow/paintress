package platform_test

import (
	"context"
	"testing"

	"github.com/hironow/paintress/internal/platform"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func sumCounter(t *testing.T, rm metricdata.ResourceMetrics, name string) int64 {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				sum := m.Data.(metricdata.Sum[int64])
				var total int64
				for _, dp := range sum.DataPoints {
					total += dp.Value
				}
				return total
			}
		}
	}
	t.Fatalf("metric %q not found", name)
	return 0
}

func TestRecordExpedition_IncreasesCounter(t *testing.T) {
	// given
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	origMeter := platform.Meter
	platform.Meter = mp.Meter("test")
	defer func() { platform.Meter = origMeter }()
	ctx := context.Background()

	// when
	platform.RecordExpedition(ctx, "success", "opus")
	platform.RecordExpedition(ctx, "failed", "sonnet")
	platform.RecordExpedition(ctx, "success", "opus")

	// then
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	total := sumCounter(t, rm, "paintress.expedition.total")
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

func TestRecordExpedition_IncludesModelAttribute(t *testing.T) {
	// given
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	origMeter := platform.Meter
	platform.Meter = mp.Meter("test")
	defer func() { platform.Meter = origMeter }()
	ctx := context.Background()

	// when
	platform.RecordExpedition(ctx, "success", "claude-sonnet-4-20250514")

	// then
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "paintress.expedition.total" {
				continue
			}
			sum := m.Data.(metricdata.Sum[int64])
			for _, dp := range sum.DataPoints {
				for _, attr := range dp.Attributes.ToSlice() {
					if string(attr.Key) == "model" && attr.Value.AsString() == "claude-sonnet-4-20250514" {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Error("expected model attribute on metric data point")
	}
}

func TestRecordEventEmitError_IncrementsCounter(t *testing.T) {
	// given
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	origMeter := platform.Meter
	platform.Meter = mp.Meter("test")
	defer func() { platform.Meter = origMeter }()
	ctx := context.Background()

	// when
	platform.RecordEventEmitError(ctx, "expedition.completed")
	platform.RecordEventEmitError(ctx, "expedition.completed")
	platform.RecordEventEmitError(ctx, "expedition.started")

	// then
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	total := sumCounter(t, rm, "paintress.event.emit_error.total")
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}
