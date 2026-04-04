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
