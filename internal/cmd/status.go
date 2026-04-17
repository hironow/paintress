package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/hironow/paintress/internal/session"
	"github.com/spf13/cobra"
)

// newStatusCommand creates the status subcommand that displays operational status.
func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status [path]",
		Short: "Show paintress operational status",
		Long: `Display operational status including expedition history, success rate,
gradient level, and pending d-mail counts.

Output goes to stdout by default (human-readable text).
Use -o json for machine-readable JSON output to stdout.`,
		Example: `  # Show status for current directory
  paintress status

  # Show status for a specific project
  paintress status /path/to/repo

  # JSON output for scripting
  paintress status -o json /path/to/repo`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := resolveTargetDir(args)
			if err != nil {
				return err
			}

			report := session.Status(cmd.Context(), baseDir, loggerFrom(cmd))

			outputFmt := mustString(cmd, "output")
			if outputFmt == "json" {
				data, jsonErr := json.Marshal(report)
				if jsonErr != nil {
					return fmt.Errorf("marshal status: %w", jsonErr)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			// Text output to stdout (human-readable, per S0027)
			fmt.Fprint(cmd.OutOrStdout(), report.FormatText())
			return nil
		},
	}
}
