package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// preFlightTriager implements port.PreFlightTriager using usecase logic.
type preFlightTriager struct {
	continent  string
	maxRetries int
	tracker    *domain.RetryTracker
	archiver   port.InboxArchiver
	emitter    port.ExpeditionEventEmitter
	logger     domain.Logger
}

// NewPreFlightTriager creates a PreFlightTriager with the given dependencies.
func NewPreFlightTriager(
	continent string, maxRetries int, tracker *domain.RetryTracker,
	archiver port.InboxArchiver, emitter port.ExpeditionEventEmitter, logger domain.Logger,
) port.PreFlightTriager {
	return &preFlightTriager{
		continent:  continent,
		maxRetries: maxRetries,
		tracker:    tracker,
		archiver:   archiver,
		emitter:    emitter,
		logger:     logger,
	}
}

// TriagePreFlightDMails delegates to the core triage function.
func (t *preFlightTriager) TriagePreFlightDMails(ctx context.Context, dmails []domain.DMail) []domain.DMail {
	return triagePreFlightDMails(ctx, dmails, t.continent, t.maxRetries, t.tracker, t.archiver, t.emitter, t.logger)
}

// triagePreFlightDMails is the core triage logic, testable with explicit dependencies.
func triagePreFlightDMails(
	ctx context.Context,
	dmails []domain.DMail,
	continent string,
	maxRetries int,
	tracker *domain.RetryTracker,
	archiver port.InboxArchiver,
	emitter port.ExpeditionEventEmitter,
	logger domain.Logger,
) []domain.DMail {
	result := make([]domain.DMail, 0, len(dmails))

	for _, dm := range dmails {
		switch dm.Action {
		case "escalate":
			if err := handleEscalation(dm, emitter, logger); err != nil {
				logger.Error("pre-flight escalation event lost: %v", err)
			}
			if archErr := archiver.ArchiveInboxDMail(ctx, continent, dm.Name); archErr != nil {
				logger.Warn("pre-flight archive %s: %v", dm.Name, archErr)
			}
			// consumed — do not pass to expedition

		case "resolve":
			logger.OK("Issue resolved per feedback: %s", dm.Name)
			if err := emitter.EmitResolved(dm.Name, dm.Issues, time.Now()); err != nil {
				logger.Warn("pre-flight resolved event: %v", err)
			}
			if archErr := archiver.ArchiveInboxDMail(ctx, continent, dm.Name); archErr != nil {
				logger.Warn("pre-flight archive %s: %v", dm.Name, archErr)
			}
			// consumed — do not pass to expedition

		case "retry":
			if len(dm.Issues) == 0 {
				// retry without issues: pass through to expedition
				result = append(result, dm)
				continue
			}
			count := tracker.Track(dm.Issues)
			retryKey := domain.RetryKey(dm.Issues)

			if count > maxRetries {
				logger.Warn("Max retries (%d) reached for %s, escalating", maxRetries, dm.Name)
				if err := handleEscalation(dm, emitter, logger); err != nil {
					logger.Error("pre-flight escalation event lost: %v", err)
				}
				if archErr := archiver.ArchiveInboxDMail(ctx, continent, dm.Name); archErr != nil {
					logger.Warn("pre-flight archive %s: %v", dm.Name, archErr)
				}
				continue
			}
			logger.Info("Retry %d/%d for %s", count, maxRetries, dm.Name)
			if err := emitter.EmitRetryAttempted(retryKey, count, time.Now()); err != nil {
				logger.Warn("retry event: %v", err)
			}
			result = append(result, dm)

		default:
			// no action or unknown action: pass through
			result = append(result, dm)
		}
	}

	return result
}

// handleEscalation emits an escalation event and logs the warning.
func handleEscalation(dm domain.DMail, emitter port.ExpeditionEventEmitter, logger domain.Logger) error {
	logger.Warn("ESCALATION: %s requires human attention (issues: %v)", dm.Name, dm.Issues)
	if err := emitter.EmitEscalated(dm.Name, dm.Issues, time.Now()); err != nil {
		logger.Error("ESCALATION EVENT LOST: %s (issues: %v): %v", dm.Name, dm.Issues, err)
		return fmt.Errorf("escalation event: %w", err)
	}
	return nil
}
