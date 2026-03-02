package usecase

import (
	"context"

	"github.com/hironow/paintress"
)

// registerExpeditionPolicies registers POLICY handlers for expedition events.
// See ADR S0014 (POLICY pattern) and S0018 (Event Storming alignment).
func registerExpeditionPolicies(engine *PolicyEngine, logger *paintress.Logger) {
	engine.Register(paintress.EventExpeditionCompleted, func(_ context.Context, event paintress.Event) error {
		logger.Debug("policy: expedition completed (type=%s)", event.Type)
		return nil
	})

	engine.Register(paintress.EventInboxReceived, func(_ context.Context, event paintress.Event) error {
		logger.Debug("policy: inbox received (type=%s)", event.Type)
		return nil
	})

	engine.Register(paintress.EventGradientChanged, func(_ context.Context, event paintress.Event) error {
		logger.Debug("policy: gradient changed (type=%s)", event.Type)
		return nil
	})

	engine.Register(paintress.EventDMailStaged, func(_ context.Context, event paintress.Event) error {
		logger.Debug("policy: dmail staged (type=%s)", event.Type)
		return nil
	})
}
