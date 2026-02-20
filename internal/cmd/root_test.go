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

func TestNewRootCommand_HasSubcommands(t *testing.T) {
	// given
	cmd := NewRootCommand()

	// when
	subs := cmd.Commands()

	// then: expect 5 subcommands
	names := make(map[string]bool)
	for _, s := range subs {
		names[s.Name()] = true
	}
	want := []string{"run", "init", "doctor", "issues", "archive-prune"}
	for _, name := range want {
		if !names[name] {
			t.Errorf("subcommand %q not registered", name)
		}
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
