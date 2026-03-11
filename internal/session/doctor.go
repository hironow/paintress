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
		check := domain.DoctorCheck{
			Name:     cmd.name,
			Required: cmd.required,
		}

		path, err := lookPath(cmd.name)
		if err != nil {
			if cmd.required {
				check.Hint = fmt.Sprintf("install %s and ensure it is in PATH", cmd.name)
			}
			checks = append(checks, check)
			continue
		}

		check.Path = path
		check.OK = true

		// Try to get version (best-effort, 500ms timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		out, err := newShellCmd(ctx, cmd.name, "--version").Output()
		cancel()
		if err == nil {
			firstLine := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
			check.Version = firstLine
		}

		checks = append(checks, check)
	}

	// gh auth scope check (requires gh binary to be found)
	ghOK := false
	for _, c := range checks {
		if c.Name == "gh" && c.OK {
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
			if c.Name == claudeCmd && c.OK {
				claudeOK = true
				break
			}
		}
		if !claudeOK {
			checks = append(checks, domain.DoctorCheck{
				Name: "claude-auth", Required: false,
				Version: "skipped (claude not available)",
			})
			checks = append(checks, domain.DoctorCheck{
				Name: "linear-mcp", Required: false,
				Version: "skipped (claude not available)",
			})
			checks = append(checks, domain.DoctorCheck{
				Name: "claude-inference", Required: false,
				Version: "skipped (claude not available)",
			})
			checks = append(checks, domain.DoctorCheck{
				Name: "context-budget", Required: false,
				Version: "skipped (claude not available)",
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
			if !authCheck.OK {
				checks = append(checks, domain.DoctorCheck{
					Name: "claude-inference", Required: false,
					Version: "skipped (auth failed)",
				})
				checks = append(checks, domain.DoctorCheck{
					Name: "context-budget", Required: false,
					Version: "skipped (auth failed)",
				})
			} else {
				inferCtx, inferCancel := context.WithTimeout(context.Background(), 60*time.Second)
				inferCmd := newShellCmd(inferCtx, claudeCmd, "--print", "--output-format", "stream-json", "--max-turns", "1", "1+1=")
				inferCmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
				inferOut, inferErr := inferCmd.Output()
				inferCancel()
				inferOutput := string(inferOut)
				checks = append(checks, checkClaudeInference(strings.TrimSpace(ExtractStreamResult(inferOutput)), inferErr))
				checks = append(checks, CheckContextBudget(inferOutput))
			}
		}
	}

	return checks
}

// checkContinent verifies the .expedition/ directory structure exists.
// Returns a Warning-level check (Required=false).
func checkContinent(continent string) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "continent",
		Required: false,
	}

	expeditionDir := filepath.Join(continent, domain.StateDir)
	info, err := os.Stat(expeditionDir)
	if err != nil || !info.IsDir() {
		check.Version = domain.StateDir + "/ not found"
		check.Hint = `run "paintress init <repo-path>" to set up expedition`
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
		check.Hint = `run "paintress init <repo-path>" to recreate the expedition structure`
		return check
	}

	check.OK = true
	check.Path = expeditionDir
	check.Version = "structure OK"
	return check
}

// checkGitRepo verifies that the continent directory is inside a git repository.
// Returns a Warning-level check (Required=false).
func checkGitRepo(continent string) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "git-repo",
		Required: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	cmd := exec.CommandContext(ctx, "git", "-C", continent, "rev-parse", "--git-dir")
	out, err := cmd.Output()
	cancel()
	if err != nil {
		check.Version = "not a git repository"
		check.Hint = `run "git init" or navigate to a git repository`
		return check
	}

	check.OK = true
	check.Path = strings.TrimSpace(string(out))
	check.Version = "git repo OK"
	return check
}

// checkGitRemote verifies that the git repository has at least one remote configured.
// Paintress creates Pull Requests for Linear issues, so a remote is required.
// Returns a Warning-level check (Required=false).
func checkGitRemote(continent string) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "git-remote",
		Required: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	cmd := exec.CommandContext(ctx, "git", "-C", continent, "remote")
	out, err := cmd.Output()
	cancel()
	if err != nil {
		check.Version = "failed to check git remote"
		check.Hint = `ensure the directory is a git repository`
		return check
	}

	if strings.TrimSpace(string(out)) == "" {
		check.Version = "no remote configured"
		check.Hint = `paintress creates Pull Requests for Linear issues — run "git remote add origin <url>" to connect to GitHub`
		return check
	}

	remotes := strings.Fields(strings.TrimSpace(string(out)))
	check.OK = true
	check.Version = fmt.Sprintf("%d remote(s): %s", len(remotes), strings.Join(remotes, ", "))
	return check
}

