package cmd

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/hironow/paintress"
	"github.com/spf13/cobra"
)

type loggerKeyType struct{}

var loggerKey loggerKeyType

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
			verbose, _ := cmd.Flags().GetBool("verbose")
			out := cmd.ErrOrStderr()
			if os.Getenv("PAINTRESS_QUIET") != "" {
				out = io.Discard
			}
			logger := paintress.NewLogger(out, verbose)
			ctx := context.WithValue(cmd.Context(), loggerKey, logger)
			shutdownTracer = initTracer("paintress", Version)
			spanCtx := startRootSpan(ctx, cmd.Name())
			cmd.SetContext(spanCtx)
			return nil
		},
	}

	finalizerOnce.Do(func() {
		cobra.OnFinalize(func() {
			endRootSpan()
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
		newCleanCommand(),
		newVersionCommand(),
		newUpdateCommand(),
	)

	return rootCmd
}

// loggerFrom extracts the *paintress.Logger from the cobra command context.
// Falls back to a stderr logger if PersistentPreRunE was not executed (e.g., in tests).
func loggerFrom(cmd *cobra.Command) *paintress.Logger {
	if l, ok := cmd.Context().Value(loggerKey).(*paintress.Logger); ok {
		return l
	}
	return paintress.NewLogger(cmd.ErrOrStderr(), false)
}
