package cmd

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// tryWriteHandover writes a handover document if the error is due to context
// cancellation and the outer (shutdown) context is still alive. Returns the
// original error unchanged.
func tryWriteHandover(ctx context.Context, err error, continent string, inProgress string, logger domain.Logger) error {
	if err == nil || !errors.Is(err, context.Canceled) {
		return err
	}
	outerCtx, ok := ctx.Value(domain.ShutdownKey).(context.Context)
	if !ok || outerCtx.Err() != nil {
		return err
	}

	hw := &session.FileHandoverWriter{}
	state := domain.HandoverState{
		Tool:       "paintress",
		Operation:  "expedition",
		Timestamp:  time.Now(),
		InProgress: inProgress,
	}
	stateDir := filepath.Join(continent, domain.StateDir)
	if hwErr := hw.WriteHandover(outerCtx, stateDir, state); hwErr != nil {
		if logger != nil {
			logger.Warn("handover write failed: %v", hwErr)
		}
	} else if logger != nil {
		logger.Info("Handover written to %s/handover.md", stateDir)
	}
	return err
}