// checkWritability verifies that the .expedition/ directory is writable.
// Creates and removes a probe file to test write access.
// Returns a Warning-level check (Required=false).
func checkWritability(continent string) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "writable",
		Required: false,
	}

	expeditionDir := filepath.Join(continent, domain.StateDir)
	probe := filepath.Join(expeditionDir, ".doctor-probe")
	if err := os.WriteFile(probe, []byte("probe"), 0644); err != nil {
		check.Version = "not writable: " + err.Error()
		check.Hint = "check file permissions on the .expedition/ directory"
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
func checkConfig(continent string) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "config",
		Required: false,
	}

	configPath := domain.ProjectConfigPath(continent)
	if _, err := os.Stat(configPath); err != nil {
		check.Version = "config.yaml not found"
		check.Hint = `run "paintress init <repo-path>" to create config`
		return check
	}

	cfg, err := LoadProjectConfig(continent)
	if err != nil {
		check.Version = "load error: " + err.Error()
		check.Hint = "check YAML syntax in .expedition/config.yaml"
		return check
	}

	check.OK = true
	check.Path = configPath
	if cfg.HasTrackerTeam() {
		check.Version = "team=" + cfg.TrackerTeam()
	} else {
		check.Version = "loaded OK"
	}
	return check
}

// checkSkills verifies that SKILL.md files exist and contain dmail-schema-version.
// Searches .expedition/skills/*/SKILL.md for valid skill definitions.
// Returns a Warning-level check (Required=false).
func checkSkills(continent string) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "skills",
		Required: false,
	}

	skillsDir := filepath.Join(continent, domain.StateDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		check.Version = "skills/ not found"
		check.Hint = `run "paintress init <repo-path>" to set up skills`
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
		check.Hint = `run "paintress init <repo-path>" to create skill files`
		return check
	}

	if valid < found {
		check.Version = fmt.Sprintf("%d/%d skills missing dmail-schema-version", found-valid, found)
		check.Hint = `add "dmail-schema-version" to SKILL.md metadata`
		return check
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
			check.Required = true // deprecated kind is a blocking failure (aligned with amadeus/sightjack)
			check.Version = fmt.Sprintf("%s/SKILL.md uses deprecated kind 'feedback'", entry.Name())
			check.Hint = "deprecated kind 'feedback'; migrate to 'implementation-feedback' (run 'paintress init --force' to regenerate SKILL.md)"
			return check
		}
	}

	check.OK = true
	check.Path = skillsDir
	check.Version = fmt.Sprintf("%d skills OK", found)
	return check
}

// checkEventStore verifies that event JSONL files are parseable.
// Scans .expedition/events/*.jsonl and validates each line is valid JSON. // nosemgrep: layer-session-no-event-persistence [permanent]
// Returns a Warning-level check (Required=false).
func checkEventStore(continent string) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "events",
		Required: false,
	}

	eventsDir := filepath.Join(continent, domain.StateDir, "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		check.Version = "events/ not found"
		check.Hint = `run "paintress init <repo-path>" to create events directory`
		return check
	}

	var files, lines int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") { // nosemgrep: layer-session-no-event-persistence [permanent]
			continue
		}
		f, err := os.Open(filepath.Join(eventsDir, entry.Name()))
		if err != nil {
			check.Version = "read error: " + err.Error()
			check.Hint = "check file permissions on .expedition/events/"
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
				check.Hint = "check event files for corruption in .expedition/events/"
				return check
			}
			lines++
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			check.Version = "scan error: " + err.Error()
			check.Hint = "check file permissions on .expedition/events/"
			return check
		}
		files++
	}

	if files == 0 {
		check.Version = "no .jsonl files found" // nosemgrep: layer-session-no-event-persistence [permanent]
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
func checkClaudeAuth(mcpOutput string, mcpErr error) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "claude-auth",
		Required: false,
	}
	if mcpErr != nil {
		check.Version = "not authenticated: " + mcpErr.Error()
		check.Hint = `run "claude login" to authenticate (in Docker: set CLAUDE_CONFIG_DIR=~/.claude to use host credentials)`
		return check
	}
	check.OK = true
	check.Version = "authenticated"
	return check
}

