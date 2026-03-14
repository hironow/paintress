package cmd_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/cmd"
)

func TestNewRootCommand_Use(t *testing.T) {
	// given / when
	root := cmd.NewRootCommand()

	// then
	if root.Use != "paintress" {
		t.Errorf("Use = %q, want %q", root.Use, "paintress")
	}
}

func TestNewRootCommand_PersistentFlags_Output(t *testing.T) {
	// given
	root := cmd.NewRootCommand()

	// when
	f := root.PersistentFlags().Lookup("output")

	// then
	if f == nil {
		t.Fatal("--output PersistentFlag not found")
	}
	if f.DefValue != "text" {
		t.Errorf("--output default = %q, want %q", f.DefValue, "text")
	}
	if f.Shorthand != "o" {
		t.Errorf("--output shorthand = %q, want %q", f.Shorthand, "o")
	}
}

func TestNewRootCommand_PersistentFlags_Lang(t *testing.T) {
	// given
	root := cmd.NewRootCommand()

	// when
	f := root.PersistentFlags().Lookup("lang")

	// then
	if f == nil {
		t.Fatal("--lang PersistentFlag not found")
	}
	if f.DefValue != "" {
		t.Errorf("--lang default = %q, want %q (empty = config default)", f.DefValue, "")
	}
	if f.Shorthand != "l" {
		t.Errorf("--lang shorthand = %q, want %q", f.Shorthand, "l")
	}
}

func TestNewRootCommand_PersistentFlags_Verbose(t *testing.T) {
	// given
	root := cmd.NewRootCommand()

	// when
	f := root.PersistentFlags().Lookup("verbose")

	// then
	if f == nil {
		t.Fatal("--verbose PersistentFlag not found")
	}
	if f.DefValue != "false" {
		t.Errorf("--verbose default = %q, want %q", f.DefValue, "false")
	}
	if f.Shorthand != "v" {
		t.Errorf("--verbose shorthand = %q, want %q", f.Shorthand, "v")
	}
}

func TestNewRootCommand_HasSubcommands(t *testing.T) {
	// given
	root := cmd.NewRootCommand()

	// when
	subs := root.Commands()

	// then: expect 6 subcommands
	names := make(map[string]bool)
	for _, s := range subs {
		names[s.Name()] = true
	}
	want := []string{"run", "init", "doctor", "issues", "archive-prune", "version", "update"}
	for _, name := range want {
		if !names[name] {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}

func TestNewRootCommand_BarePathDelegatesToRun(t *testing.T) {
	// given — a bare repo path (no "run" subcommand), using NeedsDefaultRun
	args := []string{"/nonexistent/repo"}
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	if cmd.NeedsDefaultRun(root, args) {
		root.SetArgs(append([]string{"run"}, args...))
	} else {
		root.SetArgs(args)
	}

	// when
	err := root.Execute()

	// then — should fail with a run-related error, NOT "unknown command"
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Contains(err.Error(), "unknown command") {
		t.Errorf("got 'unknown command' error, expected delegation to run: %v", err)
	}
}

func TestNewRootCommand_RunFlagsWithoutSubcommand(t *testing.T) {
	// given — run-specific flags without "run" subcommand
	args := []string{"--model", "opus", "/nonexistent/repo"}
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	if cmd.NeedsDefaultRun(root, args) {
		root.SetArgs(append([]string{"run"}, args...))
	} else {
		root.SetArgs(args)
	}

	// when
	err := root.Execute()

	// then — should fail with a run-related error, NOT "unknown flag"
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("got 'unknown flag' error, expected delegation to run: %v", err)
	}
}

func TestNewRootCommand_NoArgShowsHelp(t *testing.T) {
	// given — no args at all
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{})

	// when
	err := root.Execute()

	// then — should show help (no error)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "paintress") {
		t.Errorf("help output = %q, want to contain 'paintress'", out)
	}
}

func TestNewRootCommand_Version(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--version"})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "paintress") {
		t.Errorf("version output = %q, want to contain 'paintress'", out)
	}
}

func TestNewRootCommand_NoColorFlag(t *testing.T) {
	// given
	root := cmd.NewRootCommand()

	// when
	f := root.PersistentFlags().Lookup("no-color")

	// then
	if f == nil {
		t.Fatal("--no-color PersistentFlag not found")
	}
	if f.DefValue != "false" {
		t.Errorf("--no-color default = %q, want %q", f.DefValue, "false")
	}
}

func TestRootCmd_VerboseIncreasesStderrOutput(t *testing.T) {
	// given: initialized project (uses initProject from config_test.go)
	dir := initProject(t)

	// when: run status without verbose
	root1 := cmd.NewRootCommand()
	var stdout1, stderr1 bytes.Buffer
	root1.SetOut(&stdout1)
	root1.SetErr(&stderr1)
	root1.SetArgs([]string{"status", dir})
	root1.Execute()

	// when: run status WITH verbose
	root2 := cmd.NewRootCommand()
	var stdout2, stderr2 bytes.Buffer
	root2.SetOut(&stdout2)
	root2.SetErr(&stderr2)
	root2.SetArgs([]string{"-v", "status", dir})
	root2.Execute()

	// then: verbose should produce at least as much stderr
	if stderr2.Len() < stderr1.Len() {
		t.Errorf("verbose stderr (%d bytes) should be >= non-verbose stderr (%d bytes)",
			stderr2.Len(), stderr1.Len())
	}
}

func TestRootCmd_NoColorSetsEnv(t *testing.T) {
	// given
	origVal := os.Getenv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	t.Cleanup(func() {
		if origVal != "" {
			os.Setenv("NO_COLOR", origVal)
		} else {
			os.Unsetenv("NO_COLOR")
		}
	})

	dir := initProject(t)

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--no-color", "status", dir})

	// when
	root.Execute()

	// then
	if got := os.Getenv("NO_COLOR"); got == "" {
		t.Error("expected NO_COLOR env to be set after --no-color flag")
	}
	output := buf.String()
	if strings.Contains(output, "\033[") {
		t.Errorf("--no-color output should not contain ANSI codes, got: %q", output)
	}
}
