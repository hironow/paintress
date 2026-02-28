package paintress

import "encoding/json"

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
