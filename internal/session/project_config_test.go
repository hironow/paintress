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

func TestUpdateProjectConfig_SetTeam(t *testing.T) {
	// given
	dir := t.TempDir()
	cfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{Team: "OLD", Project: "Test"},
	}
	session.SaveProjectConfig(dir, cfg)

	// when
	err := session.UpdateProjectConfig(dir, "tracker.team", "NEW")

	// then
	if err != nil {
		t.Fatalf("UpdateProjectConfig: %v", err)
	}
	loaded, _ := session.LoadProjectConfig(dir)
	if loaded.Tracker.Team != "NEW" {
		t.Errorf("expected team 'NEW', got %q", loaded.Tracker.Team)
	}
	if loaded.Tracker.Project != "Test" {
		t.Errorf("project should be preserved, got %q", loaded.Tracker.Project)
	}
}

func TestUpdateProjectConfig_SetProject(t *testing.T) {
	// given
	dir := t.TempDir()
	cfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{Team: "MY"},
	}
	session.SaveProjectConfig(dir, cfg)

	// when
	err := session.UpdateProjectConfig(dir, "tracker.project", "NewProject")

	// then
	if err != nil {
		t.Fatalf("UpdateProjectConfig: %v", err)
	}
	loaded, _ := session.LoadProjectConfig(dir)
	if loaded.Tracker.Project != "NewProject" {
		t.Errorf("expected project 'NewProject', got %q", loaded.Tracker.Project)
	}
}

func TestUpdateProjectConfig_InvalidKey(t *testing.T) {
	// given
	dir := t.TempDir()
	cfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{Team: "MY"},
	}
	session.SaveProjectConfig(dir, cfg)

	// when
	err := session.UpdateProjectConfig(dir, "bad.key", "value")

	// then
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestLoadProjectConfig_FileNotFound_ReturnsDefaults(t *testing.T) {
	// given
	dir := t.TempDir()

	// when
	cfg, err := session.LoadProjectConfig(dir)

	// then — should return DefaultProjectConfig, not zero-value
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Tracker.Team != "" {
		t.Errorf("Team = %q, want empty", cfg.Tracker.Team)
	}
	if cfg.Lang != "ja" {
		t.Errorf("expected default lang=ja, got %q", cfg.Lang)
	}
}

func TestLoadProjectConfig_AppliesDefaults(t *testing.T) {
	// given — config file with only tracker, no lang
	dir := t.TempDir()
	cfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{Team: "MY"},
	}
	session.SaveProjectConfig(dir, cfg)

	// when
	loaded, err := session.LoadProjectConfig(dir)

	// then — lang should be filled from defaults
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if loaded.Tracker.Team != "MY" {
		t.Errorf("Team = %q, want MY", loaded.Tracker.Team)
	}
	if loaded.Lang != "ja" {
		t.Errorf("expected default lang=ja applied, got %q", loaded.Lang)
	}
}

func TestUpdateProjectConfig_SetLang(t *testing.T) {
	// given
	dir := t.TempDir()
	cfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{Team: "MY"},
		Lang:    "ja",
	}
	session.SaveProjectConfig(dir, cfg)

	// when
	err := session.UpdateProjectConfig(dir, "lang", "en")

	// then
	if err != nil {
		t.Fatalf("UpdateProjectConfig: %v", err)
	}
	loaded, _ := session.LoadProjectConfig(dir)
	if loaded.Lang != "en" {
		t.Errorf("expected lang=en, got %q", loaded.Lang)
	}
	if loaded.Tracker.Team != "MY" {
		t.Errorf("team should be preserved, got %q", loaded.Tracker.Team)
	}
}

func TestUpdateProjectConfig_InvalidLang(t *testing.T) {
	// given
	dir := t.TempDir()
	cfg := &domain.ProjectConfig{Lang: "ja"}
	session.SaveProjectConfig(dir, cfg)

	// when
	err := session.UpdateProjectConfig(dir, "lang", "fr")

	// then
	if err == nil {
		t.Error("expected error for invalid lang")
	}
}
