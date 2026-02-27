package cmd

import (
	"fmt"
	"os"

	"github.com/hironow/paintress"
	"github.com/hironow/paintress/internal/session"
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

	cfgPath := paintress.ProjectConfigPath(repoPath)
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("%s already exists", cfgPath)
	}

	w := cmd.ErrOrStderr()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "╔══════════════════════════════════════════════╗")
	fmt.Fprintln(w, "║          Paintress Init                      ║")
	fmt.Fprintln(w, "╚══════════════════════════════════════════════╝")
	fmt.Fprintln(w)

	if err := session.RunInitWithReader(repoPath, cmd.InOrStdin(), cmd.ErrOrStderr()); err != nil {
		return err
	}
	return nil
}
