package session_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// buildFakeClaude compiles the fake-claude binary and returns its absolute path.
func buildFakeClaude(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "fake-claude")

	// Locate fake-claude source relative to this test file.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// thisFile = internal/session/doctor_test.go → project root = ../../
	projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	fakeSrc := filepath.Join(projectRoot, "tests", "scenario", "testdata", "fake-claude")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = fakeSrc
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build fake-claude: %v\n%s", err, out)
	}
	return binPath
}

func TestRunDoctor_GitFound(t *testing.T) {
	// given/when
	checks := session.RunDoctor("claude", "")
	var gitCheck *struct {
		Name    string
		OK      bool
		Path    string
		Version string
	}
	for i := range checks {
		if checks[i].Name == "git" {
			gitCheck = &struct {
				Name    string
				OK      bool
				Path    string
				Version string
			}{checks[i].Name, checks[i].OK, checks[i].Path, checks[i].Version}
			break
		}
	}

	// then
	if gitCheck == nil {
		t.Fatal("expected git check in results")
	}
	if !gitCheck.OK {
		t.Error("git should be found in test environment")
	}
	if gitCheck.Path == "" {
		t.Error("git path should not be empty")
	}
	if gitCheck.Version == "" {
		t.Error("git version should not be empty")
	}
}

func TestRunDoctor_DockerIsOptional(t *testing.T) {
	// given/when
	checks := session.RunDoctor("claude", "")

	// then
	for i := range checks {
		if checks[i].Name == "docker" {
			if checks[i].Required {
				t.Error("docker should be optional (Required=false), used only for tracing and container tests")
			}
			return
		}
	}
	t.Fatal("expected docker check in results")
}

func TestRunDoctor_MissingCommand(t *testing.T) {
	// given/when
	checks := session.RunDoctor("nonexistent-paintress-cmd-12345", "")

	// then
	for i := range checks {
		if checks[i].Name == "nonexistent-paintress-cmd-12345" {
			if checks[i].OK {
				t.Error("nonexistent command should not be OK")
			}
			if checks[i].Path != "" {
				t.Errorf("path should be empty for missing command, got %q", checks[i].Path)
			}
			return
		}
	}
	t.Fatal("expected claude cmd check in results")
}

func TestRunDoctor_CheckContinent_ValidStructure(t *testing.T) {
	// given — valid .expedition/ structure
	dir := t.TempDir()
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when
	checks := session.RunDoctor("claude", dir)

	// then — continent check should be OK
	for _, c := range checks {
		if c.Name == "continent" {
			if !c.OK {
				t.Error("continent check should pass for valid structure")
			}
			if c.Required {
				t.Error("continent check should NOT be required (warning only)")
			}
			return
		}
	}
	t.Error("expected continent check in doctor output")
}

func TestRunDoctor_CheckContinent_MissingDir(t *testing.T) {
	// given — empty continent (no .expedition/)
	dir := t.TempDir()

	// when
	checks := session.RunDoctor("claude", dir)

	// then — continent check should be NOT OK but NOT required (warning)
	for _, c := range checks {
		if c.Name == "continent" {
			if c.OK {
				t.Error("continent check should fail when .expedition/ is missing")
			}
			if c.Required {
				t.Error("continent check should NOT be required")
			}
			return
		}
	}
	t.Error("expected continent check in doctor output")
}

func TestRunDoctor_CheckContinent_Empty_Skipped(t *testing.T) {
	// given — no continent path provided
	// when
	checks := session.RunDoctor("claude", "")

	// then — no continent check should appear
	for _, c := range checks {
		if c.Name == "continent" {
			t.Error("continent check should not appear when no continent provided")
		}
	}
}

func TestRunDoctor_CheckConfig_ValidConfig(t *testing.T) {
	// given — valid config.yaml
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)
	os.WriteFile(domain.ProjectConfigPath(dir), []byte("tracker:\n  team: TEST\n"), 0644)

	// when
	checks := session.RunDoctor("claude", dir)

	// then — config check should be OK
	for _, c := range checks {
		if c.Name == "config" {
			if !c.OK {
				t.Errorf("config check should pass for valid config, version: %s", c.Version)
			}
			if c.Required {
				t.Error("config check should NOT be required")
			}
			return
		}
	}
	t.Error("expected config check in doctor output")
}

