package usecase

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// FetchIssues loads project config and fetches Linear issues via the project ops interface.
func FetchIssues(ctx context.Context, absPath string, runner port.ClaudeRunner, stateFilter []string, ops port.ProjectOps) ([]domain.Issue, error) {
	cfg, err := ops.LoadProjectConfig(absPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if !cfg.HasTrackerTeam() {
		return nil, fmt.Errorf("linear.team not set in %s", domain.ProjectConfigPath(absPath))
	}

	workDir := filepath.Join(absPath, domain.StateDir, ".run")
	issues, err := ops.FetchIssuesViaMCP(ctx, runner, cfg.TrackerTeam(), cfg.TrackerProject(), workDir)
	if err != nil {
		return nil, err
	}

	filtered := domain.FilterIssuesByState(issues, stateFilter)
	domain.SortByPriority(filtered)
	return filtered, nil
}
