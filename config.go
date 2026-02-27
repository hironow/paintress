package paintress

import (
	"embed"
	"path/filepath"
)

//go:embed templates/skills/*/SKILL.md
var SkillsFS embed.FS

// DefaultClaudeCmd is the default CLI command name for Claude Code.
const DefaultClaudeCmd = "claude"

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
}

// ProjectConfig holds project-scoped configuration stored in .expedition/config.yaml.
type ProjectConfig struct {
	Linear LinearConfig `yaml:"linear"`
}

// LinearConfig holds Linear integration settings.
type LinearConfig struct {
	Team    string `yaml:"team"`
	Project string `yaml:"project,omitempty"`
}

// ProjectConfigPath returns the path to the project config file.
func ProjectConfigPath(continent string) string {
	return filepath.Join(continent, ".expedition", "config.yaml")
}
