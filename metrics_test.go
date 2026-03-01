package paintress_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hironow/paintress"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func makeCompletedEvent(status string, t time.Time) paintress.Event {
	data, _ := json.Marshal(paintress.ExpeditionCompletedData{Status: status})
	return paintress.Event{ID: "test", Type: paintress.EventExpeditionCompleted, Timestamp: t, Data: data}
}

func TestSuccessRate_AllSuccess(t *testing.T) {
	now := time.Now()
	events := []paintress.Event{
		makeCompletedEvent("success", now),
		makeCompletedEvent("success", now.Add(time.Minute)),
	}

	rate := paintress.SuccessRate(events)

	if rate != 1.0 {
		t.Errorf("SuccessRate = %f, want 1.0", rate)
	}
}

func TestSuccessRate_AllFailed(t *testing.T) {
	now := time.Now()
	events := []paintress.Event{
		makeCompletedEvent("failed", now),
		makeCompletedEvent("failed", now.Add(time.Minute)),
	}

	rate := paintress.SuccessRate(events)

	if rate != 0.0 {
		t.Errorf("SuccessRate = %f, want 0.0", rate)
	}
}

func TestSuccessRate_Mixed(t *testing.T) {
	now := time.Now()
	events := []paintress.Event{
		makeCompletedEvent("success", now),
		makeCompletedEvent("failed", now.Add(time.Minute)),
		makeCompletedEvent("success", now.Add(2*time.Minute)),
		makeCompletedEvent("skipped", now.Add(3*time.Minute)),
	}

	// 2 success out of 3 non-skipped = 0.666...
	rate := paintress.SuccessRate(events)

	if rate < 0.66 || rate > 0.67 {
		t.Errorf("SuccessRate = %f, want ~0.666", rate)
	}
}

func TestSuccessRate_NoEvents(t *testing.T) {
	rate := paintress.SuccessRate(nil)

	if rate != 0.0 {
		t.Errorf("SuccessRate = %f, want 0.0", rate)
	}
}

func TestSuccessRate_OnlySkipped(t *testing.T) {
	now := time.Now()
	events := []paintress.Event{
		makeCompletedEvent("skipped", now),
		makeCompletedEvent("skipped", now.Add(time.Minute)),
	}

	// All skipped → no relevant events → 0
	rate := paintress.SuccessRate(events)

	if rate != 0.0 {
		t.Errorf("SuccessRate = %f, want 0.0", rate)
	}
}

func TestSuccessRate_IgnoresNonCompletedEvents(t *testing.T) {
	now := time.Now()
	events := []paintress.Event{
		{ID: "1", Type: paintress.EventExpeditionStarted, Timestamp: now},
		makeCompletedEvent("success", now.Add(time.Minute)),
		{ID: "3", Type: paintress.EventDMailStaged, Timestamp: now.Add(2 * time.Minute)},
		makeCompletedEvent("failed", now.Add(3*time.Minute)),
	}

	rate := paintress.SuccessRate(events)

	if rate != 0.5 {
		t.Errorf("SuccessRate = %f, want 0.5", rate)
	}
}

func TestRecordExpedition_IncreasesCounter(t *testing.T) {
	// given
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	origMeter := paintress.Meter
	paintress.Meter = mp.Meter("test")
	defer func() { paintress.Meter = origMeter }()
	ctx := context.Background()

	// when
	paintress.RecordExpedition(ctx, "success")
	paintress.RecordExpedition(ctx, "failed")
	paintress.RecordExpedition(ctx, "success")

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

func TestFormatSuccessRate_NoEvents(t *testing.T) {
	// given
	rate := 0.0
	success := 0
	total := 0

	// when
	result := paintress.FormatSuccessRate(rate, success, total)

	// then
	if result != "no events" {
		t.Errorf("FormatSuccessRate = %q, want %q", result, "no events")
	}
}

func TestFormatSuccessRate_AllSuccess(t *testing.T) {
	// given
	rate := 1.0
	success := 3
	total := 3

	// when
	result := paintress.FormatSuccessRate(rate, success, total)

	// then
	expected := "100.0% (3/3)"
	if result != expected {
		t.Errorf("FormatSuccessRate = %q, want %q", result, expected)
	}
}

func TestFormatSuccessRate_Mixed(t *testing.T) {
	// given
	rate := 2.0 / 3.0
	success := 2
	total := 3

	// when
	result := paintress.FormatSuccessRate(rate, success, total)

	// then
	expected := "66.7% (2/3)"
	if result != expected {
		t.Errorf("FormatSuccessRate = %q, want %q", result, expected)
	}
}

func TestFormatSuccessRate_AllFailed(t *testing.T) {
	// given
	rate := 0.0
	success := 0
	total := 5

	// when
	result := paintress.FormatSuccessRate(rate, success, total)

	// then
	expected := "0.0% (0/5)"
	if result != expected {
		t.Errorf("FormatSuccessRate = %q, want %q", result, expected)
	}
}
