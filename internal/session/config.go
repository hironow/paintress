package session

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// expeditionGitignoreEntries lists paths that must be gitignored in .expedition/.
var expeditionGitignoreEntries = []string{
	".run/",
	"inbox/",
	"outbox/",
	".otel.env",
	"events/",
	".mcp.json",
	".claude/",
}

// ValidateContinent ensures the .expedition directory structure exists.
// Uses the shared EnsureStateDir helper for core directories, then adds
// paintress-specific journal/ dir and skill templates.
func ValidateContinent(continent string, logger domain.Logger) error {
	if logger == nil {
		logger = &domain.NopLogger{}
	}
	stateDir := filepath.Join(continent, domain.StateDir)

	// Core directories + mail dirs + journal
	if err := EnsureStateDir(stateDir, WithMailDirs(), WithExtraDirs("journal")); err != nil {
		return err
	}

	// Skill templates (idempotent sync from embedded FS)
	skillDirs := []string{"dmail-sendable", "dmail-readable"}
	for _, dir := range skillDirs {
		skillDir := filepath.Join(stateDir, "skills", dir)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		skillFile := filepath.Join(skillDir, "SKILL.md")
		content, err := fs.ReadFile(platform.SkillsFS, filepath.Join("templates", "skills", dir, "SKILL.md"))
		if err != nil {
			return fmt.Errorf("read embedded skill %s: %w", dir, err)
		}
		existing, readErr := os.ReadFile(skillFile)
		if readErr != nil || !bytesEqual(existing, content) {
			if readErr == nil {
				logger.Info("updated SKILL.md: %s (template changed)", dir)
			}
			if err := os.WriteFile(skillFile, content, 0644); err != nil {
				return err
			}
		}
	}

	// Gitignore (append-only)
	return EnsureGitignoreEntries(filepath.Join(stateDir, ".gitignore"), expeditionGitignoreEntries)
}

// bytesEqual compares two byte slices without importing bytes.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
