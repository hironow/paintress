package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// installSkillsRefFn runs "uv tool install skills-ref". Injectable for testing.
var installSkillsRefFn = func() error {
	cmd := exec.Command("uv", "tool", "install", "skills-ref")
	return cmd.Run()
}

// findSkillsRefDirFn searches for skills-ref submodule directory relative to baseDir.
var findSkillsRefDirFn = findSkillsRefDir

// generateSkillsFn regenerates SKILL.md files via ValidateContinent.
var generateSkillsFn = func(continent string) error {
	_, err := ValidateContinent(continent, nil)
	return err
}

func findSkillsRefDir(baseDir string) string {
	candidates := []string{
		filepath.Join(baseDir, "..", "skills-ref"),
		filepath.Join(baseDir, "..", "..", "skills-ref"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c
		}
	}
	return ""
}

// OverrideInstallSkillsRef replaces the skills-ref installer for testing.
func OverrideInstallSkillsRef(fn func() error) func() {
	old := installSkillsRefFn
	installSkillsRefFn = fn
	return func() { installSkillsRefFn = old }
}

// OverrideFindSkillsRefDir replaces the skills-ref directory finder for testing.
func OverrideFindSkillsRefDir(fn func(string) string) func() {
	old := findSkillsRefDirFn
	findSkillsRefDirFn = fn
	return func() { findSkillsRefDirFn = old }
}

// OverrideGenerateSkills replaces the skills generator for testing.
func OverrideGenerateSkills(fn func(string) error) func() {
	old := generateSkillsFn
	generateSkillsFn = fn
	return func() { generateSkillsFn = old }
}

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
// When repair is true, auto-fixable issues are repaired in-place.
func RunDoctor(ctx context.Context, claudeCmd string, continent string, repair bool, mode domain.TrackingMode) []domain.DoctorCheck {
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
		vCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		out, err := newShellCmd(vCtx, cmd.name, "--version").Output()
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
		ghCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		cmd := exec.CommandContext(ghCtx, "gh", "auth", "status")
		out, err := cmd.CombinedOutput()
		cancel()
		checks = append(checks, checkGHScopes(string(out), err))
	}

	if strings.TrimSpace(continent) != "" {
		checks = append(checks, checkContinent(continent, repair))
		checks = append(checks, checkConfig(continent))
		checks = append(checks, checkGitRepo(ctx, continent))
		checks = append(checks, checkGitRemote(ctx, continent))
		checks = append(checks, checkWritability(continent))
		skillResult := checkSkills(continent)
		if repair && (skillResult.Status == domain.CheckFail || skillResult.Status == domain.CheckWarn) {
			if err := generateSkillsFn(continent); err == nil {
				recheck := checkSkills(continent)
				if recheck.Status == domain.CheckOK {
					checks = append(checks, domain.DoctorCheck{
						Name: "skills", Status: domain.CheckFixed,
						Message: "regenerated SKILL.md files",
					})
				} else {
					checks = append(checks, skillResult)
				}
			} else {
				checks = append(checks, skillResult)
			}
		} else {
			checks = append(checks, skillResult)
		}
		checks = append(checks, checkEventStore(continent))
		checks = append(checks, checkDeadLetters(ctx, continent))

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
			mcpCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			cmd := newShellCmd(mcpCtx, claudeCmd, "mcp", "list")
			out, err := cmd.Output()
			cancel()
			mcpOutput := string(out)
			authCheck := checkClaudeAuth(mcpOutput, err, claudeCmd)
			checks = append(checks, authCheck)

			// Linear MCP: skip in wave mode (no Linear dependency)
			if mode.IsWave() {
				checks = append(checks, domain.DoctorCheck{
					Name:    "linear-mcp",
					Status:  domain.CheckSkip,
					Message: "skipped (wave mode)",
				})
			} else if authCheck.Status != domain.CheckOK {
				checks = append(checks, domain.DoctorCheck{
					Name:    "linear-mcp",
					Status:  domain.CheckSkip,
					Message: "skipped (auth failed)",
				})
			} else {
				checks = append(checks, checkLinearMCP(mcpOutput, err))
			}

			// Inference: runs independently of mcp list result.
			// MCP config issues don't affect core inference capability.
			inferCtx, inferCancel := context.WithTimeout(ctx, 3*time.Minute)
			inferCmd := newShellCmd(inferCtx, claudeCmd, "--print", "--verbose", "--output-format", "stream-json", "--max-turns", "1", "1+1=")
			// Filter CLAUDECODE only for the doctor inference probe to prevent
			// nested-session errors. Other subprocesses must preserve CLAUDECODE.
			if inferCmd.Env != nil {
				inferCmd.Env = platform.FilterEnv(inferCmd.Env, "CLAUDECODE")
			} else {
				inferCmd.Env = platform.FilterEnv(os.Environ(), "CLAUDECODE")
			}
			inferOut, inferErr := inferCmd.Output()
			inferCancel()
			inferOutput := string(inferOut)
			inferResult := checkClaudeInference(strings.TrimSpace(ExtractStreamResult(inferOutput)), inferErr)
			checks = append(checks, inferResult)

			// Context budget check: skip if inference failed
			if inferResult.Status != domain.CheckOK {
				checks = append(checks, domain.DoctorCheck{
					Name:    "context-budget",
					Status:  domain.CheckSkip,
					Message: "skipped (inference failed)",
				})
			} else {
				checks = append(checks, CheckContextBudget(inferOutput, continent))
			}
		}
	}

	// --- skills-ref toolchain ---
	checks = append(checks, checkSkillsRefToolchain(continent, repair)...)

	// --- Repair: stale PID cleanup ---
	if repair && continent != "" {
		pidPath := filepath.Join(continent, domain.StateDir, "watch.pid")
		if data, err := os.ReadFile(pidPath); err == nil {
			pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			if pid > 0 && !platform.IsProcessAlive(pid) {
				_ = os.Remove(pidPath)
				checks = append(checks, domain.DoctorCheck{
					Name: "stale-pid", Status: domain.CheckFixed,
					Message: "removed stale PID file",
				})
			}
		}
	}

	return checks
}

