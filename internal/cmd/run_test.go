package cmd

import (
	"bytes"
	"testing"
)

func TestRunCommand_RequiresRepoPath(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"run"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for missing repo-path, got nil")
	}
}

func TestRunCommand_AllFlagsExist(t *testing.T) {
	// given
	root := NewRootCommand()
	runCmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatalf("find run command: %v", err)
	}

	// then: all expected flags should be registered
	flags := []struct {
		name     string
		defValue string
	}{
		{"max-expeditions", "50"},
		{"timeout", "1980"},
		{"model", "opus"},
		{"base-branch", "main"},
		{"claude-cmd", "claude"},
		{"dev-cmd", "npm run dev"},
		{"dev-dir", ""},
		{"dev-url", "http://localhost:3000"},
		{"review-cmd", ""},
		{"workers", "1"},
		{"setup-cmd", ""},
		{"no-dev", "false"},
		{"dry-run", "false"},
	}

	for _, tc := range flags {
		f := runCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Errorf("--%s flag not found", tc.name)
			continue
		}
		if f.DefValue != tc.defValue {
			t.Errorf("--%s default = %q, want %q", tc.name, f.DefValue, tc.defValue)
		}
	}
}

func TestRunCommand_ShortAliases(t *testing.T) {
	// given
	root := NewRootCommand()
	runCmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatalf("find run command: %v", err)
	}

	// then: short aliases must exist for frequently used flags
	aliases := []struct {
		name      string
		shorthand string
	}{
		{"dry-run", "n"},
		{"model", "m"},
		{"timeout", "t"},
		{"base-branch", "b"},
		{"workers", "w"},
	}

	for _, tc := range aliases {
		f := runCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Errorf("--%s flag not found", tc.name)
			continue
		}
		if f.Shorthand != tc.shorthand {
			t.Errorf("--%s shorthand = %q, want %q", tc.name, f.Shorthand, tc.shorthand)
		}
	}
}

func TestRunCommand_DynamicReviewCmd(t *testing.T) {
	// given: --base-branch set but --review-cmd not set
	root := NewRootCommand()
	runCmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatalf("find run command: %v", err)
	}

	// when: review-cmd default should be empty (derived in PreRunE)
	f := runCmd.Flags().Lookup("review-cmd")

	// then
	if f == nil {
		t.Fatal("--review-cmd flag not found")
	}
	// Default is empty; PreRunE derives from --base-branch
	if f.DefValue != "" {
		t.Errorf("--review-cmd default = %q, want empty (dynamically derived)", f.DefValue)
	}
}
