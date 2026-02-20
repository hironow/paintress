package paintress

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

// DoctorCheck represents the result of checking a single external command.
type DoctorCheck struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Path     string `json:"path"`
	Version  string `json:"version"`
	OK       bool   `json:"ok"`
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
		{"docker", false},
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

// FormatDoctorJSON returns the checks as a JSON array string.
func FormatDoctorJSON(checks []DoctorCheck) (string, error) {
	data, err := json.Marshal(checks)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
