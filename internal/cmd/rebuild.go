package cmd

import (
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/spf13/cobra"
)

func newRebuildCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rebuild [path]",
		Short: "Rebuild projections from event store",
		Long: `Replays all events from .expedition/events/ to regenerate materialized projection state from scratch.

If path is omitted, the current working directory is used.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoPath(args)
			if err != nil {
				return err
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
