package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	cmd "github.com/hironow/paintress/internal/cmd"
	"github.com/hironow/paintress/internal/domain"
	"github.com/spf13/cobra"
)

func TestRootCommand_Help(t *testing.T) {
	rootCmd := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected help output, got empty string")
	}
}

func TestRootCommand_UnknownSubcommand(t *testing.T) {
	rootCmd := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestSubcommands_Exist(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	expected := []string{"init", "doctor", "run", "status", "clean", "archive-prune", "version", "update"}
	for _, name := range expected {
		found := false
		for _, c := range rootCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestRootCommand_PersistentFlags(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	flags := []struct {
		long  string
		short string
	}{
		{"verbose", "v"},
	}
	for _, f := range flags {
		flag := rootCmd.PersistentFlags().Lookup(f.long)
		if flag == nil {
			t.Errorf("root command missing persistent flag %q", f.long)
			continue
		}
		if flag.Shorthand != f.short {
			t.Errorf("flag %q: shorthand = %q, want %q", f.long, flag.Shorthand, f.short)
		}
	}
}

func TestRootCommand_VersionFlag(t *testing.T) {
	rootCmd := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "paintress version") {
		t.Errorf("--version output should contain 'paintress version', got %q", output)
	}
}

func TestVersionCommand_Output(t *testing.T) {
	rootCmd := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "paintress") {
		t.Error("version output should contain 'paintress'")
	}
	if !strings.Contains(output, "commit:") {
		t.Error("version output should contain 'commit:'")
	}
}

func TestVersionCommand_JSONFlag(t *testing.T) {
	for _, flag := range []string{"--json", "-j"} {
		t.Run(flag, func(t *testing.T) {
			rootCmd := cmd.NewRootCommand()
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"version", flag})

			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, `"version"`) {
				t.Error("JSON output should contain 'version' key")
			}
			if !strings.Contains(output, `"commit"`) {
				t.Error("JSON output should contain 'commit' key")
			}
		})
	}
}

func TestHandleError_ExitErrorIsSilent(t *testing.T) {
	// given: an ExitError wrapping a message
	exitErr := &cmd.ExitError{Code: 130, Err: fmt.Errorf("interrupted")}
	buf := new(bytes.Buffer)

	// when: handleError is called
	code := handleError(exitErr, buf)

	// then: exit code should be preserved but no message printed
	if code != 130 {
		t.Errorf("exit code = %d, want 130", code)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no stderr output for ExitError, got %q", buf.String())
	}
}

func TestHandleError_SilentErrorIsSilent(t *testing.T) {
	// given: a SilentError
	inner := fmt.Errorf("already printed")
	silentErr := &domain.SilentError{Err: inner}
	buf := new(bytes.Buffer)

	// when
	code := handleError(silentErr, buf)

	// then
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no stderr output for SilentError, got %q", buf.String())
	}
}

func TestHandleError_RegularErrorPrintsMessage(t *testing.T) {
	// given: a regular error
	regularErr := fmt.Errorf("something went wrong")
	buf := new(bytes.Buffer)

	// when
	code := handleError(regularErr, buf)

	// then
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(buf.String(), "something went wrong") {
		t.Errorf("expected error message in output, got %q", buf.String())
	}
}

func TestHandleError_NilReturnsZero(t *testing.T) {
	// given
	buf := new(bytes.Buffer)

	// when
	code := handleError(nil, buf)

	// then
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for nil error, got %q", buf.String())
	}
}

func TestHandleError_WrappedExitErrorIsSilent(t *testing.T) {
	// given: ExitError wrapped in another error
	inner := &cmd.ExitError{Code: 2, Err: fmt.Errorf("deviation")}
	wrapped := fmt.Errorf("run failed: %w", inner)
	buf := new(bytes.Buffer)

	// when
	code := handleError(wrapped, buf)

	// then
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no stderr output for wrapped ExitError, got %q", buf.String())
	}
}

func TestUpdateCommand_HasCheckFlag(t *testing.T) {
	rootCmd := cmd.NewRootCommand()
	var updateCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "update" {
			updateCmd = c
			break
		}
	}
	if updateCmd == nil {
		t.Fatal("update subcommand not found")
	}

	flag := updateCmd.Flags().Lookup("check")
	if flag == nil {
		t.Fatal("update subcommand missing flag 'check'")
	}
	if flag.Shorthand != "C" {
		t.Errorf("flag 'check': shorthand = %q, want %q", flag.Shorthand, "C")
	}
}
