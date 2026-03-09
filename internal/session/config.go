package session

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// ValidateContinent ensures the .expedition directory structure exists.
func ValidateContinent(continent string, logger domain.Logger) error {
	if logger == nil {
		logger = &domain.NopLogger{}
	}
	journalDir := filepath.Join(continent, domain.StateDir, "journal")
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return err
	}

	// Ensure .run/ directory exists for ephemeral files (flag.md, logs/, worktrees/)
	runDir := filepath.Join(continent, domain.StateDir, ".run")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return err
	}

	// Ensure d-mail directories exist (inbox, outbox, archive)
	for _, sub := range []string{"inbox", "outbox", "archive"} {
		d := filepath.Join(continent, domain.StateDir, sub)
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	// Ensure Agent Skills SKILL.md files are always up-to-date with the
	// embedded template. SKILL.md is not user-editable — it is derived from
	// the binary's embedded template and must stay in sync across upgrades.
	skillDirs := []string{"dmail-sendable", "dmail-readable"}
	for _, dir := range skillDirs {
		skillDir := filepath.Join(continent, domain.StateDir, "skills", dir)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		skillFile := filepath.Join(skillDir, "SKILL.md")
		content, err := fs.ReadFile(platform.SkillsFS, filepath.Join("templates", "skills", dir, "SKILL.md"))
		if err != nil {
			return fmt.Errorf("read embedded skill %s: %w", dir, err)
		}
		existing, readErr := os.ReadFile(skillFile)
		if readErr != nil || !bytes.Equal(existing, content) {
			if readErr == nil {
				logger.Info("updated SKILL.md: %s (template changed)", dir)
			}
			if err := os.WriteFile(skillFile, content, 0644); err != nil {
				return err
			}
		}
	}

	// Ensure .run/, inbox/, outbox/ are gitignored
	gitignore := filepath.Join(continent, domain.StateDir, ".gitignore")
	content, err := os.ReadFile(gitignore)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.WriteFile(gitignore, []byte(".run/\ninbox/\noutbox/\n.otel.env\nevents/\n"), 0644); err != nil {
			return err
		}
	} else {
		entries := []string{".run/", "inbox/", "outbox/", ".otel.env", "events/"}
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
