package cmd

import (
	"fmt"
	"strings"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/spf13/cobra"
)

func newIssuesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues [path]",
		Short: "List Linear issues via Claude MCP",
		Long: `Query Linear issues via Claude MCP tools for the configured team and project.

Reads the team/project from .expedition/config.yaml. Supports filtering
by issue state (e.g. todo, in-progress). Hyphens in state names are
converted to spaces automatically.

If path is omitted, the current working directory is used.`,
		Example: `  # List all issues (current directory)
  paintress issues

  # List all issues (explicit path)
  paintress issues /path/to/repo

  # Filter by state
  paintress issues --state todo,in-progress /path/to/repo

  # JSON output for scripting
  paintress issues -o json /path/to/repo`,
		Args: cobra.MaximumNArgs(1),
		RunE: runIssues,
	}

	cmd.Flags().StringP("state", "s", "", "Comma-separated state filter (e.g. todo,in-progress)")

	return cmd
}

func runIssues(cmd *cobra.Command, args []string) error {
	absPath, err := resolveTargetDir(args)
	if err != nil {
		return err
	}
	outputFmt := mustString(cmd, "output")
	stateRaw := mustString(cmd, "state")

	var stateFilter []string
	if stateRaw != "" {
		parts := strings.Split(stateRaw, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.ReplaceAll(p, "-", " ")
			if p != "" {
				stateFilter = append(stateFilter, p)
			}
		}
	}

	projectOps := session.NewProjectOps()
	claudeCmd := loadClaudeCmd(absPath)
	runner := &session.ClaudeAdapter{
		ClaudeCmd:  claudeCmd,
		TimeoutSec: 300,
		Logger:     &domain.NopLogger{},
		StreamBus:  session.SharedStreamBus(),
		ToolName:   "paintress",
	}
	issues, err := usecase.FetchIssues(cmd.Context(), absPath, runner, stateFilter, projectOps)
	if err != nil {
		return err
	}

	logger := loggerFrom(cmd)
	logger.Info("fetched %d issues from %s", len(issues), absPath)

	w := cmd.OutOrStdout()
	switch outputFmt {
	case "json":
		out, err := domain.FormatIssuesJSON(issues)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, out)
	case "text":
		fmt.Fprintln(w, domain.FormatIssuesTable(issues))
	default:
		out, err := domain.FormatIssuesJSONL(issues)
		if err != nil {
			return err
		}
		if out != "" {
			fmt.Fprintln(w, out)
		}
	}
	return nil
}
