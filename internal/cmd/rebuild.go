package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
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
			repoPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}

			logger := loggerFrom(cmd)

			return usecase.RebuildFromDir(domain.RebuildCommand{
				RepoPath: repoPath,
			}, logger)
		},
	}
}
