package session

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hironow/paintress"
)

// makeMCPListCmd creates the exec.Cmd for `claude mcp list`.
// Package-level variable for test injection.
var makeMCPListCmd = func(ctx context.Context, claudeCmd string) *exec.Cmd {
	return exec.CommandContext(ctx, claudeCmd, "mcp", "list")
}

// RunDoctor checks all required external commands and returns the results.
// claudeCmd is the configured Claude CLI command name (e.g. "claude", "cc-p").
// continent is the optional .expedition/ root directory. When non-empty,
// additional checks for .expedition/ structure and config.yaml are included
// as warnings (not required).
func RunDoctor(claudeCmd string, continent string) []paintress.DoctorCheck {
	commands := []struct {
		name     string
		required bool
	}{
		{"git", true},
		{claudeCmd, true},
		{"gh", true},
		{"docker", false},
	}

	checks := make([]paintress.DoctorCheck, 0, len(commands)+6)
	for _, cmd := range commands {
		check := paintress.DoctorCheck{
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

	if strings.TrimSpace(continent) != "" {
		checks = append(checks, checkContinent(continent))
		checks = append(checks, checkConfig(continent))
		checks = append(checks, checkGitRepo(continent))
		checks = append(checks, checkWritability(continent))
		checks = append(checks, checkSkills(continent))
		checks = append(checks, checkEventStore(continent))
	}

	return checks
}

// checkContinent verifies the .expedition/ directory structure exists.
// Returns a Warning-level check (Required=false).
func checkContinent(continent string) paintress.DoctorCheck {
	check := paintress.DoctorCheck{
		Name:     "continent",
		Required: false,
	}

	expeditionDir := filepath.Join(continent, ".expedition")
	info, err := os.Stat(expeditionDir)
	if err != nil || !info.IsDir() {
		check.Version = ".expedition/ not found"
		return check
	}

	requiredDirs := []string{"journal", ".run", "inbox", "outbox", "archive"}
	var missing []string
	for _, sub := range requiredDirs {
		d := filepath.Join(expeditionDir, sub)
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
			missing = append(missing, sub)
		}
	}

	if len(missing) > 0 {
		check.Version = "missing: " + strings.Join(missing, ", ")
		return check
	}

	check.OK = true
	check.Path = expeditionDir
	check.Version = "structure OK"
	return check
}

// checkGitRepo verifies that the continent directory is inside a git repository.
// Returns a Warning-level check (Required=false).
func checkGitRepo(continent string) paintress.DoctorCheck {
	check := paintress.DoctorCheck{
		Name:     "git-repo",
		Required: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	cmd := exec.CommandContext(ctx, "git", "-C", continent, "rev-parse", "--git-dir")
	out, err := cmd.Output()
	cancel()
	if err != nil {
		check.Version = "not a git repository"
		return check
	}

	check.OK = true
	check.Path = strings.TrimSpace(string(out))
	check.Version = "git repo OK"
	return check
}

// checkWritability verifies that the .expedition/ directory is writable.
// Creates and removes a probe file to test write access.
// Returns a Warning-level check (Required=false).
func checkWritability(continent string) paintress.DoctorCheck {
	check := paintress.DoctorCheck{
		Name:     "writable",
		Required: false,
	}

	expeditionDir := filepath.Join(continent, ".expedition")
	probe := filepath.Join(expeditionDir, ".doctor-probe")
	if err := os.WriteFile(probe, []byte("probe"), 0644); err != nil {
		check.Version = "not writable: " + err.Error()
		return check
	}
	os.Remove(probe)

	check.OK = true
	check.Path = expeditionDir
	check.Version = "writable OK"
	return check
}

// checkConfig verifies that config.yaml exists and can be loaded.
// Returns a Warning-level check (Required=false).
func checkConfig(continent string) paintress.DoctorCheck {
	check := paintress.DoctorCheck{
		Name:     "config",
		Required: false,
	}

	configPath := paintress.ProjectConfigPath(continent)
	if _, err := os.Stat(configPath); err != nil {
		check.Version = "config.yaml not found"
		return check
	}

	cfg, err := LoadProjectConfig(continent)
	if err != nil {
		check.Version = "load error: " + err.Error()
		return check
	}

	check.OK = true
	check.Path = configPath
	if cfg.Linear.Team != "" {
		check.Version = "team=" + cfg.Linear.Team
	} else {
		check.Version = "loaded OK"
	}
	return check
}

// checkSkills verifies that SKILL.md files exist and contain dmail-schema-version.
// Searches .expedition/skills/*/SKILL.md for valid skill definitions.
// Returns a Warning-level check (Required=false).
func checkSkills(continent string) paintress.DoctorCheck {
	check := paintress.DoctorCheck{
		Name:     "skills",
		Required: false,
	}

	skillsDir := filepath.Join(continent, ".expedition", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		check.Version = "skills/ not found"
		return check
	}

	var found, valid int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}
		found++
		if strings.Contains(string(data), "dmail-schema-version:") {
			valid++
		}
	}

	if found == 0 {
		check.Version = "no SKILL.md files found"
		return check
	}

	if valid < found {
		check.Version = fmt.Sprintf("%d/%d skills missing dmail-schema-version", found-valid, found)
		return check
	}

	check.OK = true
	check.Path = skillsDir
	check.Version = fmt.Sprintf("%d skills OK", found)
	return check
}

