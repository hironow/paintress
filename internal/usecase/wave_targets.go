package usecase

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// waveTargetProvider implements port.TargetProvider for wave-centric mode.
// Primary path: reads step completion state from the event-sourced Read Model.
// Fallback path: reads inbox D-Mails when no spec.registered events exist yet
// (legacy/migration scenario).
type waveTargetProvider struct {
	stepProgress port.StepProgressReader
	inbox        port.InboxReader // fallback for legacy environments
}

// NewWaveTargetProvider creates a TargetProvider for wave mode.
// stepProgress is the primary source (event-sourced Read Model).
// inbox is used as fallback when no spec.registered events exist.
func NewWaveTargetProvider(stepProgress port.StepProgressReader, inbox port.InboxReader) port.TargetProvider {
	return &waveTargetProvider{stepProgress: stepProgress, inbox: inbox}
}

func (p *waveTargetProvider) FetchTargets(ctx context.Context) ([]domain.ExpeditionTarget, error) {
	progress, err := p.stepProgress.ReadStepProgress(ctx)
	if err != nil {
		return nil, err
	}
	if !progress.HasWaves() {
		// Fallback: no spec.registered events yet — use inbox (legacy path).
		// This is transitional: once spec.registered events accumulate,
		// this path is never reached.
		return p.legacyFetchFromInbox(ctx)
	}
	targets := progress.PendingTargets()

	// Append waveless feedback D-Mails as standalone targets.
	// These are reactive tasks (e.g., conflict resolution from amadeus)
	// that don't belong to any wave but are still paintress's responsibility.
	standalone, sErr := p.collectWavelessFeedbackTargets(ctx)
	if sErr != nil {
		return targets, nil // non-fatal: return wave targets even if inbox read fails
	}
	targets = append(targets, standalone...)
	return targets, nil
}

// collectWavelessFeedbackTargets reads inbox for implementation-feedback D-Mails
// without a Wave reference and converts them to standalone expedition targets.
// These represent reactive tasks like conflict resolution that don't belong to any wave.
func (p *waveTargetProvider) collectWavelessFeedbackTargets(ctx context.Context) ([]domain.ExpeditionTarget, error) {
	if p.inbox == nil {
		return nil, nil
	}
	dmails, err := p.inbox.ReadInboxDMails(ctx)
	if err != nil {
		return nil, err
	}
	var targets []domain.ExpeditionTarget
	for _, dm := range dmails {
		if dm.Wave != nil {
			continue // has Wave reference — handled by wave projection
		}
		if dm.Kind != "implementation-feedback" && dm.Kind != "design-feedback" {
			continue // only feedback D-Mails become standalone targets
		}
		targets = append(targets, domain.ExpeditionTarget{
			ID:          dm.Name,
			Title:       dm.Description,
			Description: dm.Body,
		})
	}
	return targets, nil
}

// legacyFetchFromInbox reads inbox D-Mails and projects wave state.
// Used only when no spec.registered events exist in the event store.
func (p *waveTargetProvider) legacyFetchFromInbox(ctx context.Context) ([]domain.ExpeditionTarget, error) {
	if p.inbox == nil {
		return nil, nil
	}
	inboxDMails, err := p.inbox.ReadInboxDMails(ctx)
	if err != nil {
		return nil, err
	}
	waves := domain.ProjectWaveState(inboxDMails)
	return domain.ExpeditionTargetsFromWaves(waves), nil
}
