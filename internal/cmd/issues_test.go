package cmd

import (
	"bytes"
	"testing"
)

func TestIssuesCommand_RequiresRepoPath(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"issues"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for missing repo-path, got nil")
	}
}

func TestIssuesCommand_StateFlagDefault(t *testing.T) {
	// given
	root := NewRootCommand()
	issuesCmd, _, err := root.Find([]string{"issues"})
	if err != nil {
		t.Fatalf("find issues command: %v", err)
	}

	// when
	f := issuesCmd.Flags().Lookup("state")

	// then
	if f == nil {
		t.Fatal("--state flag not found on issues command")
	}
	if f.DefValue != "" {
		t.Errorf("--state default = %q, want empty", f.DefValue)
	}
}

func TestIssuesCommand_OutputFlagInherited(t *testing.T) {
	// given: --output is a PersistentFlag on root, issues should inherit it
	root := NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"issues", "--output", "json", t.TempDir()})

	// when
	err := root.Execute()

	// then: may error due to missing config, but flag should be parsed
	_ = err // business logic may fail; we just care about flag inheritance

	outputFlag, err := root.PersistentFlags().GetString("output")
	if err != nil {
		t.Fatalf("get output flag: %v", err)
	}
	if outputFlag != "json" {
		t.Errorf("output = %q, want %q", outputFlag, "json")
	}
}
