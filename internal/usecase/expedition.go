package usecase

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// RunExpeditions delegates to the expedition runner.
// Creates aggregate + EventEmitter with PolicyEngine as dispatcher, injects via SetEmitter.
// The RunExpeditionCommand is already valid by construction (parse-don't-validate).
func RunExpeditions(ctx context.Context, cmd domain.RunExpeditionCommand,
	runner port.ExpeditionRunner, eventStore port.EventStore, logger domain.Logger,
	notifier port.Notifier, metrics port.PolicyMetrics,
	archiver port.InboxArchiver, followUp port.FollowUpRunner,
	continent string, maxRetries int,
) (int, error) {
	engine := NewPolicyEngine(logger)
	if metrics == nil {
		metrics = &port.NopPolicyMetrics{}
	}
	registerExpeditionPolicies(engine, logger, notifier, metrics)

	agg := domain.NewExpeditionAggregate()
	emitter := NewExpeditionEventEmitter(agg, eventStore, engine, logger)
	runner.SetEmitter(emitter)

	// Wire pre-flight triage if archiver is available
	tracker := domain.NewRetryTracker()
	if archiver != nil {
		triager := NewPreFlightTriager(
			continent, maxRetries, tracker,
			archiver, emitter, logger,
		)
		runner.SetPreFlightTriager(triager)
	}

	// Wire feedback action handler if followUp runner is available
	if followUp != nil {
		handler := NewFeedbackActionHandler(maxRetries, tracker, emitter, followUp, logger)
		runner.SetFeedbackHandler(handler)
	}

	return runner.Run(ctx), nil
}
