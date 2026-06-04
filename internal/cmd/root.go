package cmd

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/spf13/cobra"
)

// Version, Commit, and Date are set at build time via -ldflags.
var (
	Version = "dev"
	Commit  = "dev"
	Date    = "dev"
)

type loggerKeyType struct{}

var loggerKey loggerKeyType

// shutdownTracer holds the OTel tracer shutdown function registered by
// PersistentPreRunE. cobra.OnFinalize calls it after Execute completes.
var (
	shutdownTracer func(context.Context) error
	shutdownMeter  func(context.Context) error
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
		Short:   "Expedition journal/gradient MCP data plane",
		Long:    "The Paintress — exposes the expedition journal/gradient data plane over MCP for a Claude Code interactive session (jun15 MCP pivot).",
		Version: Version,
		// Silence usage on RunE errors (cobra prints usage by default on error)
		SilenceUsage:  true,
		SilenceErrors: true, // nosemgrep: cobra-silence-errors-without-output — main.go handles error output [permanent]
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			applyOtelEnv(domain.StateDir)
			noColor := mustBool(cmd, "no-color")
			if noColor {
				_ = os.Setenv("NO_COLOR", "1")
			}
			verbose := mustBool(cmd, "verbose")
			out := cmd.ErrOrStderr()
			quiet := mustBool(cmd, "quiet")
			if quiet {
				out = io.Discard
			}
			logger := platform.NewLogger(out, verbose)
			outputFmt := mustString(cmd, "output")
			if outputFmt != "json" {
				logger.Header("paintress", Version)
				logger.Section(cmd.Name())
			}
			ctx := context.WithValue(cmd.Context(), loggerKey, logger)
			shutdownTracer = initTracer("paintress", Version)
			shutdownMeter = initMeter("paintress", Version)
			spanCtx := startRootSpan(ctx, cmd.Name())
			cmd.SetContext(spanCtx)

			return nil
		},
	}

	finalizerOnce.Do(func() {
		cobra.OnFinalize(func() {
			endRootSpan()
			if shutdownMeter != nil {
				_ = shutdownMeter(context.Background())
			}
			if shutdownTracer != nil {
				_ = shutdownTracer(context.Background())
			}
		})
	})

	rootCmd.PersistentFlags().StringP("config", "c", "", "Config file path")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output (respects NO_COLOR env)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all stderr output")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format: text, json")
	rootCmd.PersistentFlags().StringP("lang", "l", "", "Output language: en, ja (default from config)")
	rootCmd.PersistentFlags().Bool("linear", false, "Use Linear MCP for issue tracking (default: wave-centric mode)")

	rootCmd.AddCommand(
		newInitCommand(),
		newDoctorCommand(),
		newStatusCommand(),
		newArchivePruneCommand(),
		newCleanCommand(),
		newRebuildCommand(),
		newVersionCommand(),
		newUpdateCommand(),
		newMCPCommand(),
		newConfigCommand(),
		newMCPConfigCommand(),
		newSessionsCommand(),
		newDeadLettersCommand(),
	)

	return rootCmd
}

// loggerFrom extracts the domain.Logger from the cobra command context.
// Falls back to a stderr logger if PersistentPreRunE was not executed (e.g., in tests).
func loggerFrom(cmd *cobra.Command) domain.Logger {
	if l, ok := cmd.Context().Value(loggerKey).(domain.Logger); ok {
		return l
	}
	return platform.NewLogger(cmd.ErrOrStderr(), false)
}
