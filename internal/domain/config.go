package domain

import (
	"fmt"
	"path/filepath"
)

// Config holds the runtime configuration for a Paintress session.
type Config struct {
	Continent      string
	MaxExpeditions int
	TimeoutSec     int
	Model          string // "opus" or "opus,sonnet,haiku" for reserve party
	BaseBranch     string
	ClaudeCmd      string // CLI command name for Claude Code (e.g. "claude", "cc-p")
	DevCmd         string
	DevDir         string // working directory for dev server (defaults to Continent)
	DevURL         string
	ReviewCmd      string // Code review command (e.g. "codex review --base main")
	Workers        int    // Number of worktrees in pool (0 = direct execution)
	SetupCmd       string // Command to run after worktree creation (e.g. "bun install")
	NoDev          bool   // Skip dev server startup entirely
	DryRun         bool
	OutputFormat   string // "text" (default) or "json"
	NotifyCmd      string // External notification command ({title}, {message} placeholders)
	ApproveCmd     string // External approval command ({message} placeholder, exit 0 = approve)
	AutoApprove    bool   // Skip approval gate for HIGH severity D-Mail
	MaxRetries     int    // Maximum retry attempts per unique issue set (default: 3)
}

// DefaultConfig returns a Config populated with sensible defaults.
// Values are sourced from DefaultProjectConfig() for consistency.
func DefaultConfig() Config {
	pc := DefaultProjectConfig()
	return Config{
		MaxExpeditions: pc.MaxExpeditions,
		TimeoutSec:     pc.TimeoutSec,
		Model:          pc.Model,
		BaseBranch:     pc.BaseBranch,
		ClaudeCmd:      pc.ClaudeCmd,
		DevCmd:         pc.DevCmd,
		DevURL:         pc.DevURL,
		Workers:        pc.Workers,
		OutputFormat:   "text",
		MaxRetries:     pc.MaxRetries,
	}
}

// ComputedConfig holds system-written fields. Empty for paintress today.
type ComputedConfig struct{}

// ProjectConfig holds project-scoped configuration stored in .expedition/config.yaml.
// Runtime-only fields (Continent, DryRun, OutputFormat) are NOT persisted here.
type ProjectConfig struct {
	Tracker        IssueTrackerConfig `yaml:"tracker"`
	Lang           string             `yaml:"lang,omitempty"`
	MaxExpeditions int                `yaml:"max_expeditions,omitempty"`
	TimeoutSec     int                `yaml:"timeout_sec,omitempty"`
	Model          string             `yaml:"model,omitempty"`
	BaseBranch     string             `yaml:"base_branch,omitempty"`
	ClaudeCmd      string             `yaml:"claude_cmd,omitempty"`
	DevCmd         string             `yaml:"dev_cmd,omitempty"`
	DevDir         string             `yaml:"dev_dir,omitempty"`
	DevURL         string             `yaml:"dev_url,omitempty"`
	ReviewCmd      string             `yaml:"review_cmd,omitempty"`
	Workers        int                `yaml:"workers,omitempty"`
	SetupCmd       string             `yaml:"setup_cmd,omitempty"`
	NoDev          bool               `yaml:"no_dev,omitempty"`
	NotifyCmd      string             `yaml:"notify_cmd,omitempty"`
	ApproveCmd     string             `yaml:"approve_cmd,omitempty"`
	AutoApprove    bool               `yaml:"auto_approve,omitempty"`
	MaxRetries     int                `yaml:"max_retries,omitempty"`
	Computed       ComputedConfig     `yaml:"computed,omitempty"`
}

// DefaultProjectConfig returns a ProjectConfig populated with sensible defaults.
func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		Lang:           "ja",
		MaxExpeditions: 50,
		TimeoutSec:     1980, // 33 minutes
		Model:          "opus",
		BaseBranch:     "main",
		ClaudeCmd:      "claude",
		DevCmd:         "npm run dev",
		DevURL:         "http://localhost:3000",
		Workers:        1,
		MaxRetries:     3,
	}
}

// IssueTrackerConfig holds issue tracker integration settings.
type IssueTrackerConfig struct {
	Team    string `yaml:"team"`
	Project string `yaml:"project"`
	Cycle   string `yaml:"cycle"`
}

// TrackerTeam returns the issue tracker team key (e.g. "MY").
func (c ProjectConfig) TrackerTeam() string { return c.Tracker.Team }

// TrackerProject returns the issue tracker project name.
func (c ProjectConfig) TrackerProject() string { return c.Tracker.Project }

// HasTrackerTeam reports whether a tracker team is configured.
func (c ProjectConfig) HasTrackerTeam() bool { return c.Tracker.Team != "" }

// ValidLang reports whether lang is a supported language code.
func ValidLang(lang string) bool {
	return lang == "ja" || lang == "en"
}

// ValidateProjectConfig checks the project config for consistency and returns errors.
// An empty slice means the config is valid.
func ValidateProjectConfig(cfg ProjectConfig) []string {
	var errs []string
	if cfg.Lang != "" && !ValidLang(cfg.Lang) {
		errs = append(errs, fmt.Sprintf("lang must be \"ja\" or \"en\" (got %q)", cfg.Lang))
	}
	if cfg.MaxExpeditions < 0 {
		errs = append(errs, fmt.Sprintf("max_expeditions must be non-negative (got %d)", cfg.MaxExpeditions))
	}
	if cfg.TimeoutSec < 0 {
		errs = append(errs, fmt.Sprintf("timeout_sec must be non-negative (got %d)", cfg.TimeoutSec))
	}
	if cfg.Workers < 0 {
		errs = append(errs, fmt.Sprintf("workers must be non-negative (got %d)", cfg.Workers))
	}
	if cfg.MaxRetries < 0 {
		errs = append(errs, fmt.Sprintf("max_retries must be non-negative (got %d)", cfg.MaxRetries))
	}
	return errs
}

// ProjectConfigPath returns the path to the project config file.
func ProjectConfigPath(continent string) string {
	return filepath.Join(continent, StateDir, "config.yaml")
}
