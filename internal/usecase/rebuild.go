package usecase

import (
	"fmt"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// Rebuild replays events to regenerate projection state.
// Validates the RebuildCommand and performs the rebuild.
func Rebuild(cmd domain.RebuildCommand, events domain.EventStore, projector domain.EventApplier, logger domain.Logger) error {
	if errs := cmd.Validate(); len(errs) > 0 {
		return fmt.Errorf("command validation: %w", errs[0])
	}

	allEvents, err := events.LoadAll()
	if err != nil {
		return fmt.Errorf("load events: %w", err)
	}

	logger.Info("rebuilding projections from %d event(s)", len(allEvents))

	if err := projector.Rebuild(allEvents); err != nil {
		return fmt.Errorf("rebuild: %w", err)
	}

	logger.OK("rebuild complete")
	return nil
}

// RebuildFromDir constructs event store and projection applier from a repo directory,
// then replays events to regenerate projections.
// This is the cmd-facing entry point that eliminates session imports from cmd.
func RebuildFromDir(cmd domain.RebuildCommand, logger domain.Logger) error {
	eventsDir := domain.EventsDir(cmd.RepoPath)
	eventStore := session.NewEventStore(eventsDir)
	projector := session.NewProjectionApplier()

	if err := Rebuild(cmd, eventStore, projector, logger); err != nil {
		return err
	}

	state := projector.State()
	logger.OK("projections: %d expeditions (%d ok, %d failed, %d skipped), gradient=%d",
		state.TotalExpeditions, state.Succeeded, state.Failed, state.Skipped, state.GradientLevel)

	return nil
}