func TestRunDoctor_CheckConfig_MissingConfig(t *testing.T) {
	// given — continent exists but no config.yaml
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	// when
	checks := session.RunDoctor("claude", dir)

	// then — config check should NOT be OK but NOT required (warning)
	for _, c := range checks {
		if c.Name == "config" {
			if c.OK {
				t.Error("config check should fail when config.yaml is missing")
			}
			if c.Required {
				t.Error("config check should NOT be required")
			}
			return
		}
	}
	t.Error("expected config check in doctor output")
}

func TestCheckGitRepo_InRepo(t *testing.T) {
	// given — a directory inside a git repo
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// when
	check := session.ExportCheckGitRepo(dir)

	// then
	if !check.OK {
		t.Error("git-repo check should pass inside a git repo")
	}
	if check.Required {
		t.Error("git-repo check should NOT be required (warning)")
	}
	if check.Name != "git-repo" {
		t.Errorf("expected name 'git-repo', got %q", check.Name)
	}
}

func TestCheckGitRepo_NotRepo(t *testing.T) {
	// given — a plain directory (not a git repo)
	dir := t.TempDir()

	// when
	check := session.ExportCheckGitRepo(dir)

	// then
	if check.OK {
		t.Error("git-repo check should fail outside a git repo")
	}
	if check.Required {
		t.Error("git-repo check should NOT be required")
	}
}

func TestCheckWritability_OK(t *testing.T) {
	// given — writable .expedition/
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	// when
	check := session.ExportCheckWritability(dir)

	// then
	if !check.OK {
		t.Errorf("writable check should pass, version: %s", check.Version)
	}
	if check.Required {
		t.Error("writable check should NOT be required")
	}
	if check.Name != "writable" {
		t.Errorf("expected name 'writable', got %q", check.Name)
	}
}

func TestCheckWritability_ReadOnly(t *testing.T) {
	// given — read-only .expedition/
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0555)
	t.Cleanup(func() { os.Chmod(expDir, 0755) })

	// when
	check := session.ExportCheckWritability(dir)

	// then
	if check.OK {
		t.Error("writable check should fail for read-only .expedition/")
	}
	if check.Required {
		t.Error("writable check should NOT be required")
	}
}

func TestCheckSkills_Valid(t *testing.T) {
	// given — SKILL.md with dmail-schema-version
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".expedition", "skills", "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\nmetadata:\n  dmail-schema-version: \"1\"\n---\n"), 0644)

	// when
	check := session.ExportCheckSkills(dir)

	// then
	if !check.OK {
		t.Errorf("skills check should pass, version: %s", check.Version)
	}
	if check.Required {
		t.Error("skills check should NOT be required")
	}
	if check.Name != "skills" {
		t.Errorf("expected name 'skills', got %q", check.Name)
	}
}

func TestCheckSkills_MissingFile(t *testing.T) {
	// given — no skills directory
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	// when
	check := session.ExportCheckSkills(dir)

	// then
	if check.OK {
		t.Error("skills check should fail when no SKILL.md files exist")
	}
}

func TestCheckSkills_MissingVersion(t *testing.T) {
	// given — SKILL.md without dmail-schema-version
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".expedition", "skills", "bad-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: bad\n---\n"), 0644)

	// when
	check := session.ExportCheckSkills(dir)

	// then
	if check.OK {
		t.Error("skills check should fail when dmail-schema-version is missing")
	}
}

func TestCheckSkills_DeprecatedFeedbackKind(t *testing.T) {
	// given — SKILL.md with deprecated "kind: feedback" (pre-split)
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".expedition", "skills", "dmail-readable")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: dmail-readable\nmetadata:\n  dmail-schema-version: \"1\"\nconsumes:\n    - kind: feedback\n---\n"), 0644)

	// when
	check := session.ExportCheckSkills(dir)

	// then
	if check.OK {
		t.Error("skills check should fail when deprecated 'kind: feedback' is found")
	}
	if !check.Required {
		t.Error("deprecated feedback kind should be a blocking failure (Required=true), aligned with amadeus/sightjack")
	}
	if !strings.Contains(check.Hint, "init --force") {
		t.Errorf("hint should suggest init --force, got %q", check.Hint)
	}
}

