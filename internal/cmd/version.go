package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Show the paintress version string set at build time via ldflags.",
		Example: `  # Show version
  paintress version`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "paintress %s\n", Version)
			return nil
		},
	}
}
