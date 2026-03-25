package usecase

import (
	"context"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// feedbackActionHandler implements port.FeedbackActionHandler using usecase logic.
type feedbackActionHandler struct {
	maxRetries int
	tracker    *domain.RetryTracker
	emitter    port.ExpeditionEventEmitter
	followUp   port.FollowUpRunner
	logger     domain.Logger
}

// NewFeedbackActionHandler creates a FeedbackActionHandler with the given dependencies.
func NewFeedbackActionHandler(
	maxRetries int, tracker *domain.RetryTracker,
	emitter port.ExpeditionEventEmitter, followUp port.FollowUpRunner, logger domain.Logger,
) port.FeedbackActionHandler {
	return &feedbackActionHandler{
		maxRetries: maxRetries,
		tracker:    tracker,
		emitter:    emitter,
		followUp:   followUp,
		logger:     logger,
	}
}

// HandleFeedbackAction dispatches a D-Mail based on its Action field.
// Actions: "retry" (with retry counting), "escalate", "resolve", or fallthrough.
func (h *feedbackActionHandler) HandleFeedbackAction(ctx context.Context, dm domain.DMail, workDir string, remaining time.Duration) {
	switch dm.Action {
	case "retry":
		if len(dm.Issues) == 0 {
			h.logger.Warn("Retry action without issues, falling through: %s", dm.Name)
			h.followUp.RunFollowUp(ctx, []domain.DMail{dm}, workDir, remaining)
			return
		}
		count := h.tracker.Track(dm.Issues)
		retryKey := domain.RetryKey(dm.Issues)

		if count > h.maxRetries {
			h.logger.Warn("Max retries (%d) reached for %s, escalating", h.maxRetries, dm.Name)
			if err := handleEscalation(dm, h.emitter, h.logger); err != nil {
				h.logger.Error("escalation event lost: %v", err)
			}
			return
		}
		h.logger.Info("Retry %d/%d for %s", count, h.maxRetries, dm.Name)
		if err := h.emitter.EmitRetryAttempted(retryKey, count, time.Now()); err != nil {
			h.logger.Warn("retry event: %v", err)
		}
		h.followUp.RunFollowUp(ctx, []domain.DMail{dm}, workDir, remaining)
	case "escalate":
		if err := handleEscalation(dm, h.emitter, h.logger); err != nil {
			h.logger.Error("escalation event lost: %v", err)
		}
	case "resolve":
		h.logger.OK("Issue resolved per feedback: %s", dm.Name)
		if err := h.emitter.EmitResolved(dm.Name, dm.Issues, time.Now()); err != nil {
			h.logger.Warn("resolved event: %v", err)
		}
	default:
		h.followUp.RunFollowUp(ctx, []domain.DMail{dm}, workDir, remaining)
	}
}
