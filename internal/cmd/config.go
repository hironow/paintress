package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/hironow/paintress/internal/session"
)

func newConfigCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "View or update paintress project configuration",
		Long:  "View or update the .expedition/config.yaml project configuration file.",
		Example: `  paintress config show /path/to/repo
  paintress config set tracker.team MY /path/to/repo`,
	}

	configCmd.AddCommand(newConfigShowCommand())
	configCmd.AddCommand(newConfigSetCommand())

	return configCmd
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [path]",
		Short: "Display project configuration",
		Long:  "Display the project configuration from .expedition/config.yaml.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath, err := resolveRepoPath(args)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}
			cfg, err := session.LoadProjectConfig(repoPath)
			if err != nil {
				return err
			}
			out, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value> [path]",
		Short: "Update a project configuration value",
		Long: `Update a configuration value in .expedition/config.yaml.

Supported keys:
  tracker.team     Linear team key (e.g. MY)
  tracker.project  Linear project name
  lang             Language (ja or en)`,
		Example: `  paintress config set tracker.team MY /path/to/repo
  paintress config set tracker.project "My Project" /path/to/repo
  paintress config set lang en`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			var repoPath string
			var err error
			if len(args) == 3 {
				repoPath, err = resolveRepoPath(args[2:])
			} else {
				repoPath, err = resolveRepoPath(nil)
			}
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}

			if err := session.UpdateProjectConfig(repoPath, key, value); err != nil {
				return err
			}

			logger := loggerFrom(cmd)
			logger.Info("Updated %s = %s", key, value)
			return nil
		},
	}
}
