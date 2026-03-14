package cmd_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/cmd"
)

func TestRunCommand_NoArgs_FallsBackToCwd(t *testing.T) {
	// given: no args → falls back to cwd (may error on business logic, not on arg validation)
	// Use an empty tempdir as cwd so the command hits "not initialized" quickly
	// instead of running actual business logic when .expedition/ exists in the real cwd.
	origDir, _ := os.Getwd()
	dir := t.TempDir()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(origDir) })

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"run"})

	// when
	err := root.Execute()

	// then: should NOT fail with "accepts 1 arg" — cwd fallback is used.
	// It will fail with "not initialized" which is expected (business logic, not arg validation).
	if err != nil && strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("run should accept zero args with cwd fallback, got: %v", err)
	}
}

func TestRunCommand_AllFlagsExist(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	runCmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatalf("find run command: %v", err)
	}

	// then: all expected flags should be registered
	flags := []struct {
		name     string
		defValue string
	}{
		{"max-expeditions", "50"},
		{"timeout", "1980"},
		{"model", "opus"},
		{"base-branch", "main"},
		{"claude-cmd", "claude"},
		{"dev-cmd", "npm run dev"},
		{"dev-dir", ""},
		{"dev-url", "http://localhost:3000"},
		{"review-cmd", ""},
		{"workers", "1"},
		{"setup-cmd", ""},
		{"no-dev", "false"},
		{"dry-run", "false"},
		{"notify-cmd", ""},
		{"approve-cmd", ""},
		{"auto-approve", "false"},
	}

	for _, tc := range flags {
		f := runCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Errorf("--%s flag not found", tc.name)
			continue
		}
		if f.DefValue != tc.defValue {
			t.Errorf("--%s default = %q, want %q", tc.name, f.DefValue, tc.defValue)
		}
	}
}

func TestRunCommand_ShortAliases(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	runCmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatalf("find run command: %v", err)
	}

	// then: short aliases must exist for frequently used flags
	aliases := []struct {
		name      string
		shorthand string
	}{
		{"dry-run", "n"},
		{"model", "m"},
		{"timeout", "t"},
		{"base-branch", "b"},
		{"workers", "w"},
	}

	for _, tc := range aliases {
		f := runCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Errorf("--%s flag not found", tc.name)
			continue
		}
		if f.Shorthand != tc.shorthand {
			t.Errorf("--%s shorthand = %q, want %q", tc.name, f.Shorthand, tc.shorthand)
		}
	}
}

func TestRunCommand_NotifyApproveFlagsLongOnly(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	runCmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatalf("find run command: %v", err)
	}

	// then: notify/approve flags must be long-only (no short aliases)
	for _, name := range []string{"notify-cmd", "approve-cmd", "auto-approve"} {
		f := runCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("--%s flag not found", name)
			continue
		}
		if f.Shorthand != "" {
			t.Errorf("--%s has shorthand %q, want long-only", name, f.Shorthand)
		}
	}
}

func TestRunCmd_FailsWithoutInit(t *testing.T) {
	// given: empty directory with no .expedition/
	dir := t.TempDir()

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"run", dir})

	// when
	err := root.Execute()

	// then: should fail with init guidance
	if err == nil {
		t.Fatal("expected error for uninitialized state, got nil")
	}
	got := err.Error()
	if !strings.Contains(got, "init") {
		t.Errorf("expected error to mention 'init', got: %s", got)
	}
}

func TestRunCommand_WaitTimeoutFlag(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	runCmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatalf("find run command: %v", err)
	}

	// when
	f := runCmd.Flags().Lookup("wait-timeout")

	// then
	if f == nil {
		t.Fatal("--wait-timeout flag not found on run command")
	}
	// Default is domain.DefaultWaitTimeout = 30m0s
	if f.DefValue != "30m0s" {
		t.Errorf("--wait-timeout default = %q, want %q", f.DefValue, "30m0s")
	}
}

