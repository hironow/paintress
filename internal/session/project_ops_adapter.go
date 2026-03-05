package session

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
)

type projectOps struct{}

// NewProjectOps returns a port.ProjectOps implementation.
func NewProjectOps() *projectOps {
	return &projectOps{}
}

func (*projectOps) LoadProjectConfig(absPath string) (*domain.ProjectConfig, error) {
	return LoadProjectConfig(absPath)
}

func (*projectOps) FetchIssuesViaMCP(ctx context.Context, claudeCmd, team, project, workDir string) ([]domain.Issue, error) {
	return FetchIssuesViaMCP(ctx, claudeCmd, team, project, workDir)
}
