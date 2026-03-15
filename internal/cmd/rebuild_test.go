package cmd_test

import (
	"bytes"
	"testing"

	"github.com/hironow/paintress/internal/cmd"
)

func TestRebuildCommand_SubcommandExists(t *testing.T) {
	// given
	root := cmd.NewRootCommand()

	// when
	rebuildCmd, _, err := root.Find([]string{"rebuild"})

	// then
	if err != nil {
		t.Fatalf("find rebuild command: %v", err)
	}
	if rebuildCmd.Name() != "rebuild" {
		t.Errorf("expected command name 'rebuild', got %q", rebuildCmd.Name())
	}
}

func TestRebuildCommand_WithoutInit_Succeeds(t *testing.T) {
	// given: empty directory with no .expedition/ — rebuild replays 0 events
	dir := t.TempDir()

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"rebuild", dir})

	// when
	err := root.Execute()

	// then: rebuild gracefully handles missing events (0 projections)
	if err != nil {
		t.Fatalf("expected rebuild to succeed on empty dir, got: %v", err)
	}
}
