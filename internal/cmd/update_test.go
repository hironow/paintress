package cmd

import (
	"bytes"
	"testing"
)

func TestUpdateCommand_NoArgs(t *testing.T) {
	// given — verify command accepts no args (flag wiring only, no network)
	cmd := NewRootCommand()
	updateCmd, _, err := cmd.Find([]string{"update"})

	// then
	if err != nil {
		t.Fatalf("update subcommand not found: %v", err)
	}
	if updateCmd.Args == nil {
		t.Fatal("Args validator not set on update command")
	}
	// cobra.NoArgs rejects any positional args
	if err := updateCmd.Args(updateCmd, []string{}); err != nil {
		t.Errorf("unexpected error for zero args: %v", err)
	}
}

func TestUpdateCommand_CheckShortAlias(t *testing.T) {
	// given
	root := NewRootCommand()
	updateCmd, _, err := root.Find([]string{"update"})
	if err != nil {
		t.Fatalf("find update command: %v", err)
	}

	// then
	f := updateCmd.Flags().Lookup("check")
	if f == nil {
		t.Fatal("--check flag not found")
	}
	if f.Shorthand != "C" {
		t.Errorf("--check shorthand = %q, want %q", f.Shorthand, "C")
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
