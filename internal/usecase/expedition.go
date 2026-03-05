package usecase

import (
	"context"
	"fmt"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// RunExpeditions validates the RunExpeditionCommand, then delegates to the expedition runner.
// Creates a PolicyEngine and injects it via SetDispatcher.
func RunExpeditions(ctx context.Context, cmd domain.RunExpeditionCommand,
	runner port.ExpeditionRunner, logger domain.Logger,
	notifier port.Notifier, metrics port.PolicyMetrics) (int, error) {
	if errs := cmd.Validate(); len(errs) > 0 {
		return 1, fmt.Errorf("command validation: %w", errs[0])
	}
	engine := NewPolicyEngine(logger)
	if metrics == nil {
		metrics = &port.NopPolicyMetrics{}
	}
	registerExpeditionPolicies(engine, logger, notifier, metrics)
	runner.SetDispatcher(engine)
	return runner.Run(ctx), nil
}
