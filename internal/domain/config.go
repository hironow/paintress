package domain

import (
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

// ProjectConfig holds project-scoped configuration stored in .expedition/config.yaml.
type ProjectConfig struct {
	Tracker IssueTrackerConfig `yaml:"tracker"`
}

// IssueTrackerConfig holds issue tracker integration settings.
type IssueTrackerConfig struct {
	Team    string `yaml:"team"`
	Project string `yaml:"project,omitempty"`
}

// TrackerTeam returns the issue tracker team key (e.g. "MY").
func (c ProjectConfig) TrackerTeam() string { return c.Tracker.Team }

// TrackerProject returns the issue tracker project name.
func (c ProjectConfig) TrackerProject() string { return c.Tracker.Project }

// HasTrackerTeam reports whether a tracker team is configured.
func (c ProjectConfig) HasTrackerTeam() bool { return c.Tracker.Team != "" }

// ProjectConfigPath returns the path to the project config file.
func ProjectConfigPath(continent string) string {
	return filepath.Join(continent, StateDir, "config.yaml")
}
