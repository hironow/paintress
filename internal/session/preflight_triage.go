package session

import (
	"context"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// triagePreFlightDMails processes action fields on inbox D-Mails before
// expedition creation. Returns the subset that should be passed to the
// expedition prompt. Actions "escalate" and "resolve" are consumed (removed
// from the result); "retry" is kept unless the retry count exceeds the
// configured maximum, in which case it is escalated and removed.
func (p *Paintress) triagePreFlightDMails(ctx context.Context, dmails []domain.DMail) []domain.DMail {
	result := make([]domain.DMail, 0, len(dmails))

	for _, dm := range dmails {
		switch dm.Action {
		case "escalate":
			if err := p.handleEscalation(dm); err != nil {
				p.Logger.Error("pre-flight escalation event lost: %v", err)
			}
			if archErr := ArchiveInboxDMail(ctx, p.config.Continent, dm.Name, p.Emitter); archErr != nil {
				p.Logger.Warn("pre-flight archive %s: %v", dm.Name, archErr)
			}
			// consumed — do not pass to expedition

		case "resolve":
			p.Logger.OK("Issue resolved per feedback: %s", dm.Name)
			if archErr := ArchiveInboxDMail(ctx, p.config.Continent, dm.Name, p.Emitter); archErr != nil {
				p.Logger.Warn("pre-flight archive %s: %v", dm.Name, archErr)
			}
			// consumed — do not pass to expedition

		case "retry":
			if len(dm.Issues) == 0 {
				// retry without issues: pass through to expedition
				result = append(result, dm)
				continue
			}
			count := p.retryTracker.Track(dm.Issues)
			retryKey := domain.RetryKey(dm.Issues)

			if count > p.config.MaxRetries {
				p.Logger.Warn("Max retries (%d) reached for %s, escalating", p.config.MaxRetries, dm.Name)
				if err := p.handleEscalation(dm); err != nil {
					p.Logger.Error("pre-flight escalation event lost: %v", err)
				}
				if archErr := ArchiveInboxDMail(ctx, p.config.Continent, dm.Name, p.Emitter); archErr != nil {
					p.Logger.Warn("pre-flight archive %s: %v", dm.Name, archErr)
				}
				continue
			}
			p.Logger.Info("Retry %d/%d for %s", count, p.config.MaxRetries, dm.Name)
			if err := p.Emitter.EmitRetryAttempted(retryKey, count, time.Now()); err != nil {
				p.Logger.Warn("retry event: %v", err)
			}
			result = append(result, dm)

		default:
			// no action or unknown action: pass through
			result = append(result, dm)
		}
	}

	return result
}
