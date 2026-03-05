package usecase

import (
	"context"
	"fmt"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// RunExpeditions validates the RunExpeditionCommand, then delegates to the expedition runner.
// Creates aggregate + EventEmitter with PolicyEngine as dispatcher, injects via SetEmitter.
func RunExpeditions(ctx context.Context, cmd domain.RunExpeditionCommand,
	runner port.ExpeditionRunner, eventStore port.EventStore, logger domain.Logger,
	notifier port.Notifier, metrics port.PolicyMetrics) (int, error) {
	if errs := cmd.Validate(); len(errs) > 0 {
		return 1, fmt.Errorf("command validation: %w", errs[0])
	}
	engine := NewPolicyEngine(logger)
	if metrics == nil {
		metrics = &port.NopPolicyMetrics{}
	}
	registerExpeditionPolicies(engine, logger, notifier, metrics)

	agg := domain.NewExpeditionAggregate()
	emitter := NewExpeditionEventEmitter(agg, eventStore, engine, logger)
	runner.SetEmitter(emitter)
	return runner.Run(ctx), nil
}
