package session

import (
	"github.com/hironow/paintress/internal/usecase/port"
)

// InitAdapter implements port.InitRunner by delegating to session.InitProject.
type InitAdapter struct {
	LastResult *InitResult // populated after InitProject for display by cmd layer
}

// InitProject creates the project configuration and directory structure.
func (a *InitAdapter) InitProject(baseDir string, opts ...port.InitOption) ([]string, error) {
	cfg := port.ApplyInitOptions(opts...)
	result, err := InitProject(baseDir, cfg.Team, cfg.Project)
	a.LastResult = result
	if err != nil {
		return nil, err
	}
	return result.Warnings(), nil
}
