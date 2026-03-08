package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
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

  # Re-initialize (overwrite config, keep state)
  paintress init --force --team MY --project Hades /path/to/repo

  # Defaults only (no prompts)
  paintress init /path/to/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath := args[0]

			force, _ := cmd.Flags().GetBool("force")
			cfgPath := domain.ProjectConfigPath(repoPath)
			if _, err := os.Stat(cfgPath); err == nil && !force {
				return fmt.Errorf("%s already exists\nUse --force to overwrite", cfgPath)
			}

			team, _ := cmd.Flags().GetString("team")
			project, _ := cmd.Flags().GetString("project")
			rp, err := domain.NewRepoPath(repoPath)
			if err != nil {
				return err
			}
			initCmd := domain.NewInitCommand(rp, domain.NewTeam(team), domain.NewProject(project))
			runner := &session.InitAdapter{Stderr: cmd.ErrOrStderr()}
			if err := usecase.RunInit(initCmd, runner); err != nil {
				return err
			}

			otelBackend, _ := cmd.Flags().GetString("otel-backend")
			if otelBackend != "" {
				otelEntity, _ := cmd.Flags().GetString("otel-entity")
				otelProject, _ := cmd.Flags().GetString("otel-project")
				content, otelErr := platform.OtelEnvContent(otelBackend, otelEntity, otelProject)
				if otelErr != nil {
					return otelErr
				}
				stateDir := filepath.Join(repoPath, domain.StateDir)
				otelPath := filepath.Join(stateDir, ".otel.env")
				if err := os.WriteFile(otelPath, []byte(content), 0o644); err != nil {
					return fmt.Errorf("write .otel.env: %w", err)
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "OTel backend configured: %s → %s\n", otelBackend, otelPath)
			}

			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Overwrite existing config (preserves state directories)")
	cmd.Flags().String("team", "", "Linear team key (e.g. MY)")
	cmd.Flags().String("project", "", "Linear project name")
	cmd.Flags().String("otel-backend", "", "OTel backend: jaeger, weave")
	cmd.Flags().String("otel-entity", "", "Weave entity/team (required for weave)")
	cmd.Flags().String("otel-project", "", "Weave project (required for weave)")
	return cmd
}
