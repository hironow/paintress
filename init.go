package paintress

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// RunInitWithReader executes the init flow reading input from r.
// This is separated from runInit for testability.
func RunInitWithReader(repoPath string, r io.Reader) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if err := ValidateContinent(absPath); err != nil {
		return fmt.Errorf("continent validation: %w", err)
	}

	scanner := bufio.NewScanner(r)

	fmt.Print("Linear team key (e.g. MY): ")
	var team string
	if scanner.Scan() {
		team = strings.TrimSpace(scanner.Text())
	}
	if team == "" {
		return fmt.Errorf("team key is required")
	}

	fmt.Print("Linear project name (optional, press Enter to skip): ")
	var project string
	if scanner.Scan() {
		project = strings.TrimSpace(scanner.Text())
	}

	cfg := &ProjectConfig{
		Linear: LinearConfig{
			Team:    team,
			Project: project,
		},
	}

	if err := SaveProjectConfig(absPath, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\nConfig saved to %s\n", ProjectConfigPath(absPath))
	fmt.Printf("  Linear team:    %s\n", team)
	if project != "" {
		fmt.Printf("  Linear project: %s\n", project)
	}
	return nil
}
