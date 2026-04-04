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
