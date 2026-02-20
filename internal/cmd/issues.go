package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hironow/paintress"
	"github.com/spf13/cobra"
)

func newIssuesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues <repo-path>",
		Short: "List Linear issues",
		Long: `List Linear issues assigned to the configured team and project.

Reads the Linear API key from the LINEAR_API_KEY environment variable
and the team/project from .expedition/config.yaml. Supports filtering
by issue state (e.g. todo, in-progress). Hyphens in state names are
converted to spaces automatically.`,
		Example: `  # List all issues
  paintress issues /path/to/repo

  # Filter by state
  paintress issues --state todo,in-progress /path/to/repo

  # JSON output for scripting
  paintress issues -o json /path/to/repo`,
		Args: cobra.ExactArgs(1),
		RunE: runIssues,
	}

	cmd.Flags().String("state", "", "Comma-separated state filter (e.g. todo,in-progress)")

	return cmd
}

func runIssues(cmd *cobra.Command, args []string) error {
	repoPath := args[0]
	outputFmt, _ := cmd.Flags().GetString("output")
	stateRaw, _ := cmd.Flags().GetString("state")

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

	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	cfg, err := paintress.LoadProjectConfig(absPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.Linear.Team == "" {
		return fmt.Errorf("linear.team not set in %s\nRun 'paintress init %s' first", paintress.ProjectConfigPath(absPath), repoPath)
	}

	apiKey := os.Getenv("LINEAR_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("LINEAR_API_KEY environment variable is required")
	}

	issues, err := paintress.FetchIssues(context.Background(), paintress.LinearAPIEndpoint, apiKey, cfg.Linear.Team, cfg.Linear.Project, stateFilter)
	if err != nil {
		return err
	}

	issues = paintress.FilterIssuesByState(issues, stateFilter)

	paintress.LogInfo("fetched %d issues from %s", len(issues), cfg.Linear.Team)

	w := cmd.OutOrStdout()
	switch outputFmt {
	case "json":
		out, err := paintress.FormatIssuesJSON(issues)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, out)
	case "text":
		fmt.Fprintln(w, paintress.FormatIssuesTable(issues))
	default:
		out, err := paintress.FormatIssuesJSONL(issues)
		if err != nil {
			return err
		}
		if out != "" {
			fmt.Fprintln(w, out)
		}
	}
	return nil
}
