package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/spf13/cobra"
)

func newCleanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [path]",
		Short: "Remove state directory (.expedition/)",
		Long:  "Delete the .expedition/ directory to reset to a clean state. Use 'paintress init' to reinitialize.",
		Example: `  # Clean the current directory
  paintress clean

  # Clean a specific project
  paintress clean /path/to/repo

  # Skip confirmation prompt
  paintress clean --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath, err := resolveTargetDir(args)
			if err != nil {
				return err
			}
			stateDir := filepath.Join(repoPath, domain.StateDir)

			info, err := os.Stat(stateDir)
			if err != nil || !info.IsDir() {
				fmt.Fprintf(cmd.ErrOrStderr(), "Nothing to clean at %s\n", repoPath)
				return nil
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Fprintf(cmd.ErrOrStderr(), "The following will be deleted:\n  %s/\n\nDelete? [y/N]: ", stateDir)
				var answer string
				fmt.Fscanln(cmd.InOrStdin(), &answer)
				if answer != "y" && answer != "Y" {
					fmt.Fprintf(cmd.ErrOrStderr(), "Aborted.\n")
					return nil
				}
			}

			if err := os.RemoveAll(stateDir); err != nil {
				return fmt.Errorf("remove %s: %w", stateDir, err)
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Cleaned %s\n", stateDir)
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	return cmd
}