func TestCheckSkills_UpdatedFeedbackKind(t *testing.T) {
	// given — SKILL.md with updated "kind: implementation-feedback" (post-split)
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".expedition", "skills", "dmail-readable")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: dmail-readable\nmetadata:\n  dmail-schema-version: \"1\"\nconsumes:\n    - kind: implementation-feedback\n---\n"), 0644)

	// when
	check := session.ExportCheckSkills(dir)

	// then
	if !check.OK {
		t.Errorf("skills check should pass for updated kind, version: %s", check.Version)
	}
}

func TestCheckEventStore_Valid(t *testing.T) {
	// given — valid JSONL event file
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, ".expedition", "events")
	os.MkdirAll(eventsDir, 0755)
	os.WriteFile(filepath.Join(eventsDir, "2026-03-02.jsonl"),
		[]byte("{\"type\":\"expedition.completed\",\"timestamp\":\"2026-03-02T00:00:00Z\"}\n"), 0644)

	// when
	check := session.ExportCheckEventStore(dir)

	// then
	if !check.OK {
		t.Errorf("events check should pass, version: %s", check.Version)
	}
	if check.Required {
		t.Error("events check should NOT be required")
	}
	if check.Name != "events" {
		t.Errorf("expected name 'events', got %q", check.Name)
	}
}

func TestCheckEventStore_Corrupt(t *testing.T) {
	// given — corrupt JSONL (not valid JSON)
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, ".expedition", "events")
	os.MkdirAll(eventsDir, 0755)
	os.WriteFile(filepath.Join(eventsDir, "bad.jsonl"), []byte("not json\n"), 0644)

	// when
	check := session.ExportCheckEventStore(dir)

	// then
	if check.OK {
		t.Error("events check should fail for corrupt JSONL")
	}
}

func TestCheckEventStore_NoDir(t *testing.T) {
	// given — no events directory
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	// when
	check := session.ExportCheckEventStore(dir)

	// then
	if check.OK {
		t.Error("events check should fail when events directory missing")
	}
}

func TestCheckClaudeAuth_Authenticated(t *testing.T) {
	// given: successful mcp list output
	mcpOutput := "plugin:filesystem:filesystem: /path (stdio) - ✓ Connected\n"

	// when
	check := session.ExportCheckClaudeAuth(mcpOutput, nil)

	// then
	if !check.OK {
		t.Errorf("claude-auth should be OK when mcp list succeeds, version: %s", check.Version)
	}
	if check.Required {
		t.Error("claude-auth should NOT be required (warning)")
	}
	if check.Name != "claude-auth" {
		t.Errorf("expected name 'claude-auth', got %q", check.Name)
	}
}

func TestCheckClaudeAuth_Failed(t *testing.T) {
	// given: mcp list command failed
	mcpErr := fmt.Errorf("exit status 1")

	// when
	check := session.ExportCheckClaudeAuth("", mcpErr)

	// then
	if check.OK {
		t.Error("claude-auth should fail when mcp list errors")
	}
	if check.Version == "" {
		t.Error("version should contain diagnostic message")
	}
}

func TestCheckLinearMCP_Connected(t *testing.T) {
	// given: mcp list output showing linear connected
	mcpOutput := "plugin:linear:linear: https://mcp.linear.app/mcp (HTTP) - ✓ Connected\n"

	// when
	check := session.ExportCheckLinearMCP(mcpOutput, nil)

	// then
	if !check.OK {
		t.Errorf("linear-mcp should be OK when connected, version: %s", check.Version)
	}
	if check.Required {
		t.Error("linear-mcp should NOT be required")
	}
	if check.Name != "linear-mcp" {
		t.Errorf("expected name 'linear-mcp', got %q", check.Name)
	}
}

