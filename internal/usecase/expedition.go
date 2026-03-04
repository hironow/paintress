package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/port"
	"github.com/hironow/paintress/internal/session"
)

// RunExpeditions validates the RunExpeditionCommand, then delegates to session.
// Creates a PolicyEngine and injects it into the Paintress session.
func RunExpeditions(ctx context.Context, cmd domain.RunExpeditionCommand, cfg domain.Config, logger domain.Logger, dataOut io.Writer, errOut io.Writer, stdinIn io.Reader, eventStore domain.EventStore) (int, error) {
	if errs := cmd.Validate(); len(errs) > 0 {
		return 1, fmt.Errorf("command validation: %w", errs[0])
	}
	engine := NewPolicyEngine(logger)
	notifier := session.BuildNotifier(cfg.NotifyCmd)
	registerExpeditionPolicies(engine, logger, notifier, &port.NopPolicyMetrics{})
	p := session.NewPaintress(cfg, logger, dataOut, errOut, stdinIn, eventStore)
	p.Dispatcher = engine
	return p.Run(ctx), nil
}
