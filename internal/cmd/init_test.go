package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hironow/paintress"
)

func TestInitCommand_RequiresRepoPath(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"init"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for missing repo-path, got nil")
	}
}

func TestInitCommand_AlreadyInitialized(t *testing.T) {
	// given: .expedition/config.yaml already exists
	dir := t.TempDir()
	cfgDir := dir + "/.expedition"
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("create expedition dir: %v", err)
	}
	cfgPath := paintress.ProjectConfigPath(dir)
	if err := os.WriteFile(cfgPath, []byte("linear:\n  team: MY\n"), 0644); err != nil {
		t.Fatalf("create config: %v", err)
	}

	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"init", dir})

	// when
	err := cmd.Execute()

	// then: should fail with "already exists" or "already initialized"
	if err == nil {
		t.Fatal("expected error for already initialized, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "already exists") && !strings.Contains(got, "already initialized") {
		t.Errorf("expected 'already exists' or 'already initialized' in error, got: %s", got)
	}
}

func TestInitCommand_AcceptsRepoPath(t *testing.T) {
	// given — provide deterministic stdin to avoid hanging
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetIn(strings.NewReader("MY\nmy-project\n"))
	cmd.SetArgs([]string{"init", t.TempDir()})

	// when
	err := cmd.Execute()

	// then: args validation should pass; business logic may fail
	if err == nil {
		return // success
	}
	if err.Error() == `accepts 1 arg(s), received 0` {
		t.Fatalf("init should accept repo-path arg: %v", err)
	}
}
