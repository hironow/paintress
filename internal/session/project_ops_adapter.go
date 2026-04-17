package session

import (
	"context"
	"fmt"
	"os"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

type projectOps struct{}

// NewProjectOps returns a port.ProjectOps implementation.
func NewProjectOps() *projectOps {
	return &projectOps{}
}

func (*projectOps) LoadProjectConfig(absPath string) (*domain.ProjectConfig, error) {
	return LoadProjectConfig(absPath)
}

func (*projectOps) FetchIssuesViaMCP(ctx context.Context, runner port.ProviderRunner, team, project, workDir string) ([]domain.Issue, error) { // nosemgrep: domain-primitives.multiple-string-params-go -- team/project/workDir are semantically distinct [permanent]
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	return FetchIssuesViaMCP(ctx, runner, team, project, workDir)
}
