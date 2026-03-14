package cmd_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/cmd"
	"github.com/hironow/paintress/internal/domain"
	"github.com/spf13/cobra"
)

func TestInitCommand_NoArgs_FallsBackToCwd(t *testing.T) {
	// given — no repo-path arg; resolveRepoPath falls back to os.Getwd()
	// Use a tempdir as cwd so init hits "already exists" (if seeded) or succeeds
	// without polluting the real working directory with .expedition/.
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(origDir) })

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"init"})

	// when
	err := root.Execute()

	// then — should NOT fail with "accepts 1 arg(s), received 0"
	if err != nil && strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("init should accept 0 args (cwd fallback), got: %v", err)
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

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"init", dir})

	// when
	err := root.Execute()

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
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("")) // empty stdin — must NOT hang
	root.SetArgs([]string{"init", "--team", "MY", "--project", "Hades", dir})

	// when
	err := root.Execute()

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
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("")) // empty stdin
	root.SetArgs([]string{"init", dir})

	// when
	err := root.Execute()

	// then — should succeed with empty defaults
	if err != nil {
		t.Fatalf("init with defaults failed: %v", err)
	}
	cfgPath := domain.ProjectConfigPath(dir)
	if _, readErr := os.Stat(cfgPath); readErr != nil {
		t.Fatalf("config not created: %v", readErr)
	}
}

func TestInitCommand_AlreadyExists_SuggestsForce(t *testing.T) {
	// given: .expedition/config.yaml already exists
	dir := t.TempDir()
	cfgDir := dir + "/.expedition"
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("create expedition dir: %v", err)
	}
	cfgPath := domain.ProjectConfigPath(dir)
	if err := os.WriteFile(cfgPath, []byte("tracker:\n  team: OLD\n"), 0644); err != nil {
		t.Fatalf("create config: %v", err)
	}

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"init", dir})

	// when
	err := root.Execute()

	// then
	if err == nil {
		t.Fatal("expected error when config already exists")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("expected '--force' hint in error, got: %v", err)
	}
}

func TestInitCommand_Force_OverwritesExisting(t *testing.T) {
	// given: existing config with old content
	dir := t.TempDir()
	cfgDir := dir + "/.expedition"
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("create expedition dir: %v", err)
	}
	cfgPath := domain.ProjectConfigPath(dir)
	if err := os.WriteFile(cfgPath, []byte("tracker:\n  team: OLD\n"), 0644); err != nil {
		t.Fatalf("create config: %v", err)
	}

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"init", "--force", "--team", "NEW", "--project", "NewProject", dir})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("init --force failed: %v", err)
	}
	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if !strings.Contains(content, "NEW") {
		t.Errorf("expected 'NEW' in overwritten config, got:\n%s", content)
	}
}

func TestInitCommand_Force_MergesExisting(t *testing.T) {
	// given: existing config with user customization
	dir := t.TempDir()
	cfgDir := dir + "/.expedition"
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("create expedition dir: %v", err)
	}
	cfgPath := domain.ProjectConfigPath(dir)
	if err := os.WriteFile(cfgPath, []byte("tracker:\n  team: OLD\nlang: en\n"), 0644); err != nil {
		t.Fatalf("create config: %v", err)
	}

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"init", "--force", "--team", "NEW", dir})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("init --force failed: %v", err)
	}
	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	// CLI team should win
	if !strings.Contains(content, "NEW") {
		t.Errorf("expected CLI team 'NEW', got:\n%s", content)
	}
	// User's lang=en should be preserved
	if !strings.Contains(content, "lang: en") {
		t.Errorf("expected user's lang=en preserved, got:\n%s", content)
	}
}

func TestInitCommand_ConfigHasDefaults(t *testing.T) {
	// given: fresh init
	dir := t.TempDir()
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"init", "--team", "MY", dir})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	data, _ := os.ReadFile(domain.ProjectConfigPath(dir))
	content := string(data)
	if !strings.Contains(content, "lang: ja") {
		t.Errorf("expected default lang=ja, got:\n%s", content)
	}
}

func TestInitCommand_AcceptsRepoPath(t *testing.T) {
	// given — provide deterministic stdin to avoid hanging
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("MY\nmy-project\n"))
	root.SetArgs([]string{"init", t.TempDir()})

	// when
	err := root.Execute()

	// then: args validation should pass; business logic may fail
	if err == nil {
		return // success
	}
	if strings.Contains(err.Error(), "accepts") && strings.Contains(err.Error(), "arg") {
		t.Fatalf("init should accept repo-path arg: %v", err)
	}
}

func TestInitCommand_OtelFlags_Exist(t *testing.T) {
	// given
	root := cmd.NewRootCommand()

	// when — find init subcommand
	var initCmd *cobra.Command
	for _, sub := range root.Commands() {
		if sub.Name() == "init" {
			initCmd = sub
			break
		}
	}
	if initCmd == nil {
		t.Fatal("init subcommand not found")
	}

	// then — otel flags exist
	for _, flag := range []string{"otel-backend", "otel-entity", "otel-project"} {
		if initCmd.Flags().Lookup(flag) == nil {
			t.Errorf("init flag --%s not found", flag)
		}
	}
}
