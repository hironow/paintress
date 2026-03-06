package usecase

import (
	"encoding/json"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// ComputeSuccessRate loads all events and computes success rate metrics.
// Returns nil metrics when no events exist or loading fails.
func ComputeSuccessRate(eventStore port.EventStore) *domain.DoctorMetrics {
	events, _, err := eventStore.LoadAll()
	if err != nil || len(events) == 0 {
		return &domain.DoctorMetrics{SuccessRate: "no events"}
	}

	rate := domain.SuccessRate(events)
	var success, total int
	for _, ev := range events {
		if ev.Type != domain.EventExpeditionCompleted {
			continue
		}
		var data domain.ExpeditionCompletedData
		if json.Unmarshal(ev.Data, &data) != nil {
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
	return &domain.DoctorMetrics{
		SuccessRate: domain.FormatSuccessRate(rate, success, total),
	}
}
