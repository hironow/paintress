package cmd

import (
	"fmt"
	"os"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <repo-path>",
		Short: "Initialize project configuration",
		Long: `Initialize a .expedition/ directory in the target repository.

Use --team and --project flags for non-interactive mode, or omit
flags for interactive prompts. This must be run once before
'paintress run' can operate on the repository.`,
		Example: `  # Non-interactive with flags
  paintress init --team MY --project Hades /path/to/repo

  # Defaults only (no prompts)
  paintress init /path/to/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath := args[0]

			cfgPath := domain.ProjectConfigPath(repoPath)
			if _, err := os.Stat(cfgPath); err == nil {
				return fmt.Errorf("%s already exists", cfgPath)
			}

			team, _ := cmd.Flags().GetString("team")
			project, _ := cmd.Flags().GetString("project")
			return session.InitProject(repoPath, team, project, cmd.ErrOrStderr())
		},
	}
	cmd.Flags().String("team", "", "Linear team key (e.g. MY)")
	cmd.Flags().String("project", "", "Linear project name")
	return cmd
}
