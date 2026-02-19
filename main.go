package paintress

import (
	"os"
	"path/filepath"
)

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
}

// ValidateContinent ensures the .expedition directory structure exists.
func ValidateContinent(continent string) error {
	journalDir := filepath.Join(continent, ".expedition", "journal")
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return err
	}

	// Ensure .run/ directory exists for ephemeral files (flag.md, logs/, worktrees/)
	runDir := filepath.Join(continent, ".expedition", ".run")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return err
	}

	// Ensure .run/ is gitignored
	gitignore := filepath.Join(continent, ".expedition", ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		os.WriteFile(gitignore, []byte(".run/\n"), 0644)
	}
	return nil
}
