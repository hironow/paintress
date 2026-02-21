package cmd

import (
	"bytes"
	"testing"
)

func TestUpdateCommand_NoArgs(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"update"})

	// when
	err := cmd.Execute()

	// then — dev build guard: should succeed and print dev build message
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if len(out) == 0 {
		t.Error("expected output, got empty string")
	}
}

func TestUpdateCommand_CheckFlag(t *testing.T) {
	// given
	cmd := NewRootCommand()

	// when
	updateCmd, _, err := cmd.Find([]string{"update"})

	// then
	if err != nil {
		t.Fatalf("update subcommand not found: %v", err)
	}
	f := updateCmd.Flags().Lookup("check")
	if f == nil {
		t.Fatal("--check flag not found on update command")
	}
	if f.DefValue != "false" {
		t.Errorf("--check default = %q, want %q", f.DefValue, "false")
	}
}

func TestUpdateCommand_RejectsArgs(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"update", "unexpected"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for extra args, got nil")
	}
}
