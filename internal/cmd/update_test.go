package cmd

// white-box-reason: cobra command construction: NewRootCommand and CLI routing are unexported

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

func TestUpdateCommand_IsUpToDate(t *testing.T) {
	cases := []struct {
		name     string
		current  string
		latest   string
		upToDate bool
	}{
		{name: "same version", current: "1.0.0", latest: "1.0.0", upToDate: true},
		{name: "current newer", current: "2.0.0", latest: "1.0.0", upToDate: true},
		{name: "current older", current: "1.0.0", latest: "2.0.0", upToDate: false},
		{name: "dev version", current: "dev", latest: "1.0.0", upToDate: false},
		{name: "v-prefixed", current: "v1.0.0", latest: "1.0.0", upToDate: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := isUpToDate(tc.current, tc.latest)

			// then
			if got != tc.upToDate {
				t.Errorf("isUpToDate(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.upToDate)
			}
		})
	}
}
