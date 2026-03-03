package usecase

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
)

// registerExpeditionPolicies registers POLICY handlers for expedition events.
// See ADR S0014 (POLICY pattern) and S0018 (Event Storming alignment).
func registerExpeditionPolicies(engine *PolicyEngine, logger *domain.Logger) {
	engine.Register(domain.EventExpeditionCompleted, func(_ context.Context, event domain.Event) error {
		logger.Debug("policy: expedition completed (type=%s)", event.Type)
		return nil
	})

	engine.Register(domain.EventInboxReceived, func(_ context.Context, event domain.Event) error {
		logger.Debug("policy: inbox received (type=%s)", event.Type)
		return nil
	})

	engine.Register(domain.EventGradientChanged, func(_ context.Context, event domain.Event) error {
		logger.Debug("policy: gradient changed (type=%s)", event.Type)
		return nil
	})

	engine.Register(domain.EventDMailStaged, func(_ context.Context, event domain.Event) error {
		logger.Debug("policy: dmail staged (type=%s)", event.Type)
		return nil
	})
}