func TestCheckLinearMCP_NotFound(t *testing.T) {
	// given: mcp list output without linear
	mcpOutput := "plugin:filesystem:filesystem: /path (stdio) - ✓ Connected\n"

	// when
	check := session.ExportCheckLinearMCP(mcpOutput, nil)

	// then
	if check.OK {
		t.Error("linear-mcp should fail when linear not in output")
	}
}

func TestCheckLinearMCP_Disconnected(t *testing.T) {
	// given: mcp list output showing linear disconnected (no ✓)
	mcpOutput := "plugin:linear:linear: https://mcp.linear.app/mcp (HTTP) - ✗ Disconnected\n"

	// when
	check := session.ExportCheckLinearMCP(mcpOutput, nil)

	// then
	if check.OK {
		t.Error("linear-mcp should fail when linear is disconnected")
	}
}

func TestCheckLinearMCP_MCPListFailed(t *testing.T) {
	// given: mcp list command itself failed
	mcpErr := fmt.Errorf("exit status 1")

	// when
	check := session.ExportCheckLinearMCP("", mcpErr)

	// then
	if check.OK {
		t.Error("linear-mcp should fail when mcp list errors")
	}
	if !strings.Contains(check.Version, "skipped") {
		t.Errorf("version should indicate skipped, got %q", check.Version)
	}
}

func TestRunDoctor_MCPChecks_SkippedWhenClaudeUnavailable(t *testing.T) {
	// given — nonexistent claude command, valid continent
	dir := t.TempDir()
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when
	checks := session.RunDoctor("nonexistent-claude-xyz-12345", dir)

	// then — claude-auth, linear-mcp, and claude-inference should exist with skip message
	var authFound, mcpFound, inferFound bool
	for _, c := range checks {
		switch c.Name {
		case "claude-auth":
			authFound = true
			if c.OK {
				t.Error("claude-auth should not be OK when claude unavailable")
			}
			if !strings.Contains(c.Version, "skipped") {
				t.Errorf("expected 'skipped' in version, got %q", c.Version)
			}
		case "linear-mcp":
			mcpFound = true
			if c.OK {
				t.Error("linear-mcp should not be OK when claude unavailable")
			}
			if !strings.Contains(c.Version, "skipped") {
				t.Errorf("expected 'skipped' in version, got %q", c.Version)
			}
		case "claude-inference":
			inferFound = true
			if c.OK {
				t.Error("claude-inference should not be OK when claude unavailable")
			}
			if !strings.Contains(c.Version, "skipped") {
				t.Errorf("expected 'skipped' in version, got %q", c.Version)
			}
		}
	}
	if !authFound {
		t.Error("expected claude-auth check in doctor output")
	}
	if !mcpFound {
		t.Error("expected linear-mcp check in doctor output")
	}
	if !inferFound {
		t.Error("expected claude-inference check in doctor output")
	}
}

func TestCheckGitRemote_HasRemote(t *testing.T) {
	// given — git repo with a remote configured
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	addRemote := exec.Command("git", "-C", dir, "remote", "add", "origin", "https://github.com/example/repo.git")
	if err := addRemote.Run(); err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}

	// when
	check := session.ExportCheckGitRemote(dir)

	// then
	if !check.OK {
		t.Errorf("git-remote check should pass when remote exists, version: %s", check.Version)
	}
	if check.Required {
		t.Error("git-remote check should NOT be required (warning)")
	}
	if check.Name != "git-remote" {
		t.Errorf("expected name 'git-remote', got %q", check.Name)
	}
}

func TestCheckGitRemote_NoRemote(t *testing.T) {
	// given — git repo without any remote
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// when
	check := session.ExportCheckGitRemote(dir)

	// then
	if check.OK {
		t.Error("git-remote check should fail when no remote configured")
	}
	if check.Hint == "" {
		t.Error("hint should not be empty for missing remote")
	}
	if !strings.Contains(check.Hint, "Pull Request") && !strings.Contains(check.Hint, "PR") {
		t.Errorf("hint should mention Pull Request, got %q", check.Hint)
	}
}

func TestCheckGitRemote_NotGitRepo(t *testing.T) {
	// given — a plain directory (not a git repo)
	dir := t.TempDir()

	// when
	check := session.ExportCheckGitRemote(dir)

	// then
	if check.OK {
		t.Error("git-remote check should fail for non-git directory")
	}
}

