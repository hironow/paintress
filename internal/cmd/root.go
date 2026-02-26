package cmd

import (
	"context"
	"sync"

	"github.com/spf13/cobra"
)

// shutdownTracer holds the OTel tracer shutdown function registered by
// PersistentPreRunE. cobra.OnFinalize calls it after Execute completes.
var (
	shutdownTracer func(context.Context) error
	finalizerOnce  sync.Once
)

func init() {
	cobra.EnableTraverseRunHooks = true
}

// NewRootCommand creates and returns the root cobra command for paintress.
// Exported for testability (SetArgs/SetOut) and future docgen.
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "paintress",
		Short:   "Claude Code expedition orchestrator",
		Long:    "The Paintress — drives the Expedition loop for Claude Code.",
		Version: Version,
		// Silence usage on RunE errors (cobra prints usage by default on error)
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			shutdownTracer = initTracer("paintress", Version)
			return nil
		},
	}

	finalizerOnce.Do(func() {
		cobra.OnFinalize(func() {
			if shutdownTracer != nil {
				shutdownTracer(context.Background())
			}
		})
	})

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format: text, json")
	rootCmd.PersistentFlags().StringP("lang", "l", "en", "Output language: en, ja, fr")

	rootCmd.AddCommand(
		newRunCommand(),
		newInitCommand(),
		newDoctorCommand(),
		newIssuesCommand(),
		newArchivePruneCommand(),
		newVersionCommand(),
		newUpdateCommand(),
	)

	return rootCmd
}
