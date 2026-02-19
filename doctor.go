package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DoctorCheck represents the result of checking a single external command.
type DoctorCheck struct {
	Name     string
	Required bool
	Path     string // exec.LookPath result
	Version  string // first line of --version output
	OK       bool
}

// RunDoctor checks all required external commands and returns the results.
// claudeCmd is the configured Claude CLI command name (e.g. "claude", "cc-p").
func RunDoctor(claudeCmd string) []DoctorCheck {
	commands := []struct {
		name     string
		required bool
	}{
		{"git", true},
		{claudeCmd, true},
		{"gh", true},
		{"docker", true},
	}

	checks := make([]DoctorCheck, 0, len(commands))
	for _, cmd := range commands {
		check := DoctorCheck{
			Name:     cmd.name,
			Required: cmd.required,
		}

		path, err := exec.LookPath(cmd.name)
		if err != nil {
			checks = append(checks, check)
			continue
		}

		check.Path = path
		check.OK = true

		// Try to get version (best-effort, 500ms timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		out, err := exec.CommandContext(ctx, path, "--version").Output()
		cancel()
		if err == nil {
			firstLine := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
			check.Version = firstLine
		}

		checks = append(checks, check)
	}

	return checks
}

// runDoctor is the CLI entry point for `paintress doctor`.
func runDoctor() {
	claudeCmd := defaultClaudeCmd
	checks := RunDoctor(claudeCmd)

	fmt.Println()
	fmt.Printf("%s╔══════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║          Paintress Doctor                    ║%s\n", colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
	fmt.Println()

	allOK := true
	for _, c := range checks {
		if c.OK {
			fmt.Printf("  %s✓%s  %-12s %s (%s)\n", colorGreen, colorReset, c.Name, c.Version, c.Path)
		} else {
			marker := "✗"
			color := colorRed
			label := "MISSING (required)"
			if !c.Required {
				label = "not found (optional)"
				color = colorYellow
			} else {
				allOK = false
			}
			fmt.Printf("  %s%s%s  %-12s %s\n", color, marker, colorReset, c.Name, label)
		}
	}
	fmt.Println()

	if !allOK {
		fmt.Fprintf(os.Stderr, "Some required commands are missing. Install them and try again.\n")
		os.Exit(1)
	}
	fmt.Println("All checks passed.")
}
