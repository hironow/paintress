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

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// newShellCmd creates an exec.Cmd via platform.NewShellCmd.
// Package-level variable for test injection. Used for --version and mcp list.
var newShellCmd = func(ctx context.Context, cmdLine string, args ...string) *exec.Cmd {
	return platform.NewShellCmd(ctx, cmdLine, args...)
}

// OverrideShellCmd replaces the command constructor for testing and returns a
// cleanup function.
func OverrideShellCmd(fn func(ctx context.Context, cmdLine string, args ...string) *exec.Cmd) func() {
	old := newShellCmd
	newShellCmd = fn
	return func() { newShellCmd = old }
}

// lookPath resolves the binary path for a command. Defaults to
// platform.LookPathShell. Injectable for testing tool-not-found scenarios.
var lookPath = platform.LookPathShell

// OverrideLookPath replaces the path lookup function for testing and returns a
// cleanup function.
func OverrideLookPath(fn func(cmd string) (string, error)) func() {
	old := lookPath
	lookPath = fn
	return func() { lookPath = old }
}

// RunDoctor checks all required external commands and returns the results.
// claudeCmd is the configured Claude CLI command name (e.g. "claude", "cc-p").
// continent is the optional .expedition/ root directory. When non-empty,
// additional checks for .expedition/ structure and config.yaml are included
// as warnings (not required).
func RunDoctor(claudeCmd string, continent string) []domain.DoctorCheck {
	commands := []struct {
		name     string
		required bool
	}{
		{"git", true},
		{claudeCmd, true},
		{"gh", true},
		{"docker", false},
	}

	checks := make([]domain.DoctorCheck, 0, len(commands)+8)
	for _, cmd := range commands {
		path, err := lookPath(cmd.name)
		if err != nil {
			status := domain.CheckWarn
			if cmd.required {
				status = domain.CheckFail
			}
			check := domain.DoctorCheck{
				Name:    cmd.name,
				Status:  status,
				Message: "command not found",
			}
			if cmd.required {
				check.Hint = fmt.Sprintf("install %s and ensure it is in PATH", cmd.name)
			}
			checks = append(checks, check)
			continue
		}

		// Try to get version (best-effort, 500ms timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		out, err := newShellCmd(ctx, cmd.name, "--version").Output()
		cancel()

		var versionStr string
		if err == nil {
			versionStr = strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		}

		check := domain.DoctorCheck{
			Name:    cmd.name,
			Status:  domain.CheckOK,
			Message: fmt.Sprintf("%s (%s)", path, versionStr),
		}
		checks = append(checks, check)
	}

	// gh auth scope check (requires gh binary to be found)
	ghOK := false
	for _, c := range checks {
		if c.Name == "gh" && c.Status == domain.CheckOK {
			ghOK = true
			break
		}
	}
	if ghOK {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, "gh", "auth", "status")
		out, err := cmd.CombinedOutput()
		cancel()
		checks = append(checks, checkGHScopes(string(out), err))
	}

	if strings.TrimSpace(continent) != "" {
		checks = append(checks, checkContinent(continent))
		checks = append(checks, checkConfig(continent))
		checks = append(checks, checkGitRepo(continent))
		checks = append(checks, checkGitRemote(continent))
		checks = append(checks, checkWritability(continent))
		checks = append(checks, checkSkills(continent))
		checks = append(checks, checkEventStore(continent))

		// External connectivity checks (skip if claude binary not found)
		claudeOK := false
		for _, c := range checks {
			if c.Name == claudeCmd && c.Status == domain.CheckOK {
				claudeOK = true
				break
			}
		}
		if !claudeOK {
			checks = append(checks, domain.DoctorCheck{
				Name:    "claude-auth",
				Status:  domain.CheckSkip,
				Message: "skipped (claude not available)",
			})
			checks = append(checks, domain.DoctorCheck{
				Name:    "linear-mcp",
				Status:  domain.CheckSkip,
				Message: "skipped (claude not available)",
			})
			checks = append(checks, domain.DoctorCheck{
				Name:    "claude-inference",
				Status:  domain.CheckSkip,
				Message: "skipped (claude not available)",
			})
			checks = append(checks, domain.DoctorCheck{
				Name:    "context-budget",
				Status:  domain.CheckSkip,
				Message: "skipped (claude not available)",
			})
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			cmd := newShellCmd(ctx, claudeCmd, "mcp", "list")
			out, err := cmd.Output()
			cancel()
			mcpOutput := string(out)
			authCheck := checkClaudeAuth(mcpOutput, err)
			checks = append(checks, authCheck)
			checks = append(checks, checkLinearMCP(mcpOutput, err))

			// claude-inference + context-budget: skip if auth failed
			if authCheck.Status != domain.CheckOK {
				checks = append(checks, domain.DoctorCheck{
					Name:    "claude-inference",
					Status:  domain.CheckSkip,
					Message: "skipped (auth failed)",
				})
				checks = append(checks, domain.DoctorCheck{
					Name:    "context-budget",
					Status:  domain.CheckSkip,
					Message: "skipped (auth failed)",
				})
			} else {
				inferCtx, inferCancel := context.WithTimeout(context.Background(), 60*time.Second)
				inferCmd := newShellCmd(inferCtx, claudeCmd, "--print", "--output-format", "stream-json", "--max-turns", "1", "1+1=")
				inferCmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
				inferOut, inferErr := inferCmd.Output()
				inferCancel()
				inferOutput := string(inferOut)
				checks = append(checks, checkClaudeInference(strings.TrimSpace(ExtractStreamResult(inferOutput)), inferErr))
				checks = append(checks, CheckContextBudget(inferOutput, ""))
			}
		}
	}

	return checks
}

