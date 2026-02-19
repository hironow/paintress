package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runInitWithReader executes the init flow reading input from r.
// This is separated from runInit for testability.
func runInitWithReader(repoPath string, r io.Reader) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if err := validateContinent(absPath); err != nil {
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

// runInit is the CLI entry point for `paintress init <repo-path>`.
func runInit(repoPath string) {
	fmt.Println()
	fmt.Printf("%s╔══════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║          Paintress Init                      ║%s\n", colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
	fmt.Println()

	if err := runInitWithReader(repoPath, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
