package usecase

import (
	"encoding/json"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// RunDoctor checks all required external commands and returns the results.
// claudeCmd is the configured Claude CLI command name (e.g. "claude", "cc-p").
// continent is the optional .expedition/ root directory.
func RunDoctor(claudeCmd string, continent string) []domain.DoctorCheck {
	return session.RunDoctor(claudeCmd, continent)
}

// ComputeSuccessRate loads all events from the event store and computes
// success rate metrics. Returns nil metrics when no events exist or loading fails.
func ComputeSuccessRate(repoPath string) *domain.DoctorMetrics {
	stateDir := filepath.Join(repoPath, ".expedition")
	store := session.NewEventStore(stateDir)
	events, err := store.LoadAll()
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
