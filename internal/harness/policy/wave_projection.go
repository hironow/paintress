package policy

import (
	"github.com/hironow/paintress/internal/domain"
)

// ProjectWaveState builds wave progress from a chronological list of D-Mails.
// Pure function: no I/O. D-Mails are treated as an append-only event stream.
//
// Logic:
//   - kind=specification with wave field → defines wave and its steps
//   - kind=report with wave.step → marks step completed (status in body) or failed
//   - kind=*-feedback with wave.step → step is in-progress (awaiting retry)
//
// Steps not present in any report are "pending". A step with a later successful
// report overrides an earlier failure (retry semantics).
func ProjectWaveState(dmails []domain.DMail) []domain.WaveProgress {
	type waveEntry struct {
		id    string
		title string
		steps map[string]*domain.StepProgress // stepID → progress
		order []string                        // preserve step order
	}

	waves := make(map[string]*waveEntry) // waveID → entry
	var waveOrder []string

	// Pass 1: register all specifications (defines waves and steps).
	// Must run first so that reports/feedback find their parent wave
	// regardless of D-Mail ordering (archive sorts pt-report-* before spec-*).
	for _, dm := range dmails {
		if dm.Wave == nil || dm.Wave.ID == "" || dm.Kind != "specification" {
			continue
		}
		waveID := dm.Wave.ID
		if _, exists := waves[waveID]; exists {
			continue // first spec wins (immutable)
		}
		entry := &waveEntry{
			id:    waveID,
			title: dm.Description,
			steps: make(map[string]*domain.StepProgress),
		}
		if len(dm.Wave.Steps) == 0 {
			entry.steps[waveID] = &domain.StepProgress{
				StepID: waveID,
				Title:  dm.Description,
				Status: domain.StepPending,
			}
			entry.order = []string{waveID}
		} else {
			for _, s := range dm.Wave.Steps {
				entry.steps[s.ID] = &domain.StepProgress{
					StepID:     s.ID,
					Title:      s.Title,
					Status:     domain.StepPending,
					Acceptance: s.Acceptance,
				}
				entry.order = append(entry.order, s.ID)
			}
		}
		waves[waveID] = entry
		waveOrder = append(waveOrder, waveID)
	}

	// Pass 2: apply reports and feedback (updates step status).
	for _, dm := range dmails {
		if dm.Wave == nil || dm.Wave.ID == "" {
			continue
		}
		waveID := dm.Wave.ID

		switch dm.Kind {
		case "report":
			entry, ok := waves[waveID]
			if !ok {
				continue // report without spec — ignore
			}
			stepID := dm.Wave.Step
			if stepID == "" {
				stepID = waveID // single-step wave
			}
			sp, ok := entry.steps[stepID]
			if !ok {
				continue
			}
			sp.Attempts++
			if dm.Severity == "" || dm.Severity == "low" {
				sp.Status = domain.StepCompleted
			} else {
				sp.Status = domain.StepFailed
			}

		case "implementation-feedback", "design-feedback":
			entry, ok := waves[waveID]
			if !ok {
				continue
			}
			stepID := dm.Wave.Step
			if stepID == "" {
				stepID = waveID
			}
			sp, ok := entry.steps[stepID]
			if !ok {
				continue
			}
			if sp.Status == domain.StepFailed {
				sp.Status = domain.StepPending
			}
		}
	}

	// Build result in wave order
	result := make([]domain.WaveProgress, 0, len(waveOrder))
	for _, waveID := range waveOrder {
		entry := waves[waveID]
		wp := domain.WaveProgress{
			WaveID: entry.id,
			Title:  entry.title,
		}
		for _, stepID := range entry.order {
			wp.Steps = append(wp.Steps, *entry.steps[stepID])
		}
		result = append(result, wp)
	}
	return result
}
