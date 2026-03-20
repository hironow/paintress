package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"gopkg.in/yaml.v3"
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

func TestValidLang(t *testing.T) {
	// given/when/then
	if !domain.ValidLang("ja") {
		t.Error("ja should be valid")
	}
	if !domain.ValidLang("en") {
		t.Error("en should be valid")
	}
	if domain.ValidLang("fr") {
		t.Error("fr should be invalid")
	}
	if domain.ValidLang("") {
		t.Error("empty should be invalid")
	}
}

func TestValidateProjectConfig_Valid(t *testing.T) {
	// given
	cfg := domain.DefaultProjectConfig()

	// when
	errs := domain.ValidateProjectConfig(cfg)

	// then
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidateProjectConfig_InvalidLang(t *testing.T) {
	// given
	cfg := domain.DefaultProjectConfig()
	cfg.Lang = "fr"

	// when
	errs := domain.ValidateProjectConfig(cfg)

	// then
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidateProjectConfig_EmptyLangIsValid(t *testing.T) {
	// given — empty lang is acceptable (defaults will fill it)
	cfg := domain.DefaultProjectConfig()
	cfg.Lang = ""

	// when
	errs := domain.ValidateProjectConfig(cfg)

	// then
	if len(errs) != 0 {
		t.Errorf("expected no errors for empty lang, got %v", errs)
	}
}

func TestDefaultProjectConfig_MatchesDefaultConfig(t *testing.T) {
	// given
	pc := domain.DefaultProjectConfig()
	rc := domain.DefaultConfig()

	// then — shared fields must match
	if pc.Model != rc.Model {
		t.Errorf("Model mismatch: ProjectConfig=%q, Config=%q", pc.Model, rc.Model)
	}
	if pc.Workers != rc.Workers {
		t.Errorf("Workers mismatch: ProjectConfig=%d, Config=%d", pc.Workers, rc.Workers)
	}
	if pc.MaxExpeditions != rc.MaxExpeditions {
		t.Errorf("MaxExpeditions mismatch: ProjectConfig=%d, Config=%d", pc.MaxExpeditions, rc.MaxExpeditions)
	}
	if pc.TimeoutSec != rc.TimeoutSec {
		t.Errorf("TimeoutSec mismatch: ProjectConfig=%d, Config=%d", pc.TimeoutSec, rc.TimeoutSec)
	}
	if pc.BaseBranch != rc.BaseBranch {
		t.Errorf("BaseBranch mismatch: ProjectConfig=%q, Config=%q", pc.BaseBranch, rc.BaseBranch)
	}
	if pc.MaxRetries != rc.MaxRetries {
		t.Errorf("MaxRetries mismatch: ProjectConfig=%d, Config=%d", pc.MaxRetries, rc.MaxRetries)
	}
	if pc.WaitTimeout != rc.WaitTimeout {
		t.Errorf("WaitTimeout mismatch: ProjectConfig=%v, Config=%v", pc.WaitTimeout, rc.WaitTimeout)
	}
}

func TestValidateProjectConfig_NegativeFields(t *testing.T) {
	tests := []struct {
		name string
		cfg  domain.ProjectConfig
	}{
		{"negative max_expeditions", func() domain.ProjectConfig { c := domain.DefaultProjectConfig(); c.MaxExpeditions = -1; return c }()},
		{"negative timeout_sec", func() domain.ProjectConfig { c := domain.DefaultProjectConfig(); c.TimeoutSec = -1; return c }()},
		{"negative workers", func() domain.ProjectConfig { c := domain.DefaultProjectConfig(); c.Workers = -1; return c }()},
		{"negative max_retries", func() domain.ProjectConfig { c := domain.DefaultProjectConfig(); c.MaxRetries = -1; return c }()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := domain.ValidateProjectConfig(tt.cfg)
			if len(errs) == 0 {
				t.Error("expected validation error")
			}
		})
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

func TestValidateProjectConfig_EmptyDevCmd_NoDevFalse(t *testing.T) {
	// given — dev_cmd empty with no_dev=false should be invalid
	cfg := domain.DefaultProjectConfig()
	cfg.DevCmd = ""
	cfg.NoDev = false

	// when
	errs := domain.ValidateProjectConfig(cfg)

	// then
	if len(errs) == 0 {
		t.Fatal("expected validation error for empty dev_cmd when no_dev is false")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "dev_cmd") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected dev_cmd error, got: %v", errs)
	}
}

func TestValidateProjectConfig_EmptyDevCmd_NoDevTrue_IsValid(t *testing.T) {
	// given — dev_cmd empty is OK when no_dev=true
	cfg := domain.DefaultProjectConfig()
	cfg.DevCmd = ""
	cfg.NoDev = true

	// when
	errs := domain.ValidateProjectConfig(cfg)

	// then
	if len(errs) != 0 {
		t.Errorf("expected no errors when no_dev=true, got %v", errs)
	}
}

func TestProjectConfig_ComputedConfig_EmptyByDefault(t *testing.T) {
	// given/when
	cfg := domain.DefaultProjectConfig()

	// then
	if cfg.Computed != (domain.ComputedConfig{}) {
		t.Error("Computed should be zero-value by default")
	}
}

func TestDefaultConfig_WaitTimeout(t *testing.T) {
	// when
	cfg := domain.DefaultConfig()

	// then
	if cfg.WaitTimeout != domain.DefaultWaitTimeout {
		t.Errorf("expected WaitTimeout=%v, got %v", domain.DefaultWaitTimeout, cfg.WaitTimeout)
	}
}

func TestDefaultWaitTimeout_Is30Minutes(t *testing.T) {
	// then
	if domain.DefaultWaitTimeout != 30*time.Minute {
		t.Errorf("expected 30m, got %v", domain.DefaultWaitTimeout)
	}
}

func TestParseModelConfig_Valid(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantPrimary  string
		wantReserves []string
	}{
		{"single model", "opus", "opus", nil},
		{"two models", "opus,sonnet", "opus", []string{"sonnet"}},
		{"three models", "opus,sonnet,haiku", "opus", []string{"sonnet", "haiku"}},
		{"whitespace trimmed", " opus , sonnet ", "opus", []string{"sonnet"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary, reserves, err := domain.ParseModelConfig(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if primary != tt.wantPrimary {
				t.Errorf("primary = %q, want %q", primary, tt.wantPrimary)
			}
			if len(reserves) != len(tt.wantReserves) {
				t.Fatalf("reserves = %v, want %v", reserves, tt.wantReserves)
			}
			for i, r := range reserves {
				if r != tt.wantReserves[i] {
					t.Errorf("reserves[%d] = %q, want %q", i, r, tt.wantReserves[i])
				}
			}
		})
	}
}

func TestParseModelConfig_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"empty segment", "opus,,haiku"},
		{"duplicate models", "opus,opus"},
		{"leading comma", ",sonnet"},
		{"trailing whitespace-only", "opus, "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := domain.ParseModelConfig(tt.input)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestValidateProjectConfig_ModelFormat(t *testing.T) {
	// given
	cfg := domain.DefaultProjectConfig()
	cfg.Model = "opus,,haiku"

	// when
	errs := domain.ValidateProjectConfig(cfg)

	// then
	if len(errs) == 0 {
		t.Fatal("expected validation error for malformed model string")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "model") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected model-related error, got: %v", errs)
	}
}

func TestValidateProjectConfig_EmptyBaseBranch(t *testing.T) {
	// given
	cfg := domain.DefaultProjectConfig()
	cfg.BaseBranch = ""

	// when
	errs := domain.ValidateProjectConfig(cfg)

	// then
	if len(errs) == 0 {
		t.Fatal("expected validation error for empty base_branch")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "base_branch") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected base_branch error, got: %v", errs)
	}
}

func TestProjectConfig_YAMLRoundTrip_NoComputedKey(t *testing.T) {
	// given
	cfg := domain.DefaultProjectConfig()

	// when: marshal
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	yamlStr := string(data)

	// then: no "computed" key in output
	if strings.Contains(yamlStr, "computed") {
		t.Errorf("YAML should not contain 'computed' key, got:\n%s", yamlStr)
	}

	// when: unmarshal back
	var restored domain.ProjectConfig
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// then: Lang preserved
	if restored.Lang != cfg.Lang {
		t.Errorf("Lang: expected %q, got %q", cfg.Lang, restored.Lang)
	}
}
