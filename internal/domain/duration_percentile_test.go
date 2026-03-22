package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// makeStartedEvent creates an EventExpeditionStarted event for testing.
func makeStartedEvent(expedition int, t time.Time) domain.Event {
	data, _ := json.Marshal(domain.ExpeditionStartedData{Expedition: expedition})
	return domain.Event{
		ID:        "start",
		Type:      domain.EventExpeditionStarted,
		Timestamp: t,
		Data:      data,
	}
}

// makeCompletedEventWithExpedition creates an EventExpeditionCompleted with expedition number.
func makeCompletedEventWithExpedition(expedition int, status string, t time.Time) domain.Event {
	data, _ := json.Marshal(domain.ExpeditionCompletedData{Expedition: expedition, Status: status})
	return domain.Event{
		ID:        "end",
		Type:      domain.EventExpeditionCompleted,
		Timestamp: t,
		Data:      data,
	}
}

// TestExpeditionDurations_EmptyEvents verifies empty input returns empty durations.
func TestExpeditionDurations_EmptyEvents(t *testing.T) {
	// when
	durations := domain.ExpeditionDurations(nil)

	// then
	if len(durations) != 0 {
		t.Errorf("ExpeditionDurations(nil) = %v, want empty", durations)
	}
}

// TestExpeditionDurations_SingleCompleted verifies duration for one expedition.
func TestExpeditionDurations_SingleCompleted(t *testing.T) {
	// given: expedition 1 starts and completes 5 minutes later
	base := time.Now()
	events := []domain.Event{
		makeStartedEvent(1, base),
		makeCompletedEventWithExpedition(1, "success", base.Add(5*time.Minute)),
	}

	// when
	durations := domain.ExpeditionDurations(events)

	// then
	if len(durations) != 1 {
		t.Fatalf("ExpeditionDurations = %d durations, want 1", len(durations))
	}
	want := 5 * time.Minute
	if durations[0] < want-time.Second || durations[0] > want+time.Second {
		t.Errorf("duration = %v, want ~%v", durations[0], want)
	}
}

// TestExpeditionDurations_MultipleExpeditions verifies multiple expeditions.
func TestExpeditionDurations_MultipleExpeditions(t *testing.T) {
	// given: expedition 1 = 3min, expedition 2 = 7min
	base := time.Now()
	events := []domain.Event{
		makeStartedEvent(1, base),
		makeCompletedEventWithExpedition(1, "success", base.Add(3*time.Minute)),
		makeStartedEvent(2, base.Add(4*time.Minute)),
		makeCompletedEventWithExpedition(2, "success", base.Add(11*time.Minute)),
	}

	// when
	durations := domain.ExpeditionDurations(events)

	// then
	if len(durations) != 2 {
		t.Fatalf("ExpeditionDurations = %d durations, want 2", len(durations))
	}
}

// TestExpeditionDurations_SkippedExpeditionsExcluded verifies skipped expeditions are excluded.
func TestExpeditionDurations_SkippedExpeditionsExcluded(t *testing.T) {
	// given: expedition 1 = success (included), expedition 2 = skipped (excluded)
	base := time.Now()
	events := []domain.Event{
		makeStartedEvent(1, base),
		makeCompletedEventWithExpedition(1, "success", base.Add(5*time.Minute)),
		makeStartedEvent(2, base.Add(6*time.Minute)),
		makeCompletedEventWithExpedition(2, "skipped", base.Add(7*time.Minute)),
	}

	// when
	durations := domain.ExpeditionDurations(events)

	// then: only the success expedition is included
	if len(durations) != 1 {
		t.Fatalf("ExpeditionDurations = %d durations, want 1 (skipped excluded)", len(durations))
	}
}

// TestDurationPercentiles_P50P90P99 verifies percentile calculations.
func TestDurationPercentiles_P50P90P99(t *testing.T) {
	// given: 10 durations: 1 through 10 minutes
	durations := make([]time.Duration, 10)
	for i := range durations {
		durations[i] = time.Duration(i+1) * time.Minute
	}

	// when
	p50, p90, p99 := domain.DurationPercentiles(durations)

	// then: p50 ~ 5-6 min, p90 ~ 9-10 min, p99 ~ 10 min
	if p50 < 4*time.Minute || p50 > 6*time.Minute {
		t.Errorf("p50 = %v, want ~5min", p50)
	}
	if p90 < 8*time.Minute || p90 > 10*time.Minute {
		t.Errorf("p90 = %v, want ~9min", p90)
	}
	if p99 < 9*time.Minute {
		t.Errorf("p99 = %v, want >= 9min", p99)
	}
}

// TestDurationPercentiles_SingleDuration verifies single-element percentiles.
func TestDurationPercentiles_SingleDuration(t *testing.T) {
	// given
	durations := []time.Duration{10 * time.Minute}

	// when
	p50, p90, p99 := domain.DurationPercentiles(durations)

	// then: all percentiles equal the single value
	if p50 != 10*time.Minute {
		t.Errorf("p50 = %v, want 10min", p50)
	}
	if p90 != 10*time.Minute {
		t.Errorf("p90 = %v, want 10min", p90)
	}
	if p99 != 10*time.Minute {
		t.Errorf("p99 = %v, want 10min", p99)
	}
}

// TestDurationPercentiles_EmptyDurations verifies empty input returns zero.
func TestDurationPercentiles_EmptyDurations(t *testing.T) {
	// when
	p50, p90, p99 := domain.DurationPercentiles(nil)

	// then
	if p50 != 0 || p90 != 0 || p99 != 0 {
		t.Errorf("DurationPercentiles(nil) = %v/%v/%v, want 0/0/0", p50, p90, p99)
	}
}
