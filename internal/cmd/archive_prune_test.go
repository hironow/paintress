package cmd

import (
	"bytes"
	"testing"
)

func TestArchivePruneCommand_RequiresRepoPath(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"archive-prune"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for missing repo-path, got nil")
	}
}

func TestArchivePruneCommand_DaysFlagDefault(t *testing.T) {
	// given
	root := NewRootCommand()
	pruneCmd, _, err := root.Find([]string{"archive-prune"})
	if err != nil {
		t.Fatalf("find archive-prune command: %v", err)
	}

	// when
	f := pruneCmd.Flags().Lookup("days")

	// then
	if f == nil {
		t.Fatal("--days flag not found")
	}
	if f.DefValue != "30" {
		t.Errorf("--days default = %q, want %q", f.DefValue, "30")
	}
}

func TestArchivePruneCommand_ExecuteFlagDefault(t *testing.T) {
	// given
	root := NewRootCommand()
	pruneCmd, _, err := root.Find([]string{"archive-prune"})
	if err != nil {
		t.Fatalf("find archive-prune command: %v", err)
	}

	// when
	f := pruneCmd.Flags().Lookup("execute")

	// then
	if f == nil {
		t.Fatal("--execute flag not found")
	}
	if f.DefValue != "false" {
		t.Errorf("--execute default = %q, want %q", f.DefValue, "false")
	}
}

func TestArchivePruneCommand_ShortAliases(t *testing.T) {
	// given
	root := NewRootCommand()
	pruneCmd, _, err := root.Find([]string{"archive-prune"})
	if err != nil {
		t.Fatalf("find archive-prune command: %v", err)
	}

	// then
	aliases := []struct {
		name      string
		shorthand string
	}{
		{"days", "d"},
		{"execute", "x"},
	}

	for _, tc := range aliases {
		f := pruneCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Errorf("--%s flag not found", tc.name)
			continue
		}
		if f.Shorthand != tc.shorthand {
			t.Errorf("--%s shorthand = %q, want %q", tc.name, f.Shorthand, tc.shorthand)
		}
	}
}

func TestArchivePruneCommand_NegativeDays(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"archive-prune", t.TempDir(), "--days", "-5"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for negative --days, got nil")
	}
}

func TestArchivePruneCommand_ZeroDays(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"archive-prune", t.TempDir(), "--days", "0"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for zero --days, got nil")
	}
}

func TestArchivePruneCommand_DryRunText(t *testing.T) {
	// given: temp dir with no archive → no candidates
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"archive-prune", t.TempDir()})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchivePruneCommand_DryRunJSON(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"archive-prune", t.TempDir(), "--output", "json"})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Error("expected JSON output, got empty")
	}
}
