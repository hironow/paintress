package session

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"

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

// RunInitWithReader executes the init flow reading input from r
// and writing prompts/status to w. Used for interactive mode.
func RunInitWithReader(repoPath string, r io.Reader, w io.Writer) error {
	if w == nil {
		w = io.Discard
	}

	scanner := bufio.NewScanner(r)

	fmt.Fprint(w, "Linear team key (e.g. MY): ")
	var team string
	if scanner.Scan() {
		team = strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	if team == "" {
		return fmt.Errorf("team key is required")
	}

	fmt.Fprint(w, "Linear project name (optional, press Enter to skip): ")
	var project string
	if scanner.Scan() {
		project = strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	return InitProject(repoPath, team, project, w)
}
