package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
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
	shutdownTracer  func(context.Context) error
	shutdownMeter   func(context.Context) error
	sharedStreamBus interface{ Close() }
	finalizerOnce   sync.Once
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
		SilenceUsage:  true,
		SilenceErrors: true, // nosemgrep: cobra-silence-errors-without-output — main.go handles error output [permanent]
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			applyOtelEnv(domain.StateDir)
			noColor, _ := cmd.Flags().GetBool("no-color")
			if noColor {
				os.Setenv("NO_COLOR", "1")
			}
			verbose, _ := cmd.Flags().GetBool("verbose")
			out := cmd.ErrOrStderr()
			quiet, _ := cmd.Flags().GetBool("quiet")
			if quiet {
				out = io.Discard
			}
			logger := platform.NewLogger(out, verbose)
			outputFmt, _ := cmd.Flags().GetString("output")
			if outputFmt != "json" {
				logger.Header("paintress", Version)
				logger.Section(cmd.Name())
			}
			ctx := context.WithValue(cmd.Context(), loggerKey, logger)
			shutdownTracer = initTracer("paintress", Version)
			shutdownMeter = initMeter("paintress", Version)
			spanCtx := startRootSpan(ctx, cmd.Name())
			cmd.SetContext(spanCtx)

			// StreamBus: process-wide live session event bus.
			streamBus := platform.NewInProcessSessionBus()
			session.SetStreamBus(streamBus)
			sharedStreamBus = streamBus

			// Production subscriber: bridge stream events to logger.
			sub := streamBus.Subscribe(64)
			go func() {
				for ev := range sub.C() {
					logger.Debug("stream: %s [%s] session=%s", ev.Type, ev.Tool, ev.SessionID)
				}
			}()

			return nil
		},
	}

	finalizerOnce.Do(func() {
		cobra.OnFinalize(func() {
			endRootSpan()
			if sharedStreamBus != nil {
				sharedStreamBus.Close()
			}
			if shutdownMeter != nil {
				shutdownMeter(context.Background())
			}
			if shutdownTracer != nil {
				shutdownTracer(context.Background())
			}
		})
	})

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output (respects NO_COLOR env)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all stderr output")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format: text, json")
	rootCmd.PersistentFlags().StringP("lang", "l", "", "Output language: en, ja (default from config)")
	rootCmd.PersistentFlags().Bool("linear", false, "Use Linear MCP for issue tracking (default: wave-centric mode)")

	rootCmd.AddCommand(
		newRunCommand(),
		newInitCommand(),
		newDoctorCommand(),
		newStatusCommand(),
		newIssuesCommand(),
		newArchivePruneCommand(),
		newCleanCommand(),
		newRebuildCommand(),
		newVersionCommand(),
		newUpdateCmd(),
		newConfigCommand(),
		newMCPConfigCommand(),
		newSessionsCmd(),
		newDeadLettersCommand(),
	)

	return rootCmd
}

// resolveRepoPath returns the absolute path from the first arg or cwd.
// Validates that the path exists and is a directory.
func resolveRepoPath(args []string) (string, error) {
	if len(args) > 0 {
		abs, err := filepath.Abs(args[0])
		if err != nil {
			return "", fmt.Errorf("resolve path: %w", err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", fmt.Errorf("path not found: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("not a directory: %s", abs)
		}
		return abs, nil
	}
	return os.Getwd()
}

// loggerFrom extracts the domain.Logger from the cobra command context.
// Falls back to a stderr logger if PersistentPreRunE was not executed (e.g., in tests).
func loggerFrom(cmd *cobra.Command) domain.Logger {
	if l, ok := cmd.Context().Value(loggerKey).(domain.Logger); ok {
		return l
	}
	return platform.NewLogger(cmd.ErrOrStderr(), false)
}
