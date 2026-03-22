package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

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

// ExpeditionDurations calculates the duration of each completed (non-skipped) expedition
// by pairing EventExpeditionStarted with EventExpeditionCompleted events by expedition number.
// Skipped expeditions are excluded from the result.
func ExpeditionDurations(events []Event) []time.Duration {
	// Build a map from expedition number to start timestamp.
	startTimes := make(map[int]time.Time)
	for _, ev := range events {
		if ev.Type != EventExpeditionStarted {
			continue
		}
		var data ExpeditionStartedData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			continue
		}
		startTimes[data.Expedition] = ev.Timestamp
	}

	var durations []time.Duration
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
		startTime, ok := startTimes[data.Expedition]
		if !ok {
			continue
		}
		durations = append(durations, ev.Timestamp.Sub(startTime))
	}
	return durations
}

// DurationPercentiles calculates p50, p90, and p99 percentiles from a slice of durations.
// Returns (0, 0, 0) for an empty input.
func DurationPercentiles(durations []time.Duration) (p50, p90, p99 time.Duration) {
	if len(durations) == 0 {
		return 0, 0, 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	percentile := func(pct float64) time.Duration {
		idx := int(float64(len(sorted)-1) * pct)
		return sorted[idx]
	}

	return percentile(0.50), percentile(0.90), percentile(0.99)
}

// SuccessRateTrendType represents the direction of success rate change.
type SuccessRateTrendType string

const (
	// TrendImproving indicates the recent success rate is significantly higher than early.
	TrendImproving SuccessRateTrendType = "improving"
	// TrendDeclining indicates the recent success rate is significantly lower than early.
	TrendDeclining SuccessRateTrendType = "declining"
	// TrendStable indicates the success rate has not changed significantly.
	TrendStable SuccessRateTrendType = "stable"
)

// trendThreshold is the minimum rate difference to be considered a trend change.
const trendThreshold = 0.10

// WindowedSuccessRate computes the success rate over the last windowSize completed
// (non-skipped) events. If windowSize >= total events, all events are used.
func WindowedSuccessRate(events []Event, windowSize int) float64 {
	// Collect only non-skipped completed events.
	var relevant []Event
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
		relevant = append(relevant, ev)
	}

	if len(relevant) == 0 {
		return 0.0
	}

	// Take the last windowSize events.
	if windowSize > 0 && len(relevant) > windowSize {
		relevant = relevant[len(relevant)-windowSize:]
	}

	return SuccessRate(relevant)
}

// SuccessRateTrend compares earlyRate and recentRate and returns the trend direction.
// A change of less than trendThreshold (10%) is considered stable.
func SuccessRateTrend(earlyRate, recentRate float64) SuccessRateTrendType {
	diff := recentRate - earlyRate
	if diff > trendThreshold {
		return TrendImproving
	}
	if diff < -trendThreshold {
		return TrendDeclining
	}
	return TrendStable
}

// DetectSuccessRateTrend compares the success rate of the most recent windowSize events
// against the windowSize events before that to detect improvement or decline.
// Returns TrendStable when there are fewer events than 2×windowSize.
func DetectSuccessRateTrend(events []Event, windowSize int) SuccessRateTrendType {
	// Collect only non-skipped completed events.
	var relevant []Event
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
		relevant = append(relevant, ev)
	}

	if len(relevant) < 2*windowSize {
		return TrendStable
	}

	earlyWindow := relevant[:windowSize]
	recentWindow := relevant[len(relevant)-windowSize:]

	earlyRate := SuccessRate(earlyWindow)
	recentRate := SuccessRate(recentWindow)

	return SuccessRateTrend(earlyRate, recentRate)
}
