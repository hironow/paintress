package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/port"
)

// registerExpeditionPolicies registers POLICY handlers for expedition events.
// See ADR S0014 (POLICY pattern) and S0018 (Event Storming alignment).
func registerExpeditionPolicies(engine *PolicyEngine, logger domain.Logger, notifier port.Notifier, metrics port.PolicyMetrics) {
	engine.Register(domain.EventExpeditionCompleted, func(ctx context.Context, event domain.Event) error {
		var data domain.ExpeditionCompletedData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Debug("policy: expedition completed parse error: %v", err)
			return nil
		}
		logger.Info("policy: expedition #%d completed (status=%s)", data.Expedition, data.Status)
		notifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := notifier.Notify(notifyCtx, "Paintress",
			fmt.Sprintf("Expedition #%d completed: %s", data.Expedition, data.Status)); err != nil {
			logger.Debug("policy: notify error: %v", err)
		}
		metrics.RecordPolicyEvent(ctx, "expedition.completed", "handled")
		return nil
	})

	// POLICY CONTRACT: observation-only — debug log + metrics.
	// Inbox arrival is a high-frequency event; expedition.completed
	// provides the user-facing summary notification.
	engine.Register(domain.EventInboxReceived, func(ctx context.Context, event domain.Event) error {
		logger.Debug("policy: inbox received (type=%s)", event.Type)
		metrics.RecordPolicyEvent(ctx, "inbox.received", "handled")
		return nil
	})

	// POLICY: gradient.changed → notify + metrics.
	engine.Register(domain.EventGradientChanged, func(ctx context.Context, event domain.Event) error {
		logger.Info("policy: gradient changed (type=%s)", event.Type)
		notifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := notifier.Notify(notifyCtx, "Paintress", "Gradient changed"); err != nil {
			logger.Debug("policy: notify error: %v", err)
		}
		metrics.RecordPolicyEvent(ctx, "gradient.changed", "handled")
		return nil
	})

	// POLICY: dmail.staged → notify + metrics.
	engine.Register(domain.EventDMailStaged, func(ctx context.Context, event domain.Event) error {
		logger.Info("policy: dmail staged (type=%s)", event.Type)
		notifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := notifier.Notify(notifyCtx, "Paintress", "D-Mail staged"); err != nil {
			logger.Debug("policy: notify error: %v", err)
		}
		metrics.RecordPolicyEvent(ctx, "dmail.staged", "handled")
		return nil
	})
}
