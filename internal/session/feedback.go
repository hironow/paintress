package session

import (
	"context"
	"fmt"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// stageEscalation creates and stages a feedback D-Mail for escalation when
// consecutive failures reach the threshold. Errors are logged but not
// propagated — escalation is best-effort observability.
func (p *Paintress) stageEscalation(ctx context.Context, expedition, failureCount int) {
	if p.outboxStore == nil {
		return
	}
	dm := domain.NewEscalationDMail(expedition, failureCount)
	domain.LogBanner(p.Logger, domain.BannerSend, string(dm.Kind), dm.Name, dm.Description)
	if err := SendDMail(ctx, p.outboxStore, dm, p.Emitter); err != nil {
		p.Logger.Warn("escalation dmail: %v", err)
	}
}

// handleFeedbackAction delegates to the injected FeedbackActionHandler port.
// Falls back to runFollowUp for all D-Mails when no handler is set.
func (p *Paintress) handleFeedbackAction(ctx context.Context, dm domain.DMail, workDir string, remaining time.Duration) {
	if p.feedbackHandler != nil {
		p.feedbackHandler.HandleFeedbackAction(ctx, dm, workDir, remaining)
		return
	}
	// Fallback: pass through to expedition
	p.runFollowUp(ctx, []domain.DMail{dm}, workDir, remaining)
}

// handleEscalation logs and emits an escalation event for a D-Mail that
// requires human attention. Used by stageEscalation.
func (p *Paintress) handleEscalation(dm domain.DMail) error {
	p.Logger.Warn("ESCALATION: %s requires human attention (issues: %v)", dm.Name, dm.Issues)
	if err := p.Emitter.EmitEscalated(dm.Name, dm.Issues, time.Now()); err != nil {
		p.Logger.Error("ESCALATION EVENT LOST: %s (issues: %v): %v", dm.Name, dm.Issues, err)
		return fmt.Errorf("escalation event: %w", err)
	}
	return nil
}

// SetFeedbackHandler injects the feedback action handler.
func (p *Paintress) SetFeedbackHandler(h port.FeedbackActionHandler) {
	p.feedbackHandler = h
}

// RunFollowUp implements port.FollowUpRunner by delegating to the session-layer runFollowUp.
func (p *Paintress) RunFollowUp(ctx context.Context, dmails []domain.DMail, workDir string, remaining time.Duration) {
	p.runFollowUp(ctx, dmails, workDir, remaining)
}
