package usecase

import (
	"fmt"
	"strings"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// RunInit validates the InitCommand and delegates project initialization
// to the InitRunner port.
func RunInit(cmd domain.InitCommand, runner port.InitRunner) error {
	if errs := cmd.Validate(); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("invalid init command: %s", strings.Join(msgs, "; "))
	}
	return runner.InitProject(cmd.RepoPath, cmd.Team, cmd.Project)
}
