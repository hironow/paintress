package cmd

import (
	"bytes"
	"testing"
)

func TestDoctorCommand_NoArgs(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"doctor"})

	// when
	err := cmd.Execute()

	// then: should succeed with no args
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoctorCommand_RejectsArgs(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"doctor", "extra-arg"})

	// when
	err := cmd.Execute()

	// then: should reject positional args
	if err == nil {
		t.Fatal("expected error for extra arg, got nil")
	}
}

func TestDoctorCommand_OutputFlagDefault(t *testing.T) {
	// given
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"doctor"})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFlag, err := cmd.PersistentFlags().GetString("output")
	if err != nil {
		t.Fatalf("get output flag: %v", err)
	}
	if outputFlag != "text" {
		t.Errorf("output = %q, want %q", outputFlag, "text")
	}
}

func TestDoctorCommand_OutputFlagJSON(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"doctor", "--output", "json"})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFlag, err := cmd.PersistentFlags().GetString("output")
	if err != nil {
		t.Fatalf("get output flag: %v", err)
	}
	if outputFlag != "json" {
		t.Errorf("output = %q, want %q", outputFlag, "json")
	}
}
