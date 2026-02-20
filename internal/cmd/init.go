package cmd

import (
	"fmt"
	"os"

	"github.com/hironow/paintress"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init <repo-path>",
		Short: "Initialize project configuration",
		Long: `Initialize a .expedition/ directory in the target repository.

Creates config.yaml with Linear team key, project name, and
default expedition settings. This must be run once before
'paintress run' can operate on the repository.`,
		Example: `  # Initialize a new project
  paintress init /path/to/repo

  # Initialize and then run
  paintress init /path/to/repo && paintress run /path/to/repo`,
		Args: cobra.ExactArgs(1),
		RunE: runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	repoPath := args[0]

	w := cmd.ErrOrStderr()
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s╔══════════════════════════════════════════════╗%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(w, "%s║          Paintress Init                      ║%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(w, "%s╚══════════════════════════════════════════════╝%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintln(w)

	if err := paintress.RunInitWithReader(repoPath, os.Stdin); err != nil {
		return err
	}
	return nil
}
