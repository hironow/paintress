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
// Returns an InitResult recording what was created or skipped.
func ValidateContinent(continent string, logger domain.Logger) (*InitResult, error) {
	if logger == nil {
		logger = &domain.NopLogger{}
	}
	stateDir := filepath.Join(continent, domain.StateDir)

	// Core directories + mail dirs + journal
	result, err := EnsureStateDir(stateDir, WithMailDirs(), WithExtraDirs("journal"))
	if err != nil {
		return result, err
	}

	// Skill templates (idempotent sync from embedded FS)
	skillDirs := []string{"dmail-sendable", "dmail-readable"}
	for _, dir := range skillDirs {
		skillDir := filepath.Join(stateDir, "skills", dir)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return result, err
		}
		skillFile := filepath.Join(skillDir, "SKILL.md")
		content, err := fs.ReadFile(platform.SkillsFS, filepath.Join("templates", "skills", dir, "SKILL.md"))
		if err != nil {
			return result, fmt.Errorf("read embedded skill %s: %w", dir, err)
		}
		existing, readErr := os.ReadFile(skillFile)
		if readErr != nil || !bytesEqual(existing, content) {
			if readErr == nil {
				logger.Info("updated SKILL.md: %s (template changed)", dir)
			}
			if err := os.WriteFile(skillFile, content, 0644); err != nil {
				return result, err
			}
			result.Add(domain.StateDir+"/skills/"+dir+"/", InitUpdated, "")
		} else {
			result.Add(domain.StateDir+"/skills/"+dir+"/", InitSkipped, "")
		}
	}

	// Gitignore (append-only)
	if err := EnsureGitignoreEntries(filepath.Join(stateDir, ".gitignore"), expeditionGitignoreEntries); err != nil {
		return result, err
	}
	result.Add(domain.StateDir+"/.gitignore", InitUpdated, "")

	return result, nil
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
