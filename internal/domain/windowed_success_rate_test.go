package domain_test

import (
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// TestWindowedSuccessRate_EmptyEvents verifies zero rate for no events.
func TestWindowedSuccessRate_EmptyEvents(t *testing.T) {
	// when
	rate := domain.WindowedSuccessRate(nil, 5)

	// then
	if rate != 0.0 {
		t.Errorf("WindowedSuccessRate(nil, 5) = %f, want 0.0", rate)
	}
}

// TestWindowedSuccessRate_WindowLargerThanEvents verifies full window when events < window size.
func TestWindowedSuccessRate_WindowLargerThanEvents(t *testing.T) {
	// given: 2 events, window=5 → uses all 2
	now := time.Now()
	events := []domain.Event{
		makeCompletedEvent("success", now),
		makeCompletedEvent("success", now.Add(time.Minute)),
	}

	// when
	rate := domain.WindowedSuccessRate(events, 5)

	// then: 2/2 = 1.0
	if rate != 1.0 {
		t.Errorf("WindowedSuccessRate = %f, want 1.0", rate)
	}
}

// TestWindowedSuccessRate_WindowSlicesLastN verifies only the last N events are counted.
func TestWindowedSuccessRate_WindowSlicesLastN(t *testing.T) {
	// given: 5 events, first 3 are failures, last 2 are successes; window=2
	now := time.Now()
	events := []domain.Event{
		makeCompletedEvent("failed", now),
		makeCompletedEvent("failed", now.Add(time.Minute)),
		makeCompletedEvent("failed", now.Add(2*time.Minute)),
		makeCompletedEvent("success", now.Add(3*time.Minute)),
		makeCompletedEvent("success", now.Add(4*time.Minute)),
	}

	// when: window=2 should only see the last 2 events (both success)
	rate := domain.WindowedSuccessRate(events, 2)

	// then
	if rate != 1.0 {
		t.Errorf("WindowedSuccessRate(window=2) = %f, want 1.0 (last 2 are success)", rate)
	}
}

// TestWindowedSuccessRate_SkippedExcluded verifies skipped events are excluded from window.
func TestWindowedSuccessRate_SkippedExcluded(t *testing.T) {
	// given: 3 events: success, skipped, failed; window=3
	now := time.Now()
	events := []domain.Event{
		makeCompletedEvent("success", now),
		makeCompletedEvent("skipped", now.Add(time.Minute)),
		makeCompletedEvent("failed", now.Add(2*time.Minute)),
	}

	// when: skipped is excluded → 1 success / 2 non-skipped = 0.5
	rate := domain.WindowedSuccessRate(events, 10)

	// then
	if rate < 0.49 || rate > 0.51 {
		t.Errorf("WindowedSuccessRate (skipped excluded) = %f, want ~0.5", rate)
	}
}

// TestSuccessRateTrend_Improving verifies improving trend is detected.
func TestSuccessRateTrend_Improving(t *testing.T) {
	// given: earlyRate < recentRate
	earlyRate := 0.3
	recentRate := 0.8

	// when
	trend := domain.SuccessRateTrend(earlyRate, recentRate)

	// then
	if trend != domain.TrendImproving {
		t.Errorf("SuccessRateTrend(0.3, 0.8) = %v, want TrendImproving", trend)
	}
}

// TestSuccessRateTrend_Declining verifies declining trend is detected.
func TestSuccessRateTrend_Declining(t *testing.T) {
	// given: earlyRate > recentRate
	earlyRate := 0.8
	recentRate := 0.3

	// when
	trend := domain.SuccessRateTrend(earlyRate, recentRate)

	// then
	if trend != domain.TrendDeclining {
		t.Errorf("SuccessRateTrend(0.8, 0.3) = %v, want TrendDeclining", trend)
	}
}

// TestSuccessRateTrend_Stable verifies stable trend when rates are close.
func TestSuccessRateTrend_Stable(t *testing.T) {
	// given: rates are essentially the same
	earlyRate := 0.6
	recentRate := 0.62

	// when
	trend := domain.SuccessRateTrend(earlyRate, recentRate)

	// then
	if trend != domain.TrendStable {
		t.Errorf("SuccessRateTrend(0.6, 0.62) = %v, want TrendStable", trend)
	}
}

// TestDetectSuccessRateTrend_RecentVsEarlyWindow verifies full pipeline.
func TestDetectSuccessRateTrend_RecentVsEarlyWindow(t *testing.T) {
	// given: 10 events — first 5 are failures, last 5 are successes
	now := time.Now()
	events := make([]domain.Event, 10)
	for i := range 5 {
		events[i] = makeCompletedEvent("failed", now.Add(time.Duration(i)*time.Minute))
	}
	for i := range 5 {
		events[5+i] = makeCompletedEvent("success", now.Add(time.Duration(5+i)*time.Minute))
	}

	// when: window=5, compare recent 5 vs early 5
	trend := domain.DetectSuccessRateTrend(events, 5)

	// then: recent is all success, early is all failure → improving
	if trend != domain.TrendImproving {
		t.Errorf("DetectSuccessRateTrend = %v, want TrendImproving", trend)
	}
}

// TestDetectSuccessRateTrend_InsufficientEvents verifies stable when not enough data.
func TestDetectSuccessRateTrend_InsufficientEvents(t *testing.T) {
	// given: fewer events than 2 windows
	now := time.Now()
	events := []domain.Event{
		makeCompletedEvent("success", now),
		makeCompletedEvent("success", now.Add(time.Minute)),
	}

	// when: window=5 but only 2 events total
	trend := domain.DetectSuccessRateTrend(events, 5)

	// then: not enough data → stable
	if trend != domain.TrendStable {
		t.Errorf("DetectSuccessRateTrend (insufficient) = %v, want TrendStable", trend)
	}
}
