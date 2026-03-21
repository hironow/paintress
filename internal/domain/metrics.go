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