// skillsRefBinNames lists possible binary names for the skills-ref package.
// "uv tool install skills-ref" installs as "agentskills", not "skills-ref".
var skillsRefBinNames = []string{"skills-ref", "agentskills"}

func findSkillsRefBin() (string, error) {
	for _, name := range skillsRefBinNames {
		if path, err := lookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("none of %v found on PATH", skillsRefBinNames)
}

// checkSkillsRefToolchain verifies that skills-ref tooling is available.
func checkSkillsRefToolchain(baseDir string, repair bool) []domain.DoctorCheck {
	if path, err := findSkillsRefBin(); err == nil {
		return []domain.DoctorCheck{{
			Name: "skills-ref", Status: domain.CheckOK,
			Message: fmt.Sprintf("skills-ref found on PATH (%s)", filepath.Base(path)),
		}}
	}
	_, uvErr := lookPath("uv")
	if uvErr != nil {
		return []domain.DoctorCheck{{
			Name: "skills-ref", Status: domain.CheckWarn,
			Message: "uv not found on PATH: SKILL.md spec validation is unavailable",
			Hint:    `install uv (https://docs.astral.sh/uv/) or "uv tool install skills-ref"`,
		}}
	}
	subDir := findSkillsRefDirFn(baseDir)
	if subDir != "" {
		return []domain.DoctorCheck{{
			Name: "skills-ref", Status: domain.CheckWarn,
			Message: fmt.Sprintf("skills-ref checkout found at %s but not on PATH", subDir),
			Hint:    `run "uv tool install skills-ref" or add skills-ref to PATH`,
		}}
	}
	if repair {
		if err := installSkillsRefFn(); err != nil {
			return []domain.DoctorCheck{{
				Name: "skills-ref", Status: domain.CheckWarn,
				Message: fmt.Sprintf("uv tool install skills-ref failed: %v", err),
				Hint:    `try manually: "uv tool install skills-ref"`,
			}}
		}
		if _, err := findSkillsRefBin(); err != nil {
			return []domain.DoctorCheck{{
				Name: "skills-ref", Status: domain.CheckWarn,
				Message: "installed skills-ref but executable not found on PATH",
				Hint:    `ensure uv tool bin directory is in PATH (e.g. ~/.local/bin)`,
			}}
		}
		return []domain.DoctorCheck{{
			Name: "skills-ref", Status: domain.CheckFixed,
			Message: "installed skills-ref via uv tool install",
		}}
	}
	return []domain.DoctorCheck{{
		Name: "skills-ref", Status: domain.CheckWarn,
		Message: "uv found but skills-ref not installed",
		Hint:    `run "paintress doctor --repair" or "uv tool install skills-ref"`,
	}}
}

// checkContinent verifies the .expedition/ directory structure exists.
// When repair is true, missing directories are created and CheckFixed is returned.
func checkContinent(continent string, repair bool) domain.DoctorCheck {
	expeditionDir := filepath.Join(continent, domain.StateDir)
	info, err := os.Stat(expeditionDir)
	if err != nil || !info.IsDir() {
		if !repair {
			return domain.DoctorCheck{
				Name:    "continent",
				Status:  domain.CheckWarn,
				Message: domain.StateDir + "/ not found",
				Hint:    `run "paintress init <repo-path>" or "paintress doctor --repair"`,
			}
		}
		if mkErr := os.MkdirAll(expeditionDir, 0755); mkErr != nil {
			return domain.DoctorCheck{
				Name:    "continent",
				Status:  domain.CheckWarn,
				Message: fmt.Sprintf("cannot create %s: %v", expeditionDir, mkErr),
				Hint:    `check directory permissions or run "paintress init <repo-path>"`,
			}
		}
	}

	requiredDirs := []string{"journal", ".run", "inbox", "outbox", "archive", "insights"}
	var missing []string
	for _, sub := range requiredDirs {
		d := filepath.Join(expeditionDir, sub)
		if fi, statErr := os.Stat(d); statErr != nil || !fi.IsDir() {
			missing = append(missing, sub)
		}
	}

	if len(missing) > 0 {
		if !repair {
			return domain.DoctorCheck{
				Name:    "continent",
				Status:  domain.CheckWarn,
				Message: "missing: " + strings.Join(missing, ", "),
				Hint:    `run "paintress init <repo-path>" or "paintress doctor --repair"`,
			}
		}
		for _, sub := range missing {
			if mkErr := os.MkdirAll(filepath.Join(expeditionDir, sub), 0755); mkErr != nil {
				return domain.DoctorCheck{
					Name:    "continent",
					Status:  domain.CheckWarn,
					Message: fmt.Sprintf("cannot create %s: %v", sub, mkErr),
					Hint:    `check directory permissions or run "paintress init <repo-path>"`,
				}
			}
		}
		return domain.DoctorCheck{
			Name:    "continent",
			Status:  domain.CheckFixed,
			Message: fmt.Sprintf("created %s and subdirectories", expeditionDir),
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
func checkGitRepo(ctx context.Context, continent string) domain.DoctorCheck {
	gitCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	cmd := exec.CommandContext(gitCtx, "git", "-C", continent, "rev-parse", "--git-dir")
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
func checkGitRemote(ctx context.Context, continent string) domain.DoctorCheck {
	remCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	cmd := exec.CommandContext(remCtx, "git", "-C", continent, "remote")
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

