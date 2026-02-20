package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand_Output(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version"})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "paintress") {
		t.Errorf("output = %q, want to contain 'paintress'", out)
	}
	if !strings.Contains(out, Version) {
		t.Errorf("output = %q, want to contain version %q", out, Version)
	}
}

func TestVersionCommand_NoArgs(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version", "extra"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for extra args, got nil")
	}
}
