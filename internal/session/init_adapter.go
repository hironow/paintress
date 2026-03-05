package session

import "io"

// InitAdapter implements port.InitRunner by delegating to session.InitProject.
type InitAdapter struct {
	Stderr io.Writer
}

// InitProject creates the project configuration and directory structure.
func (a *InitAdapter) InitProject(repoPath, team, project string) error {
	w := a.Stderr
	if w == nil {
		w = io.Discard
	}
	return InitProject(repoPath, team, project, w)
}
