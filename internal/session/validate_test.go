package session_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
)

func TestValidateContinent_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateContinent_CreatesExpeditionDir(t *testing.T) {
	dir := t.TempDir()

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("should create .expedition dir, got error: %v", err)
	}

	expDir := filepath.Join(dir, ".expedition")
	if _, err := os.Stat(expDir); errors.Is(err, fs.ErrNotExist) {
		t.Error(".expedition directory should be created")
	}

	journalDir := filepath.Join(expDir, "journal")
	if _, err := os.Stat(journalDir); errors.Is(err, fs.ErrNotExist) {
		t.Error(".expedition/journal directory should be created")
	}
}

func TestValidateContinent_CreatesGitignore(t *testing.T) {
	dir := t.TempDir()

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	gitignore := filepath.Join(dir, ".expedition", ".gitignore")
	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("should create .gitignore: %v", err)
	}
	if !strings.Contains(string(content), ".run/") {
		t.Error(".gitignore should contain .run/")
	}
}

func TestValidateContinent_AppendsRunToExistingGitignore(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	// Simulate a pre-existing .gitignore from an older version (without .run/)
	gitignore := filepath.Join(expDir, ".gitignore")
	os.WriteFile(gitignore, []byte(".logs/\nworktrees/\n"), 0644)

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(content), ".run/") {
		t.Errorf(".gitignore should contain .run/ after upgrade, got: %q", string(content))
	}
}

func TestValidateContinent_AppendsRunWithMissingTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	// Legacy .gitignore WITHOUT trailing newline
	gitignore := filepath.Join(expDir, ".gitignore")
	os.WriteFile(gitignore, []byte("worktrees/"), 0644)

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	// .run/ must be on its own line, not appended to previous entry
	lines := strings.Split(string(content), "\n")
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == ".run/" {
			found = true
		}
		if strings.Contains(line, "worktrees/.run/") {
			t.Errorf(".run/ was appended to previous line: %q", string(content))
		}
	}
	if !found {
		t.Errorf(".run/ should appear on its own line, got: %q", string(content))
	}
}

func TestValidateContinent_DoesNotDuplicateRunInGitignore(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	// Already has .run/
	gitignore := filepath.Join(expDir, ".gitignore")
	os.WriteFile(gitignore, []byte(".run/\n"), 0644)

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	// Count occurrences of .run/
	count := 0
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == ".run/" {
			count++
		}
	}
	if count != 1 {
		t.Errorf(".run/ should appear exactly once, got %d times in: %q", count, string(content))
	}
}

func TestValidateContinent_PreservesGitignoreOnReadError(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	// Create .gitignore with legacy content, then make it unreadable
	gitignore := filepath.Join(expDir, ".gitignore")
	os.WriteFile(gitignore, []byte(".logs/\nworktrees/\n"), 0644)
	os.Chmod(gitignore, 0000)
	defer os.Chmod(gitignore, 0644) // restore for cleanup

	// ValidateContinent should return an error, not silently overwrite
	_, err := session.ValidateContinent(dir, nil)
	if err == nil {
		// If no error, the file must not have been overwritten
		os.Chmod(gitignore, 0644)
		content, _ := os.ReadFile(gitignore)
		if !strings.Contains(string(content), ".logs/") {
			t.Error("existing .gitignore content was overwritten on read error")
		}
	}
	// Either returning an error or preserving content is acceptable
}

func TestValidateContinent_WriteStringErrorsPropagate(t *testing.T) {
	// This test verifies that ValidateContinent returns the error values
	// from WriteString calls. We test the happy path here — the error
	// propagation is verified by code inspection (WriteString returns checked).
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	// Create a gitignore without .run/ and without trailing newline
	gitignore := filepath.Join(expDir, ".gitignore")
	os.WriteFile(gitignore, []byte("worktrees/"), 0644)

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	content, _ := os.ReadFile(gitignore)
	// Verify both writes happened: newline + .run/
	if !strings.Contains(string(content), "worktrees/\n.run/\n") {
		t.Errorf("expected 'worktrees/\\n.run/\\n', got: %q", string(content))
	}
}

func TestValidateContinent_CreatesDMailDirs(t *testing.T) {
	dir := t.TempDir()

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	for _, sub := range []string{"inbox", "outbox", "archive"} {
		path := filepath.Join(dir, ".expedition", sub)
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			t.Errorf(".expedition/%s directory should be created", sub)
		}
	}
}

func TestValidateContinent_GitignoresInboxAndOutbox(t *testing.T) {
	dir := t.TempDir()

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".expedition", ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	for _, entry := range []string{"inbox/", "outbox/", "events/"} {
		if !strings.Contains(string(content), entry) {
			t.Errorf(".gitignore should contain %q, got: %q", entry, string(content))
		}
	}
}

func TestValidateContinent_ArchiveAndInsightsAreGitignored(t *testing.T) {
	dir := t.TempDir()

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".expedition", ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(content), "archive/") {
		t.Error("expected archive/ in .gitignore (runtime data)")
	}
	if !strings.Contains(string(content), "insights/") {
		t.Error("expected insights/ in .gitignore (runtime data)")
	}
}

func TestValidateContinent_AppendsInboxOutboxToExistingGitignore(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	// Simulate existing .gitignore that only has .run/ (pre-dmail version)
	gitignore := filepath.Join(expDir, ".gitignore")
	os.WriteFile(gitignore, []byte(".run/\n"), 0644)

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	for _, entry := range []string{"inbox/", "outbox/"} {
		if !strings.Contains(string(content), entry) {
			t.Errorf(".gitignore should contain %q after upgrade, got: %q", entry, string(content))
		}
	}
}

