package session

import (
	"context"
	"fmt"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// stageEscalation creates and stages a feedback D-Mail for escalation when
// consecutive failures reach the threshold. Errors are logged but not
// propagated — escalation is best-effort observability.
func (p *Paintress) stageEscalation(expedition, failureCount int) {
	if p.outboxStore == nil {
		return
	}
	dm := domain.NewEscalationDMail(expedition, failureCount)
	if err := SendDMail(p.outboxStore, dm, p.Emitter); err != nil {
		p.Logger.Warn("escalation dmail: %v", err)
	}
}

// handleFeedbackAction dispatches a D-Mail based on its Action field.
// Actions: "retry" (with retry counting), "escalate", "resolve", or fallthrough.
func (p *Paintress) handleFeedbackAction(ctx context.Context, dm domain.DMail, workDir string, remaining time.Duration) {
	switch dm.Action {
	case "retry":
		if len(dm.Issues) == 0 {
			p.Logger.Warn("Retry action without issues, falling through: %s", dm.Name)
			p.runFollowUp(ctx, []domain.DMail{dm}, workDir, remaining)
			return
		}
		count := p.retryTracker.Track(dm.Issues)
		retryKey := domain.RetryKey(dm.Issues)

		if count > p.config.MaxRetries {
			p.Logger.Warn("Max retries (%d) reached for %s, escalating", p.config.MaxRetries, dm.Name)
			if err := p.handleEscalation(dm); err != nil {
				p.Logger.Error("escalation event lost: %v", err)
			}
			return
		}
		p.Logger.Info("Retry %d/%d for %s", count, p.config.MaxRetries, dm.Name)
		if err := p.Emitter.EmitRetryAttempted(retryKey, count, time.Now()); err != nil {
			p.Logger.Warn("retry event: %v", err)
		}
		p.runFollowUp(ctx, []domain.DMail{dm}, workDir, remaining)
	case "escalate":
		if err := p.handleEscalation(dm); err != nil {
			p.Logger.Error("escalation event lost: %v", err)
		}
	case "resolve":
		p.Logger.OK("Issue resolved per feedback: %s", dm.Name)
	default:
		p.runFollowUp(ctx, []domain.DMail{dm}, workDir, remaining)
	}
}

// handleEscalation logs and emits an escalation event for a D-Mail that
// requires human attention. Returns an error if the escalation event
// cannot be persisted — escalation events are critical and must be detectable.
func (p *Paintress) handleEscalation(dm domain.DMail) error {
	p.Logger.Warn("ESCALATION: %s requires human attention (issues: %v)", dm.Name, dm.Issues)
	if err := p.Emitter.EmitEscalated(dm.Name, dm.Issues, time.Now()); err != nil {
		p.Logger.Error("ESCALATION EVENT LOST: %s (issues: %v): %v", dm.Name, dm.Issues, err)
		return fmt.Errorf("escalation event: %w", err)
	}
	return nil
}