// checkLinearMCP parses `claude mcp list` output for Linear MCP connection.
// Looks for a line containing "linear", "✓", and "connected" (case-insensitive).
// Requires "✓" to avoid false positives from "disconnected" or "not connected".
func checkLinearMCP(mcpOutput string, mcpErr error) domain.DoctorCheck {
	check := domain.DoctorCheck{
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
	check.Hint = "run \"claude mcp add --transport http --scope project linear https://mcp.linear.app/mcp\" in your project root\n" +
		"  (a fully compatible local-only Linear MCP alternative is planned — check the project README for updates)"
	return check
}

// checkClaudeInference determines if the Claude CLI can perform inference
// by interpreting the result of a minimal "1+1=" prompt.
func checkClaudeInference(output string, err error) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "claude-inference",
		Required: false,
	}
	if err != nil {
		check.Version = "inference failed: " + err.Error()
		check.Hint = `"signal: killed" = CLI startup too slow (timeout 60s); ` +
			`"nested session" = CLAUDECODE env var leaked (doctor should filter it); ` +
			`otherwise check API key, quota, and model access`
		return check
	}
	if strings.TrimSpace(output) != "2" {
		check.Version = "unexpected response: " + strings.TrimSpace(output)
		check.Hint = "model returned unexpected output; check model access and API quota"
		return check
	}
	check.OK = true
	check.Version = "inference OK"
	return check
}

// requiredGHScopes lists OAuth scopes that paintress needs for full
// functionality (e.g. gh pr edit requires read:project when PRs are linked
// to GitHub Projects).
var requiredGHScopes = []string{"repo", "read:project"}

// checkGHScopes verifies that the gh CLI token has the required OAuth scopes.
// Parses the output of `gh auth status` for the "Token scopes:" line.
func checkGHScopes(output string, err error) domain.DoctorCheck {
	check := domain.DoctorCheck{
		Name:     "gh-scopes",
		Required: false,
	}
	if err != nil {
		check.Version = "not authenticated: " + err.Error()
		check.Hint = `run "gh auth login" to authenticate`
		return check
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
		check.Version = "could not determine token scopes"
		check.Hint = `run "gh auth status" to check your token`
		return check
	}

	var missing []string
	for _, scope := range requiredGHScopes {
		if !strings.Contains(scopesLine, scope) {
			missing = append(missing, scope)
		}
	}

	if len(missing) > 0 {
		check.Version = "missing scopes: " + strings.Join(missing, ", ")
		check.Hint = fmt.Sprintf(`run "gh auth refresh -s %s" to add missing scopes`, strings.Join(missing, " -s "))
		return check
	}

	check.OK = true
	check.Version = "scopes OK"
	return check
}

// ExtractStreamResult parses stream-json output and returns the "result" field
// from the result message. Used to extract inference output from stream-json format.
func ExtractStreamResult(streamJSON string) string {
	for _, line := range strings.Split(streamJSON, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg struct {
			Type   string `json:"type"`
			Result string `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err == nil && msg.Type == "result" {
			return msg.Result
		}
	}
	return ""
}

// CheckContextBudget parses stream-json output from a Claude CLI invocation
// and reports context budget health based on hooks, plugins, skills, and MCP servers.
func CheckContextBudget(streamJSON string) domain.DoctorCheck {
	var messages []*platform.StreamMessage
	for _, line := range strings.Split(streamJSON, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		msg, err := platform.ParseStreamMessage([]byte(line))
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	report := platform.CalculateContextBudget(messages)

	check := domain.DoctorCheck{
		Name:     "context-budget",
		Required: false,
		OK:       true,
		Version: fmt.Sprintf("estimated %d tokens (tools=%d, skills=%d, plugins=%d, mcp=%d, hook_bytes=%d)",
			report.EstimatedTokens, report.ToolCount, report.SkillCount,
			report.PluginCount, report.MCPServerCount, report.HookContextBytes),
	}
	if report.Exceeds(platform.DefaultContextBudgetThreshold) {
		check.Hint = "context consumption is high; consider reducing installed plugins/skills or using an allowlist"
	}
	return check
}

// filterEnv returns a copy of env with the named variable removed.
// Used to unset CLAUDECODE so that doctor's inference check does not
// trigger the nested-session guard in Claude Code.
func filterEnv(env []string, name string) []string {
	prefix := name + "="
	out := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return out
}