// checkEventStore verifies that event JSONL files are parseable.
// Scans .expedition/events/*.jsonl and validates each line is valid JSON.
// Returns a Warning-level check (Required=false).
func checkEventStore(continent string) paintress.DoctorCheck {
	check := paintress.DoctorCheck{
		Name:     "events",
		Required: false,
	}

	eventsDir := filepath.Join(continent, ".expedition", "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		check.Version = "events/ not found"
		return check
	}

	var files, lines int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		f, err := os.Open(filepath.Join(eventsDir, entry.Name()))
		if err != nil {
			check.Version = "read error: " + err.Error()
			return check
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if !json.Valid([]byte(line)) {
				f.Close()
				check.Version = fmt.Sprintf("corrupt JSON in %s", entry.Name())
				return check
			}
			lines++
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			check.Version = "scan error: " + err.Error()
			return check
		}
		files++
	}

	if files == 0 {
		check.Version = "no .jsonl files found"
		return check
	}

	check.OK = true
	check.Path = eventsDir
	check.Version = fmt.Sprintf("%d files, %d events OK", files, lines)
	return check
}

// checkClaudeAuth determines if the Claude CLI is authenticated by
// interpreting the result of running `claude mcp list`. A successful
// command execution (no error) indicates the CLI is authenticated.
func checkClaudeAuth(mcpOutput string, mcpErr error) paintress.DoctorCheck {
	check := paintress.DoctorCheck{
		Name:     "claude-auth",
		Required: false,
	}
	if mcpErr != nil {
		check.Version = "not authenticated: " + mcpErr.Error()
		return check
	}
	check.OK = true
	check.Version = "authenticated"
	return check
}

// checkLinearMCP parses `claude mcp list` output for Linear MCP connection.
// Looks for a line containing "linear", "✓", and "connected" (case-insensitive).
// Requires "✓" to avoid false positives from "disconnected" or "not connected".
func checkLinearMCP(mcpOutput string, mcpErr error) paintress.DoctorCheck {
	check := paintress.DoctorCheck{
		Name:     "linear-mcp",
		Required: false,
	}
	if mcpErr != nil {
		check.Version = "skipped (claude not available)"
		return check
	}
	output := strings.ToLower(mcpOutput)
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "linear") &&
			strings.Contains(line, "✓") &&
			strings.Contains(line, "connected") {
			check.OK = true
			check.Version = "Linear MCP connected"
			return check
		}
	}
	check.Version = "Linear MCP not found or not connected"
	return check
}
