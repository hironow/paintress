package usecase

import (
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// RunInit delegates project initialization to the InitRunner port.
// The InitCommand is already valid by construction (parse-don't-validate).
func RunInit(cmd domain.InitCommand, runner port.InitRunner) error {
	return runner.InitProject(cmd.RepoPath().String(), cmd.Team().String(), cmd.Project().String())
}
