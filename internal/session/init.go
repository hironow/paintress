package session

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/hironow/paintress"
)

// InitProject creates .expedition/config.yaml using the provided values
// directly (no interactive prompts). Also runs ValidateContinent to ensure
// directory structure is set up.
func InitProject(repoPath, team, project string, w io.Writer) error {
	if w == nil {
		w = io.Discard
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if err := ValidateContinent(absPath); err != nil {
		return fmt.Errorf("continent validation: %w", err)
	}

	cfg := &paintress.ProjectConfig{
		Linear: paintress.LinearConfig{
			Team:    team,
			Project: project,
		},
	}

	if err := SaveProjectConfig(absPath, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(w, "\nConfig saved to %s\n", paintress.ProjectConfigPath(absPath))
	if team != "" {
		fmt.Fprintf(w, "  Linear team:    %s\n", team)
	}
	if project != "" {
		fmt.Fprintf(w, "  Linear project: %s\n", project)
	}
	return nil
}
