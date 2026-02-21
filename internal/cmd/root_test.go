package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCommand_Use(t *testing.T) {
	// given / when
	cmd := NewRootCommand()

	// then
	if cmd.Use != "paintress" {
		t.Errorf("Use = %q, want %q", cmd.Use, "paintress")
	}
}

func TestNewRootCommand_PersistentFlags_Output(t *testing.T) {
	// given
	cmd := NewRootCommand()

	// when
	f := cmd.PersistentFlags().Lookup("output")

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
	cmd := NewRootCommand()

	// when
	f := cmd.PersistentFlags().Lookup("lang")

	// then
	if f == nil {
		t.Fatal("--lang PersistentFlag not found")
	}
	if f.DefValue != "en" {
		t.Errorf("--lang default = %q, want %q", f.DefValue, "en")
	}
	if f.Shorthand != "l" {
		t.Errorf("--lang shorthand = %q, want %q", f.Shorthand, "l")
	}
}

func TestNewRootCommand_PersistentFlags_Verbose(t *testing.T) {
	// given
	cmd := NewRootCommand()

	// when
	f := cmd.PersistentFlags().Lookup("verbose")

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

func TestNewRootCommand_PersistentFlags_Config(t *testing.T) {
	// given
	cmd := NewRootCommand()

	// when
	f := cmd.PersistentFlags().Lookup("config")

	// then
	if f == nil {
		t.Fatal("--config PersistentFlag not found")
	}
	if f.DefValue != "" {
		t.Errorf("--config default = %q, want %q", f.DefValue, "")
	}
	if f.Shorthand != "c" {
		t.Errorf("--config shorthand = %q, want %q", f.Shorthand, "c")
	}
}

func TestNewRootCommand_HasSubcommands(t *testing.T) {
	// given
	cmd := NewRootCommand()

	// when
	subs := cmd.Commands()

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
	root := NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	if NeedsDefaultRun(root, args) {
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
	root := NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	if NeedsDefaultRun(root, args) {
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
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	// when
	err := cmd.Execute()

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
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "paintress") {
		t.Errorf("version output = %q, want to contain 'paintress'", out)
	}
}
