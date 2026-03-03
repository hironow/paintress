package usecase

import (
	"fmt"
	"io"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// InitProject validates the InitCommand, then delegates to session.InitProject.
func InitProject(cmd domain.InitCommand, w io.Writer) error {
	if errs := cmd.Validate(); len(errs) > 0 {
		return fmt.Errorf("command validation: %w", errs[0])
	}
	return session.InitProject(cmd.RepoPath, cmd.Team, cmd.Project, w)
}
