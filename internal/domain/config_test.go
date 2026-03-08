package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestDefaultConfig_AllFields(t *testing.T) {
	// given/when
	cfg := domain.DefaultConfig()

	// then: runtime defaults
	if cfg.MaxExpeditions != 50 {
		t.Errorf("MaxExpeditions: expected 50, got %d", cfg.MaxExpeditions)
	}
	if cfg.TimeoutSec != 1980 {
		t.Errorf("TimeoutSec: expected 1980, got %d", cfg.TimeoutSec)
	}
	if cfg.Model != "opus" {
		t.Errorf("Model: expected 'opus', got %q", cfg.Model)
	}
	if cfg.BaseBranch != "main" {
		t.Errorf("BaseBranch: expected 'main', got %q", cfg.BaseBranch)
	}
	if cfg.ClaudeCmd != "claude" {
		t.Errorf("ClaudeCmd: expected 'claude', got %q", cfg.ClaudeCmd)
	}
	if cfg.DevCmd != "npm run dev" {
		t.Errorf("DevCmd: expected 'npm run dev', got %q", cfg.DevCmd)
	}
	if cfg.DevURL != "http://localhost:3000" {
		t.Errorf("DevURL: expected 'http://localhost:3000', got %q", cfg.DevURL)
	}
	if cfg.Workers != 1 {
		t.Errorf("Workers: expected 1, got %d", cfg.Workers)
	}
	if cfg.OutputFormat != "text" {
		t.Errorf("OutputFormat: expected 'text', got %q", cfg.OutputFormat)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries: expected 3, got %d", cfg.MaxRetries)
	}

	// then: zero-value fields (set at runtime, not in defaults)
	if cfg.Continent != "" {
		t.Errorf("Continent: expected empty, got %q", cfg.Continent)
	}
	if cfg.DevDir != "" {
		t.Errorf("DevDir: expected empty, got %q", cfg.DevDir)
	}
	if cfg.ReviewCmd != "" {
		t.Errorf("ReviewCmd: expected empty, got %q", cfg.ReviewCmd)
	}
	if cfg.SetupCmd != "" {
		t.Errorf("SetupCmd: expected empty, got %q", cfg.SetupCmd)
	}
	if cfg.NoDev {
		t.Error("NoDev: expected false")
	}
	if cfg.DryRun {
		t.Error("DryRun: expected false")
	}
	if cfg.NotifyCmd != "" {
		t.Errorf("NotifyCmd: expected empty, got %q", cfg.NotifyCmd)
	}
	if cfg.ApproveCmd != "" {
		t.Errorf("ApproveCmd: expected empty, got %q", cfg.ApproveCmd)
	}
	if cfg.AutoApprove {
		t.Error("AutoApprove: expected false")
	}
}

func TestProjectConfig_TrackerMethods(t *testing.T) {
	// given
	empty := domain.ProjectConfig{}
	full := domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{
			Team:    "MY",
			Project: "Test",
		},
	}

	// then: empty
	if empty.HasTrackerTeam() {
		t.Error("empty: HasTrackerTeam should be false")
	}
	if empty.TrackerTeam() != "" {
		t.Errorf("empty: TrackerTeam = %q", empty.TrackerTeam())
	}
	if empty.TrackerProject() != "" {
		t.Errorf("empty: TrackerProject = %q", empty.TrackerProject())
	}

	// then: full
	if !full.HasTrackerTeam() {
		t.Error("full: HasTrackerTeam should be true")
	}
	if full.TrackerTeam() != "MY" {
		t.Errorf("full: TrackerTeam = %q", full.TrackerTeam())
	}
	if full.TrackerProject() != "Test" {
		t.Errorf("full: TrackerProject = %q", full.TrackerProject())
	}
}
