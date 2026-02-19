package paintress

import (
	"os"
	"path/filepath"
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
