package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
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
	cfgPath := domain.ProjectConfigPath(dir)
	if err := os.WriteFile(cfgPath, []byte("tracker:\n  team: MY\n"), 0644); err != nil {
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

// === P1-5: Flag-based init (no interactive prompts) ===

func TestInitCmd_FlagsOnly(t *testing.T) {
	// given — init via cobra command with flags, no stdin
	dir := t.TempDir()
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetIn(strings.NewReader("")) // empty stdin — must NOT hang
	cmd.SetArgs([]string{"init", "--team", "MY", "--project", "Hades", dir})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("init with flags failed: %v", err)
	}
	cfgPath := domain.ProjectConfigPath(dir)
	data, readErr := os.ReadFile(cfgPath)
	if readErr != nil {
		t.Fatalf("config not created: %v", readErr)
	}
	content := string(data)
	if !strings.Contains(content, "MY") {
		t.Errorf("expected team in config, got:\n%s", content)
	}
	if !strings.Contains(content, "Hades") {
		t.Errorf("expected project in config, got:\n%s", content)
	}
}

func TestInitCmd_MissingFlags_UsesDefaults(t *testing.T) {
	// given — init with no flags, should use defaults (no hang)
	dir := t.TempDir()
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetIn(strings.NewReader("")) // empty stdin
	cmd.SetArgs([]string{"init", dir})

	// when
	err := cmd.Execute()

	// then — should succeed with empty defaults
	if err != nil {
		t.Fatalf("init with defaults failed: %v", err)
	}
	cfgPath := domain.ProjectConfigPath(dir)
	if _, readErr := os.Stat(cfgPath); readErr != nil {
		t.Fatalf("config not created: %v", readErr)
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