// checkContinent verifies the .expedition/ directory structure exists.
// Returns a Warning-level check.
func checkContinent(continent string) domain.DoctorCheck {
	expeditionDir := filepath.Join(continent, domain.StateDir)
	info, err := os.Stat(expeditionDir)
	if err != nil || !info.IsDir() {
		return domain.DoctorCheck{
			Name:    "continent",
			Status:  domain.CheckWarn,
			Message: domain.StateDir + "/ not found",
			Hint:    `run "paintress init <repo-path>" to set up expedition`,
		}
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
		return domain.DoctorCheck{
			Name:    "continent",
			Status:  domain.CheckWarn,
			Message: "missing: " + strings.Join(missing, ", "),
			Hint:    `run "paintress init <repo-path>" to recreate the expedition structure`,
		}
	}

	return domain.DoctorCheck{
		Name:    "continent",
		Status:  domain.CheckOK,
		Message: fmt.Sprintf("%s (structure OK)", expeditionDir),
	}
}

// checkGitRepo verifies that the continent directory is inside a git repository.
// Returns a Warning-level check.
func checkGitRepo(continent string) domain.DoctorCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	cmd := exec.CommandContext(ctx, "git", "-C", continent, "rev-parse", "--git-dir")
	out, err := cmd.Output()
	cancel()
	if err != nil {
		return domain.DoctorCheck{
			Name:    "git-repo",
			Status:  domain.CheckWarn,
			Message: "not a git repository",
			Hint:    `run "git init" or navigate to a git repository`,
		}
	}

	return domain.DoctorCheck{
		Name:    "git-repo",
		Status:  domain.CheckOK,
		Message: fmt.Sprintf("%s (git repo OK)", strings.TrimSpace(string(out))),
	}
}

