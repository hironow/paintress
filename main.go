package paintress

import (
	"os"
	"path/filepath"
	"strings"
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
	OutputFormat   string // "text" (default) or "json"
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

	// Ensure d-mail directories exist (inbox, outbox, archive)
	for _, sub := range []string{"inbox", "outbox", "archive"} {
		d := filepath.Join(continent, ".expedition", sub)
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	// Ensure Agent Skills SKILL.md files exist for phonewave discovery
	skills := []struct {
		dir     string
		content string
	}{
		{
			dir:     "dmail-sendable",
			content: "---\nname: dmail-sendable\ndescription: Produces D-Mail report messages to outbox/ after expedition completion.\nproduces:\n  - report\n---\n\nD-Mail send capability for paintress.\n",
		},
		{
			dir:     "dmail-readable",
			content: "---\nname: dmail-readable\ndescription: Consumes D-Mail specifications and feedback from inbox/.\nconsumes:\n  - specification\n  - feedback\n---\n\nD-Mail receive capability for paintress.\n",
		},
	}
	for _, s := range skills {
		skillDir := filepath.Join(continent, ".expedition", "skills", s.dir)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		skillFile := filepath.Join(skillDir, "SKILL.md")
		if _, err := os.Stat(skillFile); os.IsNotExist(err) {
			if err := os.WriteFile(skillFile, []byte(s.content), 0644); err != nil {
				return err
			}
		}
	}

	// Ensure .run/, inbox/, outbox/ are gitignored (handles both fresh and upgrade scenarios)
	gitignore := filepath.Join(continent, ".expedition", ".gitignore")
	content, err := os.ReadFile(gitignore)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// File doesn't exist — create with all entries
		if err := os.WriteFile(gitignore, []byte(".run/\ninbox/\noutbox/\n"), 0644); err != nil {
			return err
		}
	} else {
		// Existing file — append any missing entries
		entries := []string{".run/", "inbox/", "outbox/"}
		var missing []string
		for _, entry := range entries {
			if !strings.Contains(string(content), entry) {
				missing = append(missing, entry)
			}
		}
		if len(missing) > 0 {
			f, err := os.OpenFile(gitignore, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			// Ensure first entry starts on its own line
			if len(content) > 0 && content[len(content)-1] != '\n' {
				if _, err := f.WriteString("\n"); err != nil {
					return err
				}
			}
			for _, entry := range missing {
				if _, err := f.WriteString(entry + "\n"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
