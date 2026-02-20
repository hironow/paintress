package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCommand creates and returns the root cobra command for paintress.
// Exported for testability (SetArgs/SetOut) and future docgen.
func NewRootCommand() *cobra.Command {
	cobra.EnableTraverseRunHooks = true

	rootCmd := &cobra.Command{
		Use:     "paintress",
		Short:   "Claude Code expedition orchestrator",
		Long:    "The Paintress — drives the Expedition loop for Claude Code.",
		Version: Version,
		// Silence usage on RunE errors (cobra prints usage by default on error)
		SilenceUsage: true,
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format: text, json")
	rootCmd.PersistentFlags().StringP("lang", "l", "en", "Output language: en, ja, fr")

	rootCmd.AddCommand(
		newRunCommand(),
		newInitCommand(),
		newDoctorCommand(),
		newIssuesCommand(),
		newArchivePruneCommand(),
		newVersionCommand(),
	)

	return rootCmd
}
