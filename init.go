package paintress

import (
	"bufio"
	"fmt"
	"io"
	"os"
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

	fmt.Fprint(os.Stderr, "Linear team key (e.g. MY): ")
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

	fmt.Fprint(os.Stderr, "Linear project name (optional, press Enter to skip): ")
	var project string
	if scanner.Scan() {
		project = strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
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

	fmt.Fprintf(os.Stderr, "\nConfig saved to %s\n", ProjectConfigPath(absPath))
	fmt.Fprintf(os.Stderr, "  Linear team:    %s\n", team)
	if project != "" {
		fmt.Fprintf(os.Stderr, "  Linear project: %s\n", project)
	}
	return nil
}
