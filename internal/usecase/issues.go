package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// FetchIssues loads project config and fetches Linear issues via the project ops interface.
func FetchIssues(ctx context.Context, absPath, claudeCmd string, stateFilter []string, ops port.ProjectOps) ([]domain.Issue, error) {
	cfg, err := ops.LoadProjectConfig(absPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg.Tracker.Team == "" {
		return nil, fmt.Errorf("linear.team not set in %s", domain.ProjectConfigPath(absPath))
	}

	workDir := filepath.Join(absPath, ".expedition", ".run")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}

	issues, err := ops.FetchIssuesViaMCP(ctx, claudeCmd, cfg.Tracker.Team, cfg.Tracker.Project, workDir)
	if err != nil {
		return nil, err
	}

	filtered := domain.FilterIssuesByState(issues, stateFilter)
	domain.SortByPriority(filtered)
	return filtered, nil
}
