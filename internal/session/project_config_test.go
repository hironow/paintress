package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestProjectConfigPath(t *testing.T) {
	got := domain.ProjectConfigPath("/tmp/repo")
	want := "/tmp/repo/.expedition/config.yaml"
	if got != want {
		t.Errorf("ProjectConfigPath = %q, want %q", got, want)
	}
}

func TestSaveAndLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	cfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{
			Team:    "MY",
			Project: "paintress",
		},
	}

	if err := session.SaveProjectConfig(dir, cfg); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	loaded, err := session.LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}

	if loaded.Tracker.Team != "MY" {
		t.Errorf("Team = %q, want %q", loaded.Tracker.Team, "MY")
	}
	if loaded.Tracker.Project != "paintress" {
		t.Errorf("Project = %q, want %q", loaded.Tracker.Project, "paintress")
	}
}

func TestSaveProjectConfig_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	// .expedition/ does NOT exist — SaveProjectConfig should create it

	cfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{Team: "TEST"},
	}

	if err := session.SaveProjectConfig(dir, cfg); err != nil {
		t.Fatalf("SaveProjectConfig should create parent dir, got: %v", err)
	}

	// Verify file was written
	loaded, err := session.LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if loaded.Tracker.Team != "TEST" {
		t.Errorf("Team = %q, want %q", loaded.Tracker.Team, "TEST")
	}
}

func TestLoadProjectConfig_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	cfg, err := session.LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Tracker.Team != "" {
		t.Errorf("Team = %q, want empty", cfg.Tracker.Team)
	}
}
