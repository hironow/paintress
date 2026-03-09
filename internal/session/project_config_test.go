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

func TestUpdateProjectConfig_SetCycle(t *testing.T) {
	// given
	dir := t.TempDir()
	cfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{Team: "MY", Project: "Test"},
	}
	session.SaveProjectConfig(dir, cfg)

	// when
	err := session.UpdateProjectConfig(dir, "tracker.cycle", "2026-Q1")

	// then
	if err != nil {
		t.Fatalf("UpdateProjectConfig: %v", err)
	}
	loaded, _ := session.LoadProjectConfig(dir)
	if loaded.Tracker.Cycle != "2026-Q1" {
		t.Errorf("expected cycle '2026-Q1', got %q", loaded.Tracker.Cycle)
	}
	if loaded.Tracker.Team != "MY" {
		t.Errorf("team should be preserved, got %q", loaded.Tracker.Team)
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

func TestUpdateProjectConfig_RuntimeKeys(t *testing.T) {
	tests := []struct {
		key   string
		value string
		check func(t *testing.T, cfg *domain.ProjectConfig)
	}{
		{"model", "sonnet", func(t *testing.T, cfg *domain.ProjectConfig) {
			t.Helper()
			if cfg.Model != "sonnet" {
				t.Errorf("Model = %q, want sonnet", cfg.Model)
			}
		}},
		{"workers", "3", func(t *testing.T, cfg *domain.ProjectConfig) {
			t.Helper()
			if cfg.Workers != 3 {
				t.Errorf("Workers = %d, want 3", cfg.Workers)
			}
		}},
		{"max_expeditions", "100", func(t *testing.T, cfg *domain.ProjectConfig) {
			t.Helper()
			if cfg.MaxExpeditions != 100 {
				t.Errorf("MaxExpeditions = %d, want 100", cfg.MaxExpeditions)
			}
		}},
		{"timeout_sec", "600", func(t *testing.T, cfg *domain.ProjectConfig) {
			t.Helper()
			if cfg.TimeoutSec != 600 {
				t.Errorf("TimeoutSec = %d, want 600", cfg.TimeoutSec)
			}
		}},
		{"base_branch", "develop", func(t *testing.T, cfg *domain.ProjectConfig) {
			t.Helper()
			if cfg.BaseBranch != "develop" {
				t.Errorf("BaseBranch = %q, want develop", cfg.BaseBranch)
			}
		}},
		{"auto_approve", "true", func(t *testing.T, cfg *domain.ProjectConfig) {
			t.Helper()
			if !cfg.AutoApprove {
				t.Error("AutoApprove should be true")
			}
		}},
		{"no_dev", "true", func(t *testing.T, cfg *domain.ProjectConfig) {
			t.Helper()
			if !cfg.NoDev {
				t.Error("NoDev should be true")
			}
		}},
		{"max_retries", "5", func(t *testing.T, cfg *domain.ProjectConfig) {
			t.Helper()
			if cfg.MaxRetries != 5 {
				t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			// given
			dir := t.TempDir()
			session.SaveProjectConfig(dir, &domain.ProjectConfig{Lang: "ja"})

			// when
			err := session.UpdateProjectConfig(dir, tt.key, tt.value)

			// then
			if err != nil {
				t.Fatalf("UpdateProjectConfig(%s, %s): %v", tt.key, tt.value, err)
			}
			loaded, _ := session.LoadProjectConfig(dir)
			tt.check(t, loaded)
		})
	}
}

func TestUpdateProjectConfig_InvalidWorkers(t *testing.T) {
	// given
	dir := t.TempDir()
	session.SaveProjectConfig(dir, &domain.ProjectConfig{Lang: "ja"})

	// when
	err := session.UpdateProjectConfig(dir, "workers", "-1")

	// then
	if err == nil {
		t.Error("expected error for negative workers")
	}
}

func TestProjectConfig_SaveLoadRoundTrip_AllFields(t *testing.T) {
	// given: DefaultProjectConfig saved to disk
	dir := t.TempDir()
	original := domain.DefaultProjectConfig()
	if err := session.SaveProjectConfig(dir, &original); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	// when: LoadProjectConfig from that directory
	loaded, err := session.LoadProjectConfig(dir)

	// then: no error
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}

	// verify key fields survive round-trip
	if loaded.Lang != "ja" {
		t.Errorf("Lang: expected 'ja', got %q", loaded.Lang)
	}
	if loaded.Model != "opus" {
		t.Errorf("Model: expected 'opus', got %q", loaded.Model)
	}
	if loaded.Workers != 1 {
		t.Errorf("Workers: expected 1, got %d", loaded.Workers)
	}
	if loaded.ClaudeCmd != "claude" {
		t.Errorf("ClaudeCmd: expected 'claude', got %q", loaded.ClaudeCmd)
	}
	if loaded.MaxExpeditions != 50 {
		t.Errorf("MaxExpeditions: expected 50, got %d", loaded.MaxExpeditions)
	}
	if loaded.TimeoutSec != 1980 {
		t.Errorf("TimeoutSec: expected 1980, got %d", loaded.TimeoutSec)
	}
	if loaded.BaseBranch != "main" {
		t.Errorf("BaseBranch: expected 'main', got %q", loaded.BaseBranch)
	}
	if loaded.DevCmd != "npm run dev" {
		t.Errorf("DevCmd: expected 'npm run dev', got %q", loaded.DevCmd)
	}
	if loaded.DevURL != "http://localhost:3000" {
		t.Errorf("DevURL: expected 'http://localhost:3000', got %q", loaded.DevURL)
	}
	if loaded.MaxRetries != 3 {
		t.Errorf("MaxRetries: expected 3, got %d", loaded.MaxRetries)
	}

	// verify Computed is zero-value after round-trip of defaults
	if loaded.Computed != (domain.ComputedConfig{}) {
		t.Errorf("Computed: expected zero-value, got %+v", loaded.Computed)
	}
}

func TestLoadProjectConfig_FileNotFound_AppliesRuntimeDefaults(t *testing.T) {
	// given — no config file
	dir := t.TempDir()

	// when
	cfg, err := session.LoadProjectConfig(dir)

	// then — runtime fields should have defaults from DefaultProjectConfig
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Model != "opus" {
		t.Errorf("Model = %q, want opus", cfg.Model)
	}
	if cfg.Workers != 1 {
		t.Errorf("Workers = %d, want 1", cfg.Workers)
	}
	if cfg.MaxExpeditions != 50 {
		t.Errorf("MaxExpeditions = %d, want 50", cfg.MaxExpeditions)
	}
	if cfg.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want main", cfg.BaseBranch)
	}
}

func TestUpdateProjectConfig_RejectComputedKey(t *testing.T) {
	// given
	dir := t.TempDir()
	session.SaveProjectConfig(dir, &domain.ProjectConfig{Lang: "ja"})

	// when — attempt to set a computed-namespace key
	err := session.UpdateProjectConfig(dir, "computed", "anything")

	// then — should be rejected (unknown key)
	if err == nil {
		t.Error("expected error when setting computed key")
	}
}
