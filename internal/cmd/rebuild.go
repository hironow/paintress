package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/spf13/cobra"
)

func newRebuildCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rebuild [repo-path]",
		Short: "Rebuild projections from event store",
		Long:  "Replays all events from .expedition/events/ to regenerate materialized projection state from scratch.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}

			logger := loggerFrom(cmd)
			stateDir := filepath.Join(repoRoot, domain.StateDir)
			eventStore := session.NewEventStore(stateDir, logger)
			projector := session.NewProjectionApplier()
			rp, rpErr := domain.NewRepoPath(repoRoot)
			if rpErr != nil {
				return rpErr
			}
			if err := usecase.Rebuild(domain.NewRebuildCommand(rp), eventStore, projector, logger); err != nil {
				return err
			}
			state := projector.State()
			logger.OK("projections: %d expeditions (%d ok, %d failed, %d skipped), gradient=%d",
				state.TotalExpeditions, state.Succeeded, state.Failed, state.Skipped, state.GradientLevel)
			return nil
		},
	}
}