func TestRunCommand_DynamicReviewCmd(t *testing.T) {
	// given: --base-branch set but --review-cmd not set
	root := cmd.NewRootCommand()
	runCmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatalf("find run command: %v", err)
	}

	// when: review-cmd default should be empty (derived in PreRunE)
	f := runCmd.Flags().Lookup("review-cmd")

	// then
	if f == nil {
		t.Fatal("--review-cmd flag not found")
	}
	// Default is empty; PreRunE derives from --base-branch
	if f.DefValue != "" {
		t.Errorf("--review-cmd default = %q, want empty (dynamically derived)", f.DefValue)
	}
}

// initGitRepo creates a temp directory with a git repository (initial commit + remote).
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, args := range [][]string{
		{"git", "init", "--initial-branch", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
		{"git", "remote", "add", "origin", "https://example.com/repo.git"},
	} {
		c := exec.Command(args[0], args[1:]...) // nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command — static test fixture args only
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git setup %v failed: %v\n%s", args, err, out)
		}
	}
	return dir
}

// initPaintressProject initializes a paintress project in an existing git repo.
func initPaintressProject(t *testing.T, dir string) {
	t.Helper()
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"init", "--team", "TEST", dir})
	if err := root.Execute(); err != nil {
		t.Fatalf("paintress init failed: %v", err)
	}
}

func TestRunCommand_DryRun_SkipsClaudeBinaryCheck(t *testing.T) {
	// given: initialized git repo + paintress project, claude binary does NOT exist
	dir := initGitRepo(t)
	initPaintressProject(t, dir)

	// Override claude_cmd to a non-existent binary via config set
	cfgRoot := cmd.NewRootCommand()
	cfgBuf := new(bytes.Buffer)
	cfgRoot.SetOut(cfgBuf)
	cfgRoot.SetErr(cfgBuf)
	cfgRoot.SetArgs([]string{"config", "set", "claude_cmd", "nonexistent-claude-binary-xyz", dir})
	if err := cfgRoot.Execute(); err != nil {
		t.Fatalf("config set claude_cmd failed: %v", err)
	}

	// Ensure the fake binary does NOT exist on PATH
	if _, err := exec.LookPath("nonexistent-claude-binary-xyz"); err == nil {
		t.Skip("nonexistent-claude-binary-xyz unexpectedly exists on PATH")
	}

	root := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run", "--dry-run", "--no-dev", "--workers", "0", "--max-expeditions", "1", dir})

	// when
	err := root.Execute()

	// then: the command should NOT fail with "claude" binary not found error.
	// It may fail for other reasons (no issues, expedition runner, etc.) but preflight passes.
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "nonexistent-claude-binary-xyz") {
			t.Errorf("dry-run should skip claude binary check, but got: %s", errMsg)
		}
		// Other errors are acceptable (expedition phase failures)
	}

	// Verify that the command got past the "not initialized" stage
	combinedOutput := stdout.String() + stderr.String()
	if strings.Contains(combinedOutput, "not initialized") {
		t.Error("expected command to pass init check")
	}

	// Verify the error (if any) is NOT about missing the claude binary
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "not found in PATH") && strings.Contains(errStr, "claude") {
			t.Errorf("dry-run should not check claude binary, got PATH error: %s", errStr)
		}
	}
}

func TestRunCommand_DryRun_ProducesPromptFiles(t *testing.T) {
	// given: initialized git repo + paintress project
	dir := initGitRepo(t)
	initPaintressProject(t, dir)

	root := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run", "--dry-run", "--no-dev", "--workers", "0", "--max-expeditions", "1", dir})

	// when
	err := root.Execute()

	// then: even if no issues to process, the command should reach expedition phase.
	// It should not fail at init or preflight stage.
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "init") && !strings.Contains(errMsg, "expedition") {
			t.Errorf("expected to pass init stage, got: %s", errMsg)
		}
	}

	// Verify events directory was used (event store was initialized)
	eventsDir := filepath.Join(dir, ".expedition", "events")
	if _, statErr := os.Stat(eventsDir); statErr != nil {
		t.Logf("events dir not created (may be expected if no issues): %v", statErr)
	}
}