func TestCheckGitRemote_IncludedInDoctorWithContinent(t *testing.T) {
	// given — git repo with remote and valid .expedition/ structure
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	addRemote := exec.Command("git", "-C", dir, "remote", "add", "origin", "https://github.com/example/repo.git")
	if err := addRemote.Run(); err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when
	checks := session.RunDoctor("claude", dir)

	// then — git-remote check should be present
	for _, c := range checks {
		if c.Name == "git-remote" {
			if !c.OK {
				t.Errorf("git-remote should pass, version: %s", c.Version)
			}
			return
		}
	}
	t.Error("expected git-remote check in doctor output when continent is provided")
}

func TestRunDoctor_MCPChecks_AllPassWithFakeClaude(t *testing.T) {
	// given — fake-claude binary, valid continent with expedition structure
	fakeClaude := buildFakeClaude(t)
	dir := t.TempDir()
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when
	checks := session.RunDoctor(fakeClaude, dir)

	// then — claude binary check should pass (fake-claude supports --version)
	var claudeFound, authFound, mcpFound, inferFound bool
	for _, c := range checks {
		switch c.Name {
		case fakeClaude:
			claudeFound = true
			if !c.OK {
				t.Errorf("claude check should pass with fake-claude, version: %s", c.Version)
			}
		case "claude-auth":
			authFound = true
			if !c.OK {
				t.Errorf("claude-auth should be OK with fake-claude, version: %s", c.Version)
			}
		case "linear-mcp":
			mcpFound = true
			if !c.OK {
				t.Errorf("linear-mcp should be OK with fake-claude, version: %s", c.Version)
			}
		case "claude-inference":
			inferFound = true
			if !c.OK {
				t.Errorf("claude-inference should be OK with fake-claude, version: %s", c.Version)
			}
		}
	}
	if !claudeFound {
		t.Error("expected claude binary check in doctor output")
	}
	if !authFound {
		t.Error("expected claude-auth check in doctor output")
	}
	if !mcpFound {
		t.Error("expected linear-mcp check in doctor output")
	}
	if !inferFound {
		t.Error("expected claude-inference check in doctor output")
	}
}

func TestRunDoctor_MCPChecks_NotPresentWithoutContinent(t *testing.T) {
	// given — no continent path
	// when
	checks := session.RunDoctor("claude", "")

	// then — MCP/inference checks should not appear
	for _, c := range checks {
		if c.Name == "claude-auth" || c.Name == "linear-mcp" || c.Name == "claude-inference" {
			t.Errorf("check %q should not appear without continent", c.Name)
		}
	}
}

func TestCheckClaudeInference_OK(t *testing.T) {
	// given: output contains "2"
	// when
	check := session.ExportCheckClaudeInference("2", nil)

	// then
	if !check.OK {
		t.Errorf("inference check should pass, version: %s", check.Version)
	}
	if check.Required {
		t.Error("inference check should NOT be required")
	}
	if check.Name != "claude-inference" {
		t.Errorf("expected name 'claude-inference', got %q", check.Name)
	}
	if check.Version != "inference OK" {
		t.Errorf("expected version 'inference OK', got %q", check.Version)
	}
}

func TestCheckClaudeInference_Error(t *testing.T) {
	// given: command failed
	// when
	check := session.ExportCheckClaudeInference("", fmt.Errorf("exit status 1"))

	// then
	if check.OK {
		t.Error("inference check should fail on error")
	}
	if !strings.Contains(check.Version, "inference failed") {
		t.Errorf("version should contain 'inference failed', got %q", check.Version)
	}
	if check.Hint == "" {
		t.Error("hint should not be empty on failure")
	}
}

func TestCheckClaudeInference_UnexpectedResponse(t *testing.T) {
	// given: output does not contain "2"
	// when
	check := session.ExportCheckClaudeInference("hello world", nil)

	// then
	if check.OK {
		t.Error("inference check should fail for unexpected response")
	}
	if check.Version != "unexpected response" {
		t.Errorf("expected version 'unexpected response', got %q", check.Version)
	}
}
