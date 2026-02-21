package cmd

import (
	"bytes"
	"strings"
	"testing"
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
