package cmd

// white-box-reason: cobra command construction: NewRootCommand and CLI routing are unexported

import (
	"bytes"
	"strings"
	"testing"
)

func TestIssuesCommand_NoArgs_FallsBackToCwd(t *testing.T) {
	// given: no args → falls back to cwd (may error on business logic, not on arg validation)
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"issues"})

	// when
	err := cmd.Execute()

	// then: should NOT fail with "accepts 1 arg" — cwd fallback is used
	if err != nil && strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("issues should accept zero args with cwd fallback, got: %v", err)
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

func TestIssuesCommand_StateShortAlias(t *testing.T) {
	// given
	root := NewRootCommand()
	issuesCmd, _, err := root.Find([]string{"issues"})
	if err != nil {
		t.Fatalf("find issues command: %v", err)
	}

	// then
	f := issuesCmd.Flags().Lookup("state")
	if f == nil {
		t.Fatal("--state flag not found")
	}
	if f.Shorthand != "s" {
		t.Errorf("--state shorthand = %q, want %q", f.Shorthand, "s")
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
