package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func makeCompletedEvent(status string, t time.Time) domain.Event {
	data, _ := json.Marshal(domain.ExpeditionCompletedData{Status: status})
	return domain.Event{ID: "test", Type: domain.EventExpeditionCompleted, Timestamp: t, Data: data}
}

func TestSuccessRate_AllSuccess(t *testing.T) {
	now := time.Now()
	events := []domain.Event{
		makeCompletedEvent("success", now),
		makeCompletedEvent("success", now.Add(time.Minute)),
	}

	rate := domain.SuccessRate(events)

	if rate != 1.0 {
		t.Errorf("SuccessRate = %f, want 1.0", rate)
	}
}

func TestSuccessRate_AllFailed(t *testing.T) {
	now := time.Now()
	events := []domain.Event{
		makeCompletedEvent("failed", now),
		makeCompletedEvent("failed", now.Add(time.Minute)),
	}

	rate := domain.SuccessRate(events)

	if rate != 0.0 {
		t.Errorf("SuccessRate = %f, want 0.0", rate)
	}
}

func TestSuccessRate_Mixed(t *testing.T) {
	now := time.Now()
	events := []domain.Event{
		makeCompletedEvent("success", now),
		makeCompletedEvent("failed", now.Add(time.Minute)),
		makeCompletedEvent("success", now.Add(2*time.Minute)),
		makeCompletedEvent("skipped", now.Add(3*time.Minute)),
	}

	// 2 success out of 3 non-skipped = 0.666...
	rate := domain.SuccessRate(events)

	if rate < 0.66 || rate > 0.67 {
		t.Errorf("SuccessRate = %f, want ~0.666", rate)
	}
}

func TestSuccessRate_NoEvents(t *testing.T) {
	rate := domain.SuccessRate(nil)

	if rate != 0.0 {
		t.Errorf("SuccessRate = %f, want 0.0", rate)
	}
}

func TestSuccessRate_OnlySkipped(t *testing.T) {
	now := time.Now()
	events := []domain.Event{
		makeCompletedEvent("skipped", now),
		makeCompletedEvent("skipped", now.Add(time.Minute)),
	}

	// All skipped → no relevant events → 0
	rate := domain.SuccessRate(events)

	if rate != 0.0 {
		t.Errorf("SuccessRate = %f, want 0.0", rate)
	}
}

func TestSuccessRate_IgnoresNonCompletedEvents(t *testing.T) {
	now := time.Now()
	events := []domain.Event{
		{ID: "1", Type: domain.EventExpeditionStarted, Timestamp: now},
		makeCompletedEvent("success", now.Add(time.Minute)),
		{ID: "3", Type: domain.EventDMailStaged, Timestamp: now.Add(2 * time.Minute)},
		makeCompletedEvent("failed", now.Add(3*time.Minute)),
	}

	rate := domain.SuccessRate(events)

	if rate != 0.5 {
		t.Errorf("SuccessRate = %f, want 0.5", rate)
	}
}

func TestFormatSuccessRate_NoEvents(t *testing.T) {
	// given
	rate := 0.0
	success := 0
	total := 0

	// when
	result := domain.FormatSuccessRate(rate, success, total)

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
	result := domain.FormatSuccessRate(rate, success, total)

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
	result := domain.FormatSuccessRate(rate, success, total)

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
	result := domain.FormatSuccessRate(rate, success, total)

	// then
	expected := "0.0% (0/5)"
	if result != expected {
		t.Errorf("FormatSuccessRate = %q, want %q", result, expected)
	}
}
