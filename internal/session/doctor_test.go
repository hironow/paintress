package session_test

import (
	"context"
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
	checks := session.RunDoctor(context.Background(), "claude", "", false, domain.ModeWave)
	var gitCheck *domain.DoctorCheck
	for i := range checks {
		if checks[i].Name == "git" {
			gitCheck = &checks[i]
			break
		}
	}

	// then
	if gitCheck == nil {
		t.Fatal("expected git check in results")
	}
	if gitCheck.Status != domain.CheckOK {
		t.Error("git should be found in test environment")
	}
	if gitCheck.Message == "" {
		t.Error("git message should not be empty")
	}
}

func TestRunDoctor_DockerIsOptional(t *testing.T) {
	// given/when
	checks := session.RunDoctor(context.Background(), "claude", "", false, domain.ModeWave)

	// then
	for i := range checks {
		if checks[i].Name == "docker" {
			if checks[i].Status == domain.CheckFail {
				t.Error("docker should be optional (not CheckFail), used only for tracing and container tests")
			}
			return
		}
	}
	t.Fatal("expected docker check in results")
}

func TestRunDoctor_MissingCommand(t *testing.T) {
	// given/when
	checks := session.RunDoctor(context.Background(), "nonexistent-paintress-cmd-12345", "", false, domain.ModeWave)

	// then
	for i := range checks {
		if checks[i].Name == "nonexistent-paintress-cmd-12345" {
			if checks[i].Status == domain.CheckOK {
				t.Error("nonexistent command should not be OK")
			}
			return
		}
	}
	t.Fatal("expected claude cmd check in results")
}

func TestRunDoctor_CheckContinent_ValidStructure(t *testing.T) {
	// given — valid .expedition/ structure
	dir := t.TempDir()
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive", "insights"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when — use nonexistent cmd to skip slow external checks (continent is the SUT)
	checks := session.RunDoctor(context.Background(), "nonexistent-paintress-cmd-12345", dir, false, domain.ModeWave)

	// then — continent check should be OK
	for _, c := range checks {
		if c.Name == "continent" {
			if c.Status != domain.CheckOK {
				t.Error("continent check should pass for valid structure")
			}
			if c.Status == domain.CheckFail {
				t.Error("continent check should NOT be a failure (warning only)")
			}
			return
		}
	}
	t.Error("expected continent check in doctor output")
}

func TestRunDoctor_CheckContinent_MissingDir(t *testing.T) {
	// given — empty continent (no .expedition/)
	dir := t.TempDir()

	// when — use nonexistent cmd to skip slow external checks
	checks := session.RunDoctor(context.Background(), "nonexistent-paintress-cmd-12345", dir, false, domain.ModeWave)

	// then — continent check should be NOT OK but NOT required (warning)
	for _, c := range checks {
		if c.Name == "continent" {
			if c.Status == domain.CheckOK {
				t.Error("continent check should fail when .expedition/ is missing")
			}
			if c.Status == domain.CheckFail {
				t.Error("continent check should NOT be a failure")
			}
			return
		}
	}
	t.Error("expected continent check in doctor output")
}