// checkGitRemote verifies that the git repository has at least one remote configured.
// Paintress creates Pull Requests for Linear issues, so a remote is required.
// Returns a Warning-level check.
func checkGitRemote(continent string) domain.DoctorCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	cmd := exec.CommandContext(ctx, "git", "-C", continent, "remote")
	out, err := cmd.Output()
	cancel()
	if err != nil {
		return domain.DoctorCheck{
			Name:    "git-remote",
			Status:  domain.CheckWarn,
			Message: "failed to check git remote",
			Hint:    `ensure the directory is a git repository`,
		}
	}

	if strings.TrimSpace(string(out)) == "" {
		return domain.DoctorCheck{
			Name:    "git-remote",
			Status:  domain.CheckWarn,
			Message: "no remote configured",
			Hint:    `paintress creates Pull Requests for Linear issues — run "git remote add origin <url>" to connect to GitHub`,
		}
	}

	remotes := strings.Fields(strings.TrimSpace(string(out)))
	return domain.DoctorCheck{
		Name:    "git-remote",
		Status:  domain.CheckOK,
		Message: fmt.Sprintf("%d remote(s): %s", len(remotes), strings.Join(remotes, ", ")),
	}
}

// checkWritability verifies that the .expedition/ directory is writable.
// Creates and removes a probe file to test write access.
// Returns a Warning-level check.
func checkWritability(continent string) domain.DoctorCheck {
	expeditionDir := filepath.Join(continent, domain.StateDir)
	probe := filepath.Join(expeditionDir, ".doctor-probe")
	if err := os.WriteFile(probe, []byte("probe"), 0644); err != nil {
		return domain.DoctorCheck{
			Name:    "writable",
			Status:  domain.CheckWarn,
			Message: "not writable: " + err.Error(),
			Hint:    "check file permissions on the .expedition/ directory",
		}
	}
	os.Remove(probe)

	return domain.DoctorCheck{
		Name:    "writable",
		Status:  domain.CheckOK,
		Message: fmt.Sprintf("%s (writable OK)", expeditionDir),
	}
}

// checkConfig verifies that config.yaml exists and can be loaded.
// Returns a Warning-level check.
func checkConfig(continent string) domain.DoctorCheck {
	configPath := domain.ProjectConfigPath(continent)
	if _, err := os.Stat(configPath); err != nil {
		return domain.DoctorCheck{
			Name:    "config",
			Status:  domain.CheckWarn,
			Message: "config.yaml not found",
			Hint:    `run "paintress init <repo-path>" to create config`,
		}
	}

	cfg, err := LoadProjectConfig(continent)
	if err != nil {
		return domain.DoctorCheck{
			Name:    "config",
			Status:  domain.CheckWarn,
			Message: "load error: " + err.Error(),
			Hint:    "check YAML syntax in .expedition/config.yaml",
		}
	}

	var msg string
	if cfg.HasTrackerTeam() {
		msg = fmt.Sprintf("%s (team=%s)", configPath, cfg.TrackerTeam())
	} else {
		msg = fmt.Sprintf("%s (loaded OK)", configPath)
	}
	return domain.DoctorCheck{
		Name:    "config",
		Status:  domain.CheckOK,
		Message: msg,
	}
}

