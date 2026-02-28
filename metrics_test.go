package paintress_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hironow/paintress"
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
		makeCompletedEvent("success", now.Add(2 * time.Minute)),
		makeCompletedEvent("skipped", now.Add(3 * time.Minute)),
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
		makeCompletedEvent("failed", now.Add(3 * time.Minute)),
	}

	rate := paintress.SuccessRate(events)

	if rate != 0.5 {
		t.Errorf("SuccessRate = %f, want 0.5", rate)
	}
}
