package domain

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Default values for Config fields. Used by DefaultProjectConfig and post-load
// validation to avoid hardcoded strings throughout the codebase.
const (
	DefaultClaudeCmd  = "claude"
	DefaultModel      = "opus"
	DefaultTimeoutSec = 1980
)

// DefaultIdleTimeout is the default D-Mail waiting phase timeout.
const DefaultIdleTimeout = 30 * time.Minute

// ApproverConfig describes how approval behavior is configured.
// Implemented by Config. Used by session.BuildApprover.
type ApproverConfig interface {
	IsAutoApprove() bool
	ApproveCmdString() string
}

// Config holds the runtime configuration for a Paintress session.
// Config implements ApproverConfig.
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
	OutputFormat   string        // "text" (default) or "json"
	NotifyCmd      string        // External notification command ({title}, {message} placeholders)
	ApproveCmd     string        // External approval command ({message} placeholder, exit 0 = approve)
	AutoApprove    bool          // Skip approval gate for HIGH severity D-Mail
	MaxRetries     int           // Maximum retry attempts per unique issue set (default: 3)
	IdleTimeout    time.Duration `yaml:"idle_timeout,omitempty"` // D-Mail waiting phase timeout (0 = 24h safety cap, <0 = disable waiting)
}

// IsAutoApprove reports whether the config is set to auto-approve.
func (c Config) IsAutoApprove() bool { return c.AutoApprove }

// ApproveCmdString returns the approval command string.
func (c Config) ApproveCmdString() string { return c.ApproveCmd }

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
		IdleTimeout:    pc.IdleTimeout,
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
	IdleTimeout    time.Duration      `yaml:"idle_timeout,omitempty"`
	Computed       ComputedConfig     `yaml:"computed,omitempty"`
}

// DefaultProjectConfig returns a ProjectConfig populated with sensible defaults.
func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		Lang:           "ja",
		MaxExpeditions: 50,
		TimeoutSec:     DefaultTimeoutSec, // 33 minutes
		Model:          DefaultModel,
		BaseBranch:     "main",
		ClaudeCmd:      DefaultClaudeCmd,
		DevCmd:         "npm run dev",
		DevURL:         "http://localhost:3000",
		Workers:        1,
		MaxRetries:     3,
		IdleTimeout:    DefaultIdleTimeout,
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

// ParseModelConfig parses a model config string (e.g. "opus,sonnet,haiku")
// into a primary model and optional reserve models. Returns an error for
// empty strings, empty segments, whitespace-only segments, or duplicates.
func ParseModelConfig(model string) (string, []string, error) {
	if strings.TrimSpace(model) == "" {
		return "", nil, fmt.Errorf("model config must not be empty")
	}
	parts := strings.Split(model, ",")
	seen := make(map[string]bool, len(parts))
	var primary string
	var reserves []string
	for i, p := range parts {
		m := strings.TrimSpace(p)
		if m == "" {
			return "", nil, fmt.Errorf("model config has empty segment at position %d", i)
		}
		if seen[m] {
			return "", nil, fmt.Errorf("model config has duplicate: %q", m)
		}
		seen[m] = true
		if i == 0 {
			primary = m
		} else {
			reserves = append(reserves, m)
		}
	}
	return primary, reserves, nil
}

// ValidLang reports whether lang is a supported language code.
func ValidLang(lang string) bool {
	return lang == "ja" || lang == "en"
}

// ValidateProjectConfig checks the project config for consistency and returns errors.
// An empty slice means the config is valid.
func ValidateProjectConfig(cfg ProjectConfig) []string {
	var errs []string
	if cfg.ClaudeCmd == "" {
		errs = append(errs, "claude_cmd must not be empty")
	}
	if cfg.BaseBranch == "" {
		errs = append(errs, "base_branch must not be empty")
	}
	if cfg.Model == "" {
		errs = append(errs, "model must not be empty")
	} else if _, _, err := ParseModelConfig(cfg.Model); err != nil {
		errs = append(errs, fmt.Sprintf("model config invalid: %v", err))
	}
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
	if !cfg.NoDev && cfg.DevCmd == "" {
		errs = append(errs, "dev_cmd must not be empty when no_dev is false")
	}
	return errs
}

// ProjectConfigPath returns the path to the project config file.
func ProjectConfigPath(continent string) string {
	return filepath.Join(continent, StateDir, "config.yaml")
}