// checkSkills verifies that SKILL.md files exist and contain dmail-schema-version.
// Searches .expedition/skills/*/SKILL.md for valid skill definitions.
// Returns a Warning-level check.
func checkSkills(continent string) domain.DoctorCheck {
	skillsDir := filepath.Join(continent, domain.StateDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return domain.DoctorCheck{
			Name:    "skills",
			Status:  domain.CheckWarn,
			Message: "skills/ not found",
			Hint:    `run "paintress init <repo-path>" to set up skills`,
		}
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
		return domain.DoctorCheck{
			Name:    "skills",
			Status:  domain.CheckWarn,
			Message: "no SKILL.md files found",
			Hint:    `run "paintress init <repo-path>" to create skill files`,
		}
	}

	if valid < found {
		return domain.DoctorCheck{
			Name:    "skills",
			Status:  domain.CheckWarn,
			Message: fmt.Sprintf("%d/%d skills missing dmail-schema-version", found-valid, found),
			Hint:    `add "dmail-schema-version" to SKILL.md metadata`,
		}
	}

	// Check for deprecated "kind: feedback" (split into design-feedback / implementation-feedback)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}
		content := string(data)
		if strings.Contains(content, "kind: feedback") &&
			!strings.Contains(content, "kind: design-feedback") &&
			!strings.Contains(content, "kind: implementation-feedback") {
			return domain.DoctorCheck{
				Name:    "skills",
				Status:  domain.CheckFail,
				Message: fmt.Sprintf("%s/SKILL.md uses deprecated kind 'feedback'", entry.Name()),
				Hint:    "deprecated kind 'feedback'; migrate to 'implementation-feedback' (run 'paintress init --force' to regenerate SKILL.md)",
			}
		}
	}

	return domain.DoctorCheck{
		Name:    "skills",
		Status:  domain.CheckOK,
		Message: fmt.Sprintf("%s (%d skills OK)", skillsDir, found),
	}
}

// checkEventStore verifies that event JSONL files are parseable.
// Scans .expedition/events/*.jsonl and validates each line is valid JSON. // nosemgrep: layer-session-no-event-persistence [permanent]
// Returns a Warning-level check.
func checkEventStore(continent string) domain.DoctorCheck {
	eventsDir := filepath.Join(continent, domain.StateDir, "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		return domain.DoctorCheck{
			Name:    "events",
			Status:  domain.CheckWarn,
			Message: "events/ not found",
			Hint:    `run "paintress init <repo-path>" to create events directory`,
		}
	}

	var files, lines int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") { // nosemgrep: layer-session-no-event-persistence [permanent]
			continue
		}
		f, err := os.Open(filepath.Join(eventsDir, entry.Name()))
		if err != nil {
			return domain.DoctorCheck{
				Name:    "events",
				Status:  domain.CheckWarn,
				Message: "read error: " + err.Error(),
				Hint:    "check file permissions on .expedition/events/",
			}
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if !json.Valid([]byte(line)) {
				f.Close()
				return domain.DoctorCheck{
					Name:    "events",
					Status:  domain.CheckWarn,
					Message: fmt.Sprintf("corrupt JSON in %s", entry.Name()),
					Hint:    "check event files for corruption in .expedition/events/",
				}
			}
			lines++
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			return domain.DoctorCheck{
				Name:    "events",
				Status:  domain.CheckWarn,
				Message: "scan error: " + err.Error(),
				Hint:    "check file permissions on .expedition/events/",
			}
		}
		files++
	}

	if files == 0 {
		return domain.DoctorCheck{
			Name:    "events",
			Status:  domain.CheckWarn,
			Message: "no .jsonl files found", // nosemgrep: layer-session-no-event-persistence [permanent]
		}
	}

	return domain.DoctorCheck{
		Name:    "events",
		Status:  domain.CheckOK,
		Message: fmt.Sprintf("%s (%d files, %d events OK)", eventsDir, files, lines),
	}
}

// checkClaudeAuth determines if the Claude CLI is authenticated by
// interpreting the result of running `claude mcp list`. A successful
// command execution (no error) indicates the CLI is authenticated.
func checkClaudeAuth(mcpOutput string, mcpErr error) domain.DoctorCheck {
	if mcpErr != nil {
		return domain.DoctorCheck{
			Name:    "claude-auth",
			Status:  domain.CheckWarn,
			Message: "not authenticated: " + mcpErr.Error(),
			Hint:    `run "claude login" to authenticate (in Docker: set CLAUDE_CONFIG_DIR=~/.claude to use host credentials)`,
		}
	}
	return domain.DoctorCheck{
		Name:    "claude-auth",
		Status:  domain.CheckOK,
		Message: "authenticated",
	}
}

