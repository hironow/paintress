package session

import (
	"fmt"

	"github.com/hironow/paintress/internal/platform"
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

	// Claude Code entry skill materialization (refs issue 0032 D5):
	// .claude/skills/expedition-next makes /expedition-next
	// auto-discovered by a bare `claude` session in this project.
	if err := InstallClaudeSkills(baseDir, platform.ClaudeSkillsFS, nil); err != nil {
		result.Add(".claude/skills", InitWarning, fmt.Sprintf("failed to install claude skills: %v", err))
	} else {
		result.Add(".claude/skills/expedition-next/", InitCreated, "")
	}
	return result.Warnings(), nil
}
