package domain

// ExpeditionTarget is the atomic work unit for an expedition.
// In Wave mode: a step (or whole wave if single-step).
// In Linear mode: wraps a Linear issue.
type ExpeditionTarget struct {
	ID          string // claim key: "waveID:stepID" or "waveID" (single-step) or issue ID (linear)
	WaveID      string // wave identifier (empty in linear mode)
	StepID      string // step identifier (empty if targeting whole wave or linear mode)
	Title       string
	Description string
	Acceptance  string // completion criteria (empty in linear mode)
}

// IsWaveTarget returns true when this target is from a wave specification.
func (t ExpeditionTarget) IsWaveTarget() bool {
	return t.WaveID != ""
}

// ExpeditionTargetsFromWaves converts pending wave steps into expedition targets.
// Steps already completed are excluded. Preserves wave/step ordering.
func ExpeditionTargetsFromWaves(waves []WaveProgress) []ExpeditionTarget {
	var targets []ExpeditionTarget
	for _, w := range waves {
		pending := w.PendingSteps()
		if len(pending) == 0 {
			continue
		}
		for _, s := range pending {
			id := w.WaveID + ":" + s.StepID
			if s.StepID == w.WaveID {
				// Single-step wave: use wave ID as claim key
				id = w.WaveID
			}
			targets = append(targets, ExpeditionTarget{
				ID:          id,
				WaveID:      w.WaveID,
				StepID:      s.StepID,
				Title:       s.Title,
				Description: w.Title, // wave title as context
				Acceptance:  s.Acceptance,
			})
		}
	}
	return targets
}
