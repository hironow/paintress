package usecase

import (
	"context"
	"fmt"
	"os"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// FetchIssues loads project config and fetches Linear issues.
// This orchestrates session.LoadProjectConfig + session.FetchIssues + filtering.
func FetchIssues(ctx context.Context, absPath string, stateFilter []string) ([]domain.Issue, error) {
	cfg, err := session.LoadProjectConfig(absPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg.Linear.Team == "" {
		return nil, fmt.Errorf("linear.team not set in %s", domain.ProjectConfigPath(absPath))
	}

	apiKey := os.Getenv("LINEAR_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("LINEAR_API_KEY environment variable is required")
	}

	issues, err := session.FetchIssues(ctx, domain.LinearAPIEndpoint, apiKey, cfg.Linear.Team, cfg.Linear.Project, stateFilter)
	if err != nil {
		return nil, err
	}

	filtered := domain.FilterIssuesByState(issues, stateFilter)
	domain.SortByPriority(filtered)
	return filtered, nil
}
