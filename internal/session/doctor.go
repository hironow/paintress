package session

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hironow/paintress"
)

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

	checks := make([]paintress.DoctorCheck, 0, len(commands)+2)
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
