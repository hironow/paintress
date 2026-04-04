package policy

import (
	"github.com/hironow/paintress/internal/domain"
)

// ExpeditionTargetsFromWaves converts pending wave steps into expedition targets.
// Steps already completed are excluded. Preserves wave/step ordering.
func ExpeditionTargetsFromWaves(waves []domain.WaveProgress) []domain.ExpeditionTarget {
	var targets []domain.ExpeditionTarget
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
			targets = append(targets, domain.ExpeditionTarget{
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