func TestRunDoctor_CheckContinent_Empty_Skipped(t *testing.T) {
	// given — no continent path provided
	// when
	checks := session.RunDoctor(context.Background(), "claude", "", false, domain.ModeWave)

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

	// when — use nonexistent cmd to skip slow external checks
	checks := session.RunDoctor(context.Background(), "nonexistent-paintress-cmd-12345", dir, false, domain.ModeWave)

	// then — config check should be OK
	for _, c := range checks {
		if c.Name == "config" {
			if c.Status != domain.CheckOK {
				t.Errorf("config check should pass for valid config, message: %s", c.Message)
			}
			if c.Status == domain.CheckFail {
				t.Error("config check should NOT be a failure")
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

	// when — use nonexistent cmd to skip slow external checks
	checks := session.RunDoctor(context.Background(), "nonexistent-paintress-cmd-12345", dir, false, domain.ModeWave)

	// then — config check should NOT be OK but NOT required (warning)
	for _, c := range checks {
		if c.Name == "config" {
			if c.Status == domain.CheckOK {
				t.Error("config check should fail when config.yaml is missing")
			}
			if c.Status == domain.CheckFail {
				t.Error("config check should NOT be a failure")
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
	if check.Status != domain.CheckOK {
		t.Error("git-repo check should pass inside a git repo")
	}
	if check.Status == domain.CheckFail {
		t.Error("git-repo check should NOT be a failure (warning)")
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
	if check.Status == domain.CheckOK {
		t.Error("git-repo check should fail outside a git repo")
	}
	if check.Status == domain.CheckFail {
		t.Error("git-repo check should NOT be a failure")
	}
}

func TestCheckWritability_OK(t *testing.T) {
	// given — writable .expedition/
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	// when
	check := session.ExportCheckWritability(dir)

	// then
	if check.Status != domain.CheckOK {
		t.Errorf("writable check should pass, message: %s", check.Message)
	}
	if check.Status == domain.CheckFail {
		t.Error("writable check should NOT be a failure")
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
	if check.Status == domain.CheckOK {
		t.Error("writable check should fail for read-only .expedition/")
	}
	if check.Status == domain.CheckFail {
		t.Error("writable check should NOT be a failure")
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
	if check.Status != domain.CheckOK {
		t.Errorf("skills check should pass, message: %s", check.Message)
	}
	if check.Status == domain.CheckFail {
		t.Error("skills check should NOT be a failure")
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
	if check.Status == domain.CheckOK {
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
	if check.Status == domain.CheckOK {
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
	if check.Status == domain.CheckOK {
		t.Error("skills check should fail when deprecated 'kind: feedback' is found")
	}
	if check.Status != domain.CheckFail {
		t.Error("deprecated feedback kind should be a blocking failure (CheckFail), aligned with amadeus/sightjack")
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
	if check.Status != domain.CheckOK {
		t.Errorf("skills check should pass for updated kind, message: %s", check.Message)
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
	if check.Status != domain.CheckOK {
		t.Errorf("events check should pass, message: %s", check.Message)
	}
	if check.Status == domain.CheckFail {
		t.Error("events check should NOT be a failure")
	}
	if check.Name != "events" {
		t.Errorf("expected name 'events', got %q", check.Name)
	}
}

func TestCheckEventStore_Corrupt(t *testing.T) {
	// given — corrupt JSONL mixed with valid events
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, ".expedition", "events")
	os.MkdirAll(eventsDir, 0755)
	valid := `{"type":"expedition.completed","data":{},"timestamp":"2026-04-08T00:00:00Z","schema_version":1}`
	os.WriteFile(filepath.Join(eventsDir, "bad.jsonl"), []byte(valid+"\nnot json\n"+valid+"\n"), 0644)

	// when
	check := session.ExportCheckEventStore(dir)

	// then
	if check.Status != domain.CheckWarn {
		t.Errorf("expected WARN, got %s: %s", check.Status.StatusLabel(), check.Message)
	}
	if !strings.Contains(check.Message, "1 corrupt line") {
		t.Errorf("expected '1 corrupt line' in message: %q", check.Message)
	}
}

func TestCheckEventStore_StructuralCorrupt(t *testing.T) {
	// given — valid JSON but invalid domain.Event (bad timestamp format)
	// json.Valid would accept this, but json.Unmarshal into domain.Event fails
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, ".expedition", "events")
	os.MkdirAll(eventsDir, 0755)
	structuralCorrupt := `{"type":"expedition.completed","data":{},"timestamp":"not-a-date"}`
	valid := `{"type":"expedition.completed","data":{},"timestamp":"2026-04-08T00:00:00Z","schema_version":1}`
	os.WriteFile(filepath.Join(eventsDir, "structural.jsonl"),
		[]byte(valid+"\n"+structuralCorrupt+"\n"), 0644)

	// when
	check := session.ExportCheckEventStore(dir)

	// then: structural corruption detected via json.Unmarshal (not json.Valid)
	if check.Status != domain.CheckWarn {
		t.Errorf("expected WARN for structural corruption, got %s: %s", check.Status.StatusLabel(), check.Message)
	}
	if !strings.Contains(check.Message, "1 corrupt line") {
		t.Errorf("expected '1 corrupt line' in message: %q", check.Message)
	}
}

func TestCheckEventStore_NoDir(t *testing.T) {
	// given — no events directory
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	// when
	check := session.ExportCheckEventStore(dir)

	// then
	if check.Status == domain.CheckOK {
		t.Error("events check should fail when events directory missing")
	}
}

func TestCheckClaudeAuth_Authenticated(t *testing.T) {
	// given: successful mcp list output
	mcpOutput := "plugin:filesystem:filesystem: /path (stdio) - ✓ Connected\n"

	// when
	check := session.ExportCheckClaudeAuth(mcpOutput, nil, "claude")

	// then
	if check.Status != domain.CheckOK {
		t.Errorf("claude-auth should be OK when mcp list succeeds, message: %s", check.Message)
	}
	if check.Status == domain.CheckFail {
		t.Error("claude-auth should NOT be a failure (warning)")
	}
	if check.Name != "claude-auth" {
		t.Errorf("expected name 'claude-auth', got %q", check.Name)
	}
}

func TestCheckClaudeAuth_Failed(t *testing.T) {
	// given: mcp list command failed
	mcpErr := fmt.Errorf("exit status 1")

	// when
	check := session.ExportCheckClaudeAuth("", mcpErr, "claude")

	// then
	if check.Status == domain.CheckOK {
		t.Error("claude-auth should fail when mcp list errors")
	}
	if check.Message == "" {
		t.Error("message should contain diagnostic message")
	}
}

func TestCheckLinearMCP_Connected(t *testing.T) {
	// given: mcp list output showing linear connected
	mcpOutput := "plugin:linear:linear: https://mcp.linear.app/mcp (HTTP) - ✓ Connected\n"

	// when
	check := session.ExportCheckLinearMCP(mcpOutput, nil)

	// then
	if check.Status != domain.CheckOK {
		t.Errorf("linear-mcp should be OK when connected, message: %s", check.Message)
	}
	if check.Status == domain.CheckFail {
		t.Error("linear-mcp should NOT be a failure")
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
	if check.Status == domain.CheckOK {
		t.Error("linear-mcp should fail when linear not in output")
	}
}

func TestCheckLinearMCP_Disconnected(t *testing.T) {
	// given: mcp list output showing linear disconnected (no ✓)
	mcpOutput := "plugin:linear:linear: https://mcp.linear.app/mcp (HTTP) - ✗ Disconnected\n"

	// when
	check := session.ExportCheckLinearMCP(mcpOutput, nil)

	// then
	if check.Status == domain.CheckOK {
		t.Error("linear-mcp should fail when linear is disconnected")
	}
}

func TestCheckLinearMCP_MCPListFailed(t *testing.T) {
	// given: mcp list command itself failed
	mcpErr := fmt.Errorf("exit status 1")

	// when
	check := session.ExportCheckLinearMCP("", mcpErr)

	// then
	if check.Status != domain.CheckWarn {
		t.Errorf("linear-mcp should be WARN when mcp list errors, got %v", check.Status)
	}
	if !strings.Contains(check.Message, "claude mcp list failed") {
		t.Errorf("message should indicate mcp list failure, got %q", check.Message)
	}
}

func TestRunDoctor_MCPChecks_SkippedWhenClaudeUnavailable(t *testing.T) {
	// given — nonexistent claude command, valid continent
	dir := t.TempDir()
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive", "insights"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when
	checks := session.RunDoctor(context.Background(), "nonexistent-claude-xyz-12345", dir, false, domain.ModeWave)

	// then — claude-auth, linear-mcp, claude-inference, and context-budget should exist with skip message
	var authFound, mcpFound, inferFound, budgetFound bool
	for _, c := range checks {
		switch c.Name {
		case "claude-auth":
			authFound = true
			if c.Status == domain.CheckOK {
				t.Error("claude-auth should not be OK when claude unavailable")
			}
			if !strings.Contains(c.Message, "skipped") {
				t.Errorf("expected 'skipped' in message, got %q", c.Message)
			}
		case "linear-mcp":
			mcpFound = true
			if c.Status == domain.CheckOK {
				t.Error("linear-mcp should not be OK when claude unavailable")
			}
			if !strings.Contains(c.Message, "skipped") {
				t.Errorf("expected 'skipped' in message, got %q", c.Message)
			}
		case "claude-inference":
			inferFound = true
			if c.Status == domain.CheckOK {
				t.Error("claude-inference should not be OK when claude unavailable")
			}
			if !strings.Contains(c.Message, "skipped") {
				t.Errorf("expected 'skipped' in message, got %q", c.Message)
			}
		case "context-budget":
			budgetFound = true
			if c.Status == domain.CheckOK {
				t.Error("context-budget should not be OK when claude unavailable")
			}
			if !strings.Contains(c.Message, "skipped") {
				t.Errorf("expected 'skipped' in message, got %q", c.Message)
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
	if !budgetFound {
		t.Error("expected context-budget check in doctor output")
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
	if check.Status != domain.CheckOK {
		t.Errorf("git-remote check should pass when remote exists, message: %s", check.Message)
	}
	if check.Status == domain.CheckFail {
		t.Error("git-remote check should NOT be a failure (warning)")
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
	if check.Status == domain.CheckOK {
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
	if check.Status == domain.CheckOK {
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
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive", "insights"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when — use nonexistent cmd to skip slow external checks (git-remote is the SUT)
	checks := session.RunDoctor(context.Background(), "nonexistent-paintress-cmd-12345", dir, false, domain.ModeWave)

	// then — git-remote check should be present
	for _, c := range checks {
		if c.Name == "git-remote" {
			if c.Status != domain.CheckOK {
				t.Errorf("git-remote should pass, message: %s", c.Message)
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
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive", "insights"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when
	checks := session.RunDoctor(context.Background(), fakeClaude, dir, false, domain.ModeLinear)

	// then — claude binary check should pass (fake-claude supports --version)
	var claudeFound, authFound, mcpFound, inferFound, budgetFound bool
	for _, c := range checks {
		switch c.Name {
		case fakeClaude:
			claudeFound = true
			if c.Status != domain.CheckOK {
				t.Errorf("claude check should pass with fake-claude, message: %s", c.Message)
			}
		case "claude-auth":
			authFound = true
			if c.Status != domain.CheckOK {
				t.Errorf("claude-auth should be OK with fake-claude, message: %s", c.Message)
			}
		case "linear-mcp":
			mcpFound = true
			if c.Status != domain.CheckOK {
				t.Errorf("linear-mcp should be OK with fake-claude, message: %s", c.Message)
			}
		case "claude-inference":
			inferFound = true
			if c.Status != domain.CheckOK {
				t.Errorf("claude-inference should be OK with fake-claude, message: %s", c.Message)
			}
		case "context-budget":
			budgetFound = true
			if c.Status != domain.CheckOK {
				t.Errorf("context-budget should be OK with fake-claude, message: %s", c.Message)
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
	if !budgetFound {
		t.Error("expected context-budget check in doctor output")
	}
}

func TestRunDoctor_MCPChecks_NotPresentWithoutContinent(t *testing.T) {
	// given — no continent path
	// when
	checks := session.RunDoctor(context.Background(), "claude", "", false, domain.ModeWave)

	// then — MCP/inference/budget checks should not appear
	for _, c := range checks {
		if c.Name == "claude-auth" || c.Name == "linear-mcp" || c.Name == "claude-inference" || c.Name == "context-budget" {
			t.Errorf("check %q should not appear without continent", c.Name)
		}
	}
}

func TestCheckClaudeInference_OK(t *testing.T) {
	// given: trimmed output is exactly "2"
	// when
	check := session.ExportCheckClaudeInference("2", nil)

	// then
	if check.Status != domain.CheckOK {
		t.Errorf("inference check should pass, message: %s", check.Message)
	}
	if check.Status == domain.CheckFail {
		t.Error("inference check should NOT be a failure")
	}
	if check.Name != "claude-inference" {
		t.Errorf("expected name 'claude-inference', got %q", check.Name)
	}
	if check.Message != "inference OK" {
		t.Errorf("expected message 'inference OK', got %q", check.Message)
	}
}

func TestCheckClaudeInference_Error(t *testing.T) {
	// given: command failed
	// when
	check := session.ExportCheckClaudeInference("", fmt.Errorf("exit status 1"))

	// then
	if check.Status == domain.CheckOK {
		t.Error("inference check should fail on error")
	}
	if !strings.Contains(check.Message, "inference failed") {
		t.Errorf("message should contain 'inference failed', got %q", check.Message)
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
	if check.Status == domain.CheckOK {
		t.Error("inference check should fail for unexpected response")
	}
	if !strings.HasPrefix(check.Message, "unexpected response: ") {
		t.Errorf("expected message starting with 'unexpected response: ', got %q", check.Message)
	}
}

func TestCheckClaudeInference_FalsePositiveContaining2(t *testing.T) {
	// given: output contains "2" but is not exactly "2" (e.g. "12")
	// when
	check := session.ExportCheckClaudeInference("12", nil)

	// then
	if check.Status == domain.CheckOK {
		t.Error("inference check should fail for '12' (false positive from Contains)")
	}
	if !strings.HasPrefix(check.Message, "unexpected response: ") {
		t.Errorf("expected message starting with 'unexpected response: ', got %q", check.Message)
	}
}

func TestCheckClaudeInference_OKWithWhitespace(t *testing.T) {
	// given: output is "2" with surrounding whitespace (trimmed to exact match)
	// when
	check := session.ExportCheckClaudeInference("  2\n", nil)

	// then
	if check.Status != domain.CheckOK {
		t.Errorf("inference check should pass for trimmed '2', message: %s", check.Message)
	}
}

func TestCheckGHScopes_AllScopesPresent(t *testing.T) {
	// given: gh auth status output with all required scopes
	output := "github.com\n  ✓ Logged in to github.com account user (keyring)\n  - Token scopes: 'admin:public_key', 'read:org', 'read:project', 'repo', 'workflow'\n"

	// when
	check := session.ExportCheckGHScopes(output, nil)

	// then
	if check.Status != domain.CheckOK {
		t.Errorf("gh-scopes should pass when all required scopes present, message: %s", check.Message)
	}
	if check.Name != "gh-scopes" {
		t.Errorf("expected name 'gh-scopes', got %q", check.Name)
	}
	if check.Status == domain.CheckFail {
		t.Error("gh-scopes should NOT be a failure (warning)")
	}
}

func TestCheckGHScopes_MissingReadProject(t *testing.T) {
	// given: gh auth status output missing read:project
	output := "github.com\n  ✓ Logged in to github.com account user (keyring)\n  - Token scopes: 'admin:public_key', 'repo', 'workflow'\n"

	// when
	check := session.ExportCheckGHScopes(output, nil)

	// then
	if check.Status == domain.CheckOK {
		t.Error("gh-scopes should fail when read:project is missing")
	}
	if !strings.Contains(check.Message, "read:project") {
		t.Errorf("message should mention missing scope, got %q", check.Message)
	}
	if !strings.Contains(check.Hint, "gh auth refresh") {
		t.Errorf("hint should suggest gh auth refresh, got %q", check.Hint)
	}
}

func TestCheckGHScopes_CommandFailed(t *testing.T) {
	// given: gh auth status failed
	// when
	check := session.ExportCheckGHScopes("", fmt.Errorf("exit status 1"))

	// then
	if check.Status == domain.CheckOK {
		t.Error("gh-scopes should fail when command errors")
	}
	if !strings.Contains(check.Message, "not authenticated") {
		t.Errorf("message should indicate not authenticated, got %q", check.Message)
	}
}

func TestCheckGHScopes_NoScopesLine(t *testing.T) {
	// given: gh auth status output without Token scopes line
	output := "github.com\n  ✓ Logged in to github.com account user (keyring)\n"

	// when
	check := session.ExportCheckGHScopes(output, nil)

	// then
	if check.Status == domain.CheckOK {
		t.Error("gh-scopes should fail when scopes line not found")
	}
}

func TestExtractStreamResult_WithResult(t *testing.T) {
	// given
	stream := `{"type":"system","subtype":"init","session_id":"s1"}
{"type":"assistant","session_id":"s1","message":{"content":[{"type":"text","text":"2"}]}}
{"type":"result","subtype":"success","session_id":"s1","result":"2","is_error":false}
`
	// when
	got := session.ExportExtractStreamResult(stream)

	// then
	if got != "2" {
		t.Errorf("expected '2', got %q", got)
	}
}

func TestExtractStreamResult_Empty(t *testing.T) {
	// given
	stream := ""

	// when
	got := session.ExportExtractStreamResult(stream)

	// then
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractStreamResult_NoResult(t *testing.T) {
	// given: stream without result line
	stream := `{"type":"system","subtype":"init","session_id":"s1"}
{"type":"assistant","session_id":"s1","message":{"content":[{"type":"text","text":"hello"}]}}
`
	// when
	got := session.ExportExtractStreamResult(stream)

	// then
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestCheckContextBudget_LowUsage(t *testing.T) {
	// given: stream-json with small init metadata
	stream := `{"type":"system","subtype":"init","session_id":"s1","tools":[{"name":"Read"},{"name":"Write"}],"mcp_servers":[{"name":"linear","status":"connected"}]}
{"type":"result","subtype":"success","session_id":"s1","result":"2","is_error":false}
`
	// when
	check := session.ExportCheckContextBudget(stream, "")

	// then
	if check.Status != domain.CheckOK {
		t.Errorf("context-budget should be OK for low usage, message: %s", check.Message)
	}
	if check.Name != "context-budget" {
		t.Errorf("expected name 'context-budget', got %q", check.Name)
	}
	if check.Hint != "" {
		t.Errorf("should not have hint for low usage, got %q", check.Hint)
	}
	if !strings.Contains(check.Message, "estimated") {
		t.Errorf("message should contain 'estimated', got %q", check.Message)
	}
}

func TestCheckContextBudget_HighUsage(t *testing.T) {
	// given: stream-json with hook output exceeding threshold
	// 100000 bytes / 4 = 25000 tokens > 20000 threshold
	hookOutput := strings.Repeat("x", 100000)
	stream := fmt.Sprintf(`{"type":"system","subtype":"init","session_id":"s1","tools":[],"mcp_servers":[{"name":"linear","status":"connected"}]}
{"type":"system","subtype":"hook_response","session_id":"s1","stdout":%q}
{"type":"result","subtype":"success","session_id":"s1","result":"2","is_error":false}
`, hookOutput)

	// when
	check := session.ExportCheckContextBudget(stream, "")

	// then
	if check.Status != domain.CheckWarn {
		t.Errorf("context-budget should be WARN for high usage, message: %s", check.Message)
	}
	if check.Hint == "" {
		t.Error("should have hint for high usage")
	}
}

func TestCheckContextBudget_EmptyStream(t *testing.T) {
	// given
	stream := ""

	// when
	check := session.ExportCheckContextBudget(stream, "")

	// then
	if check.Status != domain.CheckOK {
		t.Error("context-budget should be OK for empty stream (0 tokens)")
	}
	if !strings.Contains(check.Message, "estimated 0 tokens") {
		t.Errorf("expected '0 tokens' in message, got %q", check.Message)
	}
}

func TestCheckContextBudget_NoInitMessage(t *testing.T) {
	// given: stream without init message
	stream := `{"type":"result","subtype":"success","session_id":"s1","result":"2","is_error":false}
`
	// when
	check := session.ExportCheckContextBudget(stream, "")

	// then
	if check.Status != domain.CheckOK {
		t.Error("context-budget should be OK (0 tokens) without init")
	}
}

func TestCheckContextBudget_WarnWithBreakdown(t *testing.T) {
	// given: stream with many skills (exceeds threshold)
	initMsg := `{"type":"system","subtype":"init","tools":["Read","Write"],"skills":["a","b","c","d","e","f","g","h","i","j","k","l","m","n","o","p","q","r","s","t","u","v","w","x","y","z","aa","ab","ac","ad","ae","af","ag","ah","ai","aj","ak","al","am","an"],"plugins":[{"name":"p1"},{"name":"p2"},{"name":"p3"},{"name":"p4"},{"name":"p5"}],"mcp_servers":[{"name":"linear","status":"connected"}]}`
	stream := initMsg + "\n"

	// when
	check := session.ExportCheckContextBudget(stream, "")

	// then
	if check.Status != domain.CheckWarn {
		t.Errorf("expected WARN, got %v", check.Status.StatusLabel())
	}
	if !strings.Contains(check.Message, "skills") {
		t.Errorf("message should contain breakdown with 'skills', got: %s", check.Message)
	}
	if check.Hint == "" {
		t.Error("expected hint for threshold exceeded")
	}
}

func TestCheckContextBudget_WarnHintWithSettingsFile(t *testing.T) {
	// given: project with .claude/settings.json
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), []byte(`{}`), 0o644)

	initMsg := `{"type":"system","subtype":"init","skills":["a","b","c","d","e","f","g","h","i","j","k","l","m","n","o","p","q","r","s","t","u","v","w","x","y","z","aa","ab","ac","ad","ae","af","ag","ah","ai","aj","ak","al","am","an","ao","ap"]}`
	stream := initMsg + "\n"

	// when
	check := session.ExportCheckContextBudget(stream, dir)

	// then
	if check.Status != domain.CheckWarn {
		t.Errorf("expected WARN, got %v", check.Status.StatusLabel())
	}
	if !strings.Contains(check.Hint, "見直して") {
		t.Errorf("hint should say review settings, got: %s", check.Hint)
	}
}

// --- Fix 1: repair creates insights/ directory ---

func TestCheckContinent_RepairCreatesInsightsDir(t *testing.T) {
	// given — .expedition/ exists but insights/ is missing
	dir := t.TempDir()
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when — repair=true
	check := session.ExportCheckContinent(dir, true)

	// then — should be FIXED and insights/ should exist
	if check.Status != domain.CheckFixed {
		t.Errorf("expected CheckFixed, got %v, message: %s", check.Status.StatusLabel(), check.Message)
	}
	insightsPath := filepath.Join(dir, ".expedition", "insights")
	if fi, err := os.Stat(insightsPath); err != nil || !fi.IsDir() {
		t.Errorf("insights/ directory should exist after repair")
	}
}

func TestCheckContinent_InsightsMissingReportsWarn(t *testing.T) {
	// given — all dirs except insights/
	dir := t.TempDir()
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when — repair=false
	check := session.ExportCheckContinent(dir, false)

	// then — should be WARN mentioning insights
	if check.Status != domain.CheckWarn {
		t.Errorf("expected CheckWarn, got %v, message: %s", check.Status.StatusLabel(), check.Message)
	}
	if !strings.Contains(check.Message, "insights") {
		t.Errorf("message should mention missing insights, got: %s", check.Message)
	}
}

// --- Fix 2: skills-ref subDir existence alone doesn't produce OK ---

func TestCheckSkillsRefToolchain_SubDirExistsButNotOnPath(t *testing.T) {
	// given — skills-ref NOT on PATH, uv IS on PATH, subDir exists
	restoreLookPath := session.OverrideLookPath(func(cmd string) (string, error) {
		if cmd == "uv" {
			return "/usr/local/bin/uv", nil
		}
		return "", fmt.Errorf("not found: %s", cmd)
	})
	defer restoreLookPath()
	restoreFindDir := session.OverrideFindSkillsRefDir(func() string {
		return "/some/path/skills-ref"
	})
	defer restoreFindDir()

	// when
	checks := session.ExportCheckSkillsRefToolchain(false)

	// then — should be WARN, not OK
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
	if checks[0].Status == domain.CheckOK {
		t.Errorf("subDir existence alone should NOT produce OK, got: %s", checks[0].Message)
	}
	if checks[0].Status != domain.CheckWarn {
		t.Errorf("expected CheckWarn, got %v", checks[0].Status.StatusLabel())
	}
}

func TestCheckSkillsRefToolchain_InstallSuccessButNotOnPath(t *testing.T) {
	// given — skills-ref NOT on PATH, uv IS on PATH, no subDir, install succeeds but still not on PATH
	restoreLookPath := session.OverrideLookPath(func(cmd string) (string, error) {
		if cmd == "uv" {
			return "/usr/local/bin/uv", nil
		}
		return "", fmt.Errorf("not found: %s", cmd)
	})
	defer restoreLookPath()
	restoreFindDir := session.OverrideFindSkillsRefDir(func() string {
		return ""
	})
	defer restoreFindDir()
	restoreInstall := session.OverrideInstallSkillsRef(func() error {
		return nil // install succeeds
	})
	defer restoreInstall()

	// when — repair=true
	checks := session.ExportCheckSkillsRefToolchain(true)

	// then — should be WARN (install succeeded but not on PATH)
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
	if checks[0].Status == domain.CheckFixed {
		t.Error("should NOT report FIXED when executable not found on PATH after install")
	}
	if checks[0].Status != domain.CheckWarn {
		t.Errorf("expected CheckWarn, got %v, message: %s", checks[0].Status.StatusLabel(), checks[0].Message)
	}
	if !strings.Contains(checks[0].Hint, "PATH") {
		t.Errorf("hint should mention PATH, got: %s", checks[0].Hint)
	}
}
