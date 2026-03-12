package cmd_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/cmd"
)

func TestVersionCommand_Output(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"version"})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// Format: "paintress vVERSION (commit: COMMIT, date: DATE, go: goX.Y.Z)"
	if !strings.Contains(out, "paintress") {
		t.Errorf("output = %q, want to contain 'paintress'", out)
	}
	if !strings.Contains(out, cmd.Version) {
		t.Errorf("output = %q, want to contain version %q", out, cmd.Version)
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
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"version", "--json"})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var info map[string]string
	if err := json.Unmarshal(buf.Bytes(), &info); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, buf.String())
	}

	for _, key := range []string{"version", "commit", "date", "go", "os", "arch"} {
		if _, ok := info[key]; !ok {
			t.Errorf("JSON output missing key %q", key)
		}
	}
	if info["version"] != cmd.Version {
		t.Errorf("version = %q, want %q", info["version"], cmd.Version)
	}
}

func TestVersionCommand_JSONShortAlias(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
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

func TestVersionCommand_GoReleaserVersion(t *testing.T) {
	// given — GoReleaser sets Version WITHOUT v prefix (e.g. "1.2.3")
	origVersion := cmd.Version
	cmd.Version = "1.2.3"
	defer func() { cmd.Version = origVersion }()

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"version"})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "v1.2.3") {
		t.Errorf("output = %q, want to contain 'v1.2.3'", out)
	}
}

func TestVersionCommand_NoArgs(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"version", "extra"})

	// when
	err := root.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for extra args, got nil")
	}
}
