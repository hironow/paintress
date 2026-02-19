package main

import (
	"testing"
)

func TestExtractSubcommand_Default(t *testing.T) {
	// Path only → subcmd="run", path="./repo"
	subcmd, repoPath, flags, err := extractSubcommand([]string{"./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	if len(flags) != 0 {
		t.Errorf("flags = %v, want empty", flags)
	}
}

func TestExtractSubcommand_Init(t *testing.T) {
	// "init ./repo" → subcmd="init", path="./repo"
	subcmd, repoPath, flags, err := extractSubcommand([]string{"init", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "init" {
		t.Errorf("subcmd = %q, want %q", subcmd, "init")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	if len(flags) != 0 {
		t.Errorf("flags = %v, want empty", flags)
	}
}

func TestExtractSubcommand_FlagsBeforePath(t *testing.T) {
	// "--model opus ./repo" → subcmd="run", path="./repo", flags=["--model", "opus"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--model", "opus", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--model", "opus"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_Doctor(t *testing.T) {
	// "doctor" → subcmd="doctor", path=""
	subcmd, repoPath, flags, err := extractSubcommand([]string{"doctor"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "doctor" {
		t.Errorf("subcmd = %q, want %q", subcmd, "doctor")
	}
	if repoPath != "" {
		t.Errorf("repoPath = %q, want empty", repoPath)
	}
	if len(flags) != 0 {
		t.Errorf("flags = %v, want empty", flags)
	}
}

func TestExtractSubcommand_VersionFlag(t *testing.T) {
	// "--version" → subcmd="run", path="", flags=["--version"]
	subcmd, _, flags, err := extractSubcommand([]string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	hasVersion := false
	for _, f := range flags {
		if f == "--version" {
			hasVersion = true
		}
	}
	if !hasVersion {
		t.Errorf("flags should contain --version, got %v", flags)
	}
}

func TestExtractSubcommand_FlagsAfterPath(t *testing.T) {
	// "./repo --model opus --dry-run" → subcmd="run", path="./repo", flags=["--model", "opus", "--dry-run"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"./repo", "--model", "opus", "--dry-run"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--model", "opus", "--dry-run"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_Empty(t *testing.T) {
	// No args → subcmd="run", path=""
	subcmd, repoPath, flags, err := extractSubcommand([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "" {
		t.Errorf("repoPath = %q, want empty", repoPath)
	}
	if len(flags) != 0 {
		t.Errorf("flags = %v, want empty", flags)
	}
}
