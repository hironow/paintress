package cmd_test

import (
	"bytes"
	"os"
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
