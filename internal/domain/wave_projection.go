package domain

// StepStatus represents the completion state of a wave step.
type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
)

// StepProgress tracks the status and attempt count for a single wave step.
type StepProgress struct {
	StepID     string
	Title      string
	Status     StepStatus
	Attempts   int
	Acceptance string
}

// WaveProgress tracks the overall progress of a wave across its steps.
type WaveProgress struct {
	WaveID string
	Title  string
	Steps  []StepProgress
}

// IsComplete returns true when all steps are completed.
func (w WaveProgress) IsComplete() bool {
	for _, s := range w.Steps {
		if s.Status != StepCompleted {
			return false
		}
	}
	return len(w.Steps) > 0
}

// PendingSteps returns steps that are not yet completed.
func (w WaveProgress) PendingSteps() []StepProgress {
	var pending []StepProgress
	for _, s := range w.Steps {
		if s.Status != StepCompleted {
			pending = append(pending, s)
		}
	}
	return pending
}

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
func ProjectWaveState(dmails []DMail) []WaveProgress {
	type waveEntry struct {
		id    string
		title string
		steps map[string]*StepProgress // stepID → progress
		order []string                 // preserve step order
	}

	waves := make(map[string]*waveEntry) // waveID → entry
	var waveOrder []string

	for _, dm := range dmails {
		if dm.Wave == nil {
			continue
		}

		waveID := dm.Wave.ID
		if waveID == "" {
			continue
		}

		switch dm.Kind {
		case "specification":
			if _, exists := waves[waveID]; exists {
				continue // first spec wins (immutable)
			}
			entry := &waveEntry{
				id:    waveID,
				title: dm.Description,
				steps: make(map[string]*StepProgress),
			}
			if len(dm.Wave.Steps) == 0 {
				// Wave without explicit steps: wave itself is the single step
				entry.steps[waveID] = &StepProgress{
					StepID: waveID,
					Title:  dm.Description,
					Status: StepPending,
				}
				entry.order = []string{waveID}
			} else {
				for _, s := range dm.Wave.Steps {
					entry.steps[s.ID] = &StepProgress{
						StepID:     s.ID,
						Title:      s.Title,
						Status:     StepPending,
						Acceptance: s.Acceptance,
					}
					entry.order = append(entry.order, s.ID)
				}
			}
			waves[waveID] = entry
			waveOrder = append(waveOrder, waveID)

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
			// Determine status from severity: low/empty = success, else failure
			if dm.Severity == "" || dm.Severity == "low" {
				sp.Status = StepCompleted
			} else {
				sp.Status = StepFailed
			}

		case "implementation-feedback", "design-feedback":
			// Feedback resets a failed step back to pending (awaiting retry)
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
			if sp.Status == StepFailed {
				sp.Status = StepPending
			}
		}
	}

	// Build result in wave order
	result := make([]WaveProgress, 0, len(waveOrder))
	for _, waveID := range waveOrder {
		entry := waves[waveID]
		wp := WaveProgress{
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
