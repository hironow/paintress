package usecase

import (
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// RunInit delegates project initialization to the InitRunner port.
// The InitCommand is already valid by construction (parse-don't-validate).
func RunInit(cmd domain.InitCommand, runner port.InitRunner) ([]string, error) {
	return runner.InitProject(
		cmd.RepoPath().String(),
		port.WithTeam(cmd.Team().String()),
		port.WithProject(cmd.Project().String()),
	)
}