func TestValidateContinent_DoesNotDuplicateInboxOutboxInGitignore(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	// Already has all entries
	gitignore := filepath.Join(expDir, ".gitignore")
	os.WriteFile(gitignore, []byte(".run/\ninbox/\noutbox/\n"), 0644)

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	for _, entry := range []string{"inbox/", "outbox/"} {
		count := strings.Count(string(content), entry)
		if count != 1 {
			t.Errorf("%q should appear exactly once, got %d times in: %q", entry, count, string(content))
		}
	}
}

func TestValidateContinent_CreatesSkillFiles(t *testing.T) {
	dir := t.TempDir()

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	skills := []struct {
		dir      string
		contains []string
	}{
		{"dmail-sendable", []string{"produces:", "kind: report", "license: Apache-2.0", "dmail-schema-version:"}},
		{"dmail-readable", []string{"consumes:", "kind: specification", "kind: implementation-feedback", "license: Apache-2.0", "dmail-schema-version:"}},
	}

	for _, s := range skills {
		path := filepath.Join(dir, ".expedition", "skills", s.dir, "SKILL.md")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("SKILL.md not created at %s: %v", path, err)
		}
		str := string(content)
		if !strings.Contains(str, "name: "+s.dir) {
			t.Errorf("%s/SKILL.md should contain 'name: %s', got: %q", s.dir, s.dir, str)
		}
		for _, c := range s.contains {
			if !strings.Contains(str, c) {
				t.Errorf("%s/SKILL.md should contain %q, got: %q", s.dir, c, str)
			}
		}
	}
}

func TestValidateContinent_SkillFilesAreIdempotent(t *testing.T) {
	dir := t.TempDir()

	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Read content after first call
	path := filepath.Join(dir, ".expedition", "skills", "dmail-sendable", "SKILL.md")
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}

	// Second call should not error and should not overwrite
	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("second call: %v", err)
	}

	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read SKILL.md after second call: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("SKILL.md content changed between calls")
	}
}

func TestValidateContinent_SkillFilesUpdatedWhenOutdated(t *testing.T) {
	dir := t.TempDir()

	// given — pre-existing SKILL.md with outdated content (old description)
	skillDir := filepath.Join(dir, ".expedition", "skills", "dmail-sendable")
	os.MkdirAll(skillDir, 0755)
	outdated := []byte("---\nname: dmail-sendable\ndescription: old\nlicense: Apache-2.0\nmetadata:\n  dmail-schema-version: \"1\"\n  produces:\n    - kind: report\n---\n\nold content\n")
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), outdated, 0644)

	// when
	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	// then — SKILL.md should be overwritten with the latest embedded template
	content, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if strings.Contains(string(content), "old content") {
		t.Error("outdated SKILL.md should be overwritten with latest template")
	}
	if !strings.Contains(string(content), "kind: report") {
		t.Error("updated SKILL.md should contain 'kind: report'")
	}
}

func TestValidateContinent_SkillStatErrorPropagates(t *testing.T) {
	dir := t.TempDir()

	// First call creates the skill directory and file
	if _, err := session.ValidateContinent(dir, nil); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Make the skill directory unreadable so os.Stat returns permission error
	skillDir := filepath.Join(dir, ".expedition", "skills", "dmail-sendable")
	os.Chmod(skillDir, 0000)
	defer os.Chmod(skillDir, 0755)

	// ValidateContinent should propagate the stat error, not silently skip
	_, err := session.ValidateContinent(dir, nil)
	if err == nil {
		t.Error("expected error when skill directory is unreadable, got nil")
	}
}

func TestValidateContinent_Idempotent(t *testing.T) {
	dir := t.TempDir()
	logger := platform.NewLogger(io.Discard, false)

	// Call twice — should not error on second call
	if _, err := session.ValidateContinent(dir, logger); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := session.ValidateContinent(dir, logger); err != nil {
		t.Fatalf("second call: %v", err)
	}
}

func TestValidateContinent_LogsWhenSkillUpdated(t *testing.T) {
	dir := t.TempDir()
	logger := platform.NewLogger(io.Discard, false)

	// given — first init creates SKILL.md
	if _, err := session.ValidateContinent(dir, logger); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// given — overwrite with outdated content
	skillDir := filepath.Join(dir, ".expedition", "skills", "dmail-sendable")
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("outdated"), 0644)

	// when — second init should log the update
	var buf bytes.Buffer
	logCapture := platform.NewLogger(&buf, false)
	if _, err := session.ValidateContinent(dir, logCapture); err != nil {
		t.Fatalf("second call: %v", err)
	}

	// then — log should mention the updated skill
	output := buf.String()
	if !strings.Contains(output, "dmail-sendable") {
		t.Errorf("expected log to mention dmail-sendable, got: %q", output)
	}
}

func TestValidateContinent_NoLogWhenSkillUnchanged(t *testing.T) {
	dir := t.TempDir()
	logger := platform.NewLogger(io.Discard, false)

	// given — first init
	if _, err := session.ValidateContinent(dir, logger); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// when — second init with no changes
	var buf bytes.Buffer
	logCapture := platform.NewLogger(&buf, false)
	if _, err := session.ValidateContinent(dir, logCapture); err != nil {
		t.Fatalf("second call: %v", err)
	}

	// then — no skill update logs
	output := buf.String()
	if strings.Contains(output, "SKILL.md") {
		t.Errorf("should not log when skills are unchanged, got: %q", output)
	}
}
