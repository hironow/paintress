package cmd

import (
	"bytes"
	"encoding/json"
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
	// Format: "paintress vVERSION (commit: COMMIT, date: DATE, go: goX.Y.Z)"
	if !strings.Contains(out, "paintress") {
		t.Errorf("output = %q, want to contain 'paintress'", out)
	}
	if !strings.Contains(out, Version) {
		t.Errorf("output = %q, want to contain version %q", out, Version)
	}
	if !strings.Contains(out, "commit:") {
		t.Errorf("output = %q, want to contain 'commit:'", out)
	}
	if !strings.Contains(out, "date:") {
		t.Errorf("output = %q, want to contain 'date:'", out)
	}
	if !strings.Contains(out, "go:") {
		t.Errorf("output = %q, want to contain 'go:'", out)
	}
}

func TestVersionCommand_JSONOutput(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version", "--json"})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var info map[string]string
	if err := json.Unmarshal(buf.Bytes(), &info); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, buf.String())
	}

	for _, key := range []string{"version", "commit", "date", "go"} {
		if _, ok := info[key]; !ok {
			t.Errorf("JSON output missing key %q", key)
		}
	}
	if info["version"] != Version {
		t.Errorf("version = %q, want %q", info["version"], Version)
	}
}

func TestVersionCommand_JSONShortAlias(t *testing.T) {
	// given
	root := NewRootCommand()
	versionCmd, _, err := root.Find([]string{"version"})
	if err != nil {
		t.Fatalf("find version command: %v", err)
	}

	// then
	f := versionCmd.Flags().Lookup("json")
	if f == nil {
		t.Fatal("--json flag not found")
	}
	if f.Shorthand != "j" {
		t.Errorf("--json shorthand = %q, want %q", f.Shorthand, "j")
	}
}

func TestVersionCommand_NoDoubleV(t *testing.T) {
	// given — simulate git describe output with v prefix
	origVersion := Version
	Version = "v1.2.3"
	defer func() { Version = origVersion }()

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
	if strings.Contains(out, "vv") {
		t.Errorf("output contains double 'v': %q", out)
	}
	if !strings.Contains(out, "v1.2.3") {
		t.Errorf("output = %q, want to contain 'v1.2.3'", out)
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
