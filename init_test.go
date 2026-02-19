package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_WritesConfig(t *testing.T) {
	dir := t.TempDir()

	// Simulate user input: team key "ENG", project "backend"
	input := "ENG\nbackend\n"
	reader := strings.NewReader(input)

	if err := runInitWithReader(dir, reader); err != nil {
		t.Fatalf("runInitWithReader: %v", err)
	}

	// Verify config file was created
	cfgPath := ProjectConfigPath(dir)
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Verify content
	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if cfg.Linear.Team != "ENG" {
		t.Errorf("Team = %q, want %q", cfg.Linear.Team, "ENG")
	}
	if cfg.Linear.Project != "backend" {
		t.Errorf("Project = %q, want %q", cfg.Linear.Project, "backend")
	}
}

func TestRunInit_SkipOptionalProject(t *testing.T) {
	dir := t.TempDir()

	// Simulate user input: team key "MY", empty project (press Enter)
	input := "MY\n\n"
	reader := strings.NewReader(input)

	if err := runInitWithReader(dir, reader); err != nil {
		t.Fatalf("runInitWithReader: %v", err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if cfg.Linear.Team != "MY" {
		t.Errorf("Team = %q, want %q", cfg.Linear.Team, "MY")
	}
	if cfg.Linear.Project != "" {
		t.Errorf("Project = %q, want empty", cfg.Linear.Project)
	}
}

func TestRunInit_CreatesExpeditionDir(t *testing.T) {
	dir := t.TempDir()

	input := "MY\n\n"
	reader := strings.NewReader(input)

	if err := runInitWithReader(dir, reader); err != nil {
		t.Fatalf("runInitWithReader: %v", err)
	}

	// .expedition directory should exist
	expeditionDir := filepath.Join(dir, ".expedition")
	info, err := os.Stat(expeditionDir)
	if err != nil {
		t.Fatalf(".expedition dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".expedition should be a directory")
	}
}
