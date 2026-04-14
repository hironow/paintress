package usecase

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness"
	"github.com/hironow/paintress/internal/usecase/port"
)

// PrepareExpeditionRunner wires the expedition runner with EventEmitter, PolicyEngine,
// pre-flight triager, and feedback handler. Called by cmd (composition root).
// After calling this, the caller invokes runner.Run(ctx) directly.
func PrepareExpeditionRunner(ctx context.Context,
	runner port.ExpeditionRunner, eventStore port.EventStore, logger domain.Logger,
	notifier port.Notifier, metrics port.PolicyMetrics,
	archiver port.InboxArchiver, followUp port.FollowUpRunner,
	continent string, maxRetries int,
	mode domain.TrackingMode, targetProvider port.TargetProvider,
) {
	engine := NewPolicyEngine(logger)
	if metrics == nil {
		metrics = &port.NopPolicyMetrics{}
	}
	registerExpeditionPolicies(engine, logger, notifier, metrics)

	expeditionID := fmt.Sprintf("expedition-%d-%d", time.Now().UnixMilli(), os.Getpid())
	agg := domain.NewExpeditionAggregate()
	agg.SetExpeditionID(expeditionID)
	emitter := NewExpeditionEventEmitter(ctx, agg, eventStore, engine, logger, expeditionID)
	runner.SetEmitter(emitter)

	// Wire pre-flight triage if archiver is available
	tracker := harness.NewRetryTracker()
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

	// Wire tracking mode and target provider
	runner.SetTrackingMode(mode)
	if targetProvider != nil {
		runner.SetTargetProvider(targetProvider)
	}
}