// checkLinearMCP parses `claude mcp list` output for Linear MCP connection.
// Looks for a line containing "linear", "✓", and "connected" (case-insensitive).
// Requires "✓" to avoid false positives from "disconnected" or "not connected".
func checkLinearMCP(mcpOutput string, mcpErr error) domain.DoctorCheck {
	if mcpErr != nil {
		return domain.DoctorCheck{
			Name:    "linear-mcp",
			Status:  domain.CheckSkip,
			Message: "skipped (claude not available)",
		}
	}
	output := strings.ToLower(mcpOutput)
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "linear") &&
			strings.Contains(line, "✓") &&
			strings.Contains(line, "connected") {
			return domain.DoctorCheck{
				Name:    "linear-mcp",
				Status:  domain.CheckOK,
				Message: "Linear MCP connected",
			}
		}
	}
	return domain.DoctorCheck{
		Name:    "linear-mcp",
		Status:  domain.CheckWarn,
		Message: "Linear MCP not found or not connected",
		Hint: "run \"claude mcp add --transport http --scope project linear https://mcp.linear.app/mcp\" in your project root\n" +
			"  (a fully compatible local-only Linear MCP alternative is planned — check the project README for updates)",
	}
}

// checkClaudeInference determines if the Claude CLI can perform inference
// by interpreting the result of a minimal "1+1=" prompt.
func checkClaudeInference(output string, err error) domain.DoctorCheck {
	if err != nil {
		return domain.DoctorCheck{
			Name:    "claude-inference",
			Status:  domain.CheckWarn,
			Message: "inference failed: " + err.Error(),
			Hint: `"signal: killed" = CLI startup too slow (timeout 60s); ` +
				`"nested session" = CLAUDECODE env var leaked (doctor should filter it); ` +
				`otherwise check API key, quota, and model access`,
		}
	}
	if strings.TrimSpace(output) != "2" {
		return domain.DoctorCheck{
			Name:    "claude-inference",
			Status:  domain.CheckWarn,
			Message: "unexpected response: " + strings.TrimSpace(output),
			Hint:    "model returned unexpected output; check model access and API quota",
		}
	}
	return domain.DoctorCheck{
		Name:    "claude-inference",
		Status:  domain.CheckOK,
		Message: "inference OK",
	}
}

// requiredGHScopes lists OAuth scopes that paintress needs for full
// functionality (e.g. gh pr edit requires read:project when PRs are linked
// to GitHub Projects).
var requiredGHScopes = []string{"repo", "read:project"}

// checkGHScopes verifies that the gh CLI token has the required OAuth scopes.
// Parses the output of `gh auth status` for the "Token scopes:" line.
func checkGHScopes(output string, err error) domain.DoctorCheck {
	if err != nil {
		return domain.DoctorCheck{
			Name:    "gh-scopes",
			Status:  domain.CheckWarn,
			Message: "not authenticated: " + err.Error(),
			Hint:    `run "gh auth login" to authenticate`,
		}
	}

	// Find "Token scopes:" line
	var scopesLine string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Token scopes:") {
			scopesLine = line
			break
		}
	}
	if scopesLine == "" {
		return domain.DoctorCheck{
			Name:    "gh-scopes",
			Status:  domain.CheckWarn,
			Message: "could not determine token scopes",
			Hint:    `run "gh auth status" to check your token`,
		}
	}

	var missing []string
	for _, scope := range requiredGHScopes {
		if !strings.Contains(scopesLine, scope) {
			missing = append(missing, scope)
		}
	}

	if len(missing) > 0 {
		return domain.DoctorCheck{
			Name:    "gh-scopes",
			Status:  domain.CheckWarn,
			Message: "missing scopes: " + strings.Join(missing, ", "),
			Hint:    fmt.Sprintf(`run "gh auth refresh -s %s" to add missing scopes`, strings.Join(missing, " -s ")),
		}
	}

	return domain.DoctorCheck{
		Name:    "gh-scopes",
		Status:  domain.CheckOK,
		Message: "scopes OK",
	}
}

