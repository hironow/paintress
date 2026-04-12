package session

import (
	"io"

	"github.com/hironow/paintress/internal/usecase/port"
)

// InitAdapter implements port.InitRunner by delegating to session.InitProject.
type InitAdapter struct {
	Stderr io.Writer
}

// InitProject creates the project configuration and directory structure.
func (a *InitAdapter) InitProject(baseDir string, opts ...port.InitOption) ([]string, error) {
	cfg := port.ApplyInitOptions(opts...)
	w := a.Stderr
	if w == nil {
		w = io.Discard
	}
	return nil, InitProject(baseDir, cfg.Team, cfg.Project, w)
}
