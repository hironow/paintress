package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/hironow/paintress"
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// RunExpeditions validates the RunExpeditionCommand, then delegates to session.
// Creates a PolicyEngine and injects it into the Paintress session.
func RunExpeditions(ctx context.Context, cmd domain.RunExpeditionCommand, cfg paintress.Config, logger *domain.Logger, dataOut io.Writer, stdinIn io.Reader, eventStore domain.EventStore) (int, error) {
	if errs := cmd.Validate(); len(errs) > 0 {
		return 1, fmt.Errorf("command validation: %w", errs[0])
	}
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger)
	p := session.NewPaintress(cfg, logger, dataOut, stdinIn, eventStore)
	p.Dispatcher = engine
	return p.Run(ctx), nil
}
