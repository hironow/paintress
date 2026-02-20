package paintress

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateContinent_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	if err := ValidateContinent(dir); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateContinent_CreatesExpeditionDir(t *testing.T) {
	dir := t.TempDir()

	if err := ValidateContinent(dir); err != nil {
		t.Fatalf("should create .expedition dir, got error: %v", err)
	}

	expDir := filepath.Join(dir, ".expedition")
	if _, err := os.Stat(expDir); os.IsNotExist(err) {
		t.Error(".expedition directory should be created")
	}

	journalDir := filepath.Join(expDir, "journal")
	if _, err := os.Stat(journalDir); os.IsNotExist(err) {
		t.Error(".expedition/journal directory should be created")
	}
}

func TestValidateContinent_CreatesGitignore(t *testing.T) {
	dir := t.TempDir()

	if err := ValidateContinent(dir); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	gitignore := filepath.Join(dir, ".expedition", ".gitignore")
	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("should create .gitignore: %v", err)
	}
	if !containsStr(string(content), ".run/") {
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

	if err := ValidateContinent(dir); err != nil {
		t.Fatalf("ValidateContinent: %v", err)
	}

	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !containsStr(string(content), ".run/") {
		t.Errorf(".gitignore should contain .run/ after upgrade, got: %q", string(content))
	}
}

func TestValidateContinent_DoesNotDuplicateRunInGitignore(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	// Already has .run/
	gitignore := filepath.Join(expDir, ".gitignore")
	os.WriteFile(gitignore, []byte(".run/\n"), 0644)

	if err := ValidateContinent(dir); err != nil {
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

func TestValidateContinent_Idempotent(t *testing.T) {
	dir := t.TempDir()

	// Call twice — should not error on second call
	if err := ValidateContinent(dir); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := ValidateContinent(dir); err != nil {
		t.Fatalf("second call: %v", err)
	}
}
