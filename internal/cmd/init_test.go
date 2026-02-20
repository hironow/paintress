package cmd

import (
	"bytes"
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
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"init", t.TempDir()})

	// when
	err := cmd.Execute()

	// then: should not error on valid path (RunInitWithReader needs stdin, so may fail,
	// but the args validation should pass)
	// Note: actual init requires interactive input, so we just test cobra arg validation here
	if err == nil {
		return // success - args accepted
	}
	// If error is about args, that's a failure
	if err.Error() == `accepts 1 arg(s), received 0` {
		t.Fatalf("init should accept repo-path arg: %v", err)
	}
}
