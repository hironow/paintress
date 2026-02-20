package paintress

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArchivePrune_DryRun_ListsOldFiles(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, ".expedition", "archive")
	os.MkdirAll(archiveDir, 0755)

	// given: one old file (mtime 40 days ago) and one recent file
	oldFile := filepath.Join(archiveDir, "report-my-1.md")
	newFile := filepath.Join(archiveDir, "report-my-2.md")
	os.WriteFile(oldFile, []byte("old"), 0644)
	os.WriteFile(newFile, []byte("new"), 0644)
	os.Chtimes(oldFile, time.Now().Add(-40*24*time.Hour), time.Now().Add(-40*24*time.Hour))

	// when: dry-run with 30 days threshold
	result, err := ArchivePrune(dir, 30, false)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(result.Candidates))
	}
	if result.Candidates[0] != "report-my-1.md" {
		t.Errorf("candidate = %q, want report-my-1.md", result.Candidates[0])
	}
	if result.Deleted != 0 {
		t.Errorf("deleted = %d, want 0 (dry-run)", result.Deleted)
	}
}

func TestArchivePrune_Execute_DeletesOldFiles(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, ".expedition", "archive")
	os.MkdirAll(archiveDir, 0755)

	// given: two old files and one recent
	old1 := filepath.Join(archiveDir, "report-my-1.md")
	old2 := filepath.Join(archiveDir, "feedback-001.md")
	recent := filepath.Join(archiveDir, "report-my-3.md")
	os.WriteFile(old1, []byte("old1"), 0644)
	os.WriteFile(old2, []byte("old2"), 0644)
	os.WriteFile(recent, []byte("new"), 0644)
	past := time.Now().Add(-40 * 24 * time.Hour)
	os.Chtimes(old1, past, past)
	os.Chtimes(old2, past, past)

	// when: execute with 30 days threshold
	result, err := ArchivePrune(dir, 30, true)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deleted != 2 {
		t.Errorf("deleted = %d, want 2", result.Deleted)
	}
	if len(result.Candidates) != 2 {
		t.Errorf("candidates = %d, want 2", len(result.Candidates))
	}

	// old files should be gone
	if _, err := os.Stat(old1); !os.IsNotExist(err) {
		t.Error("old1 should be deleted")
	}
	if _, err := os.Stat(old2); !os.IsNotExist(err) {
		t.Error("old2 should be deleted")
	}
	// recent file should remain
	if _, err := os.Stat(recent); err != nil {
		t.Error("recent file should still exist")
	}
}

func TestArchivePrune_NoArchiveDir_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	// no .expedition/archive/ directory

	// when
	result, err := ArchivePrune(dir, 30, false)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Errorf("candidates = %d, want 0", len(result.Candidates))
	}
}

func TestArchivePrune_IgnoresNonMdFiles(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, ".expedition", "archive")
	os.MkdirAll(archiveDir, 0755)

	// given: old .txt file and old .md file
	oldTxt := filepath.Join(archiveDir, "notes.txt")
	oldMd := filepath.Join(archiveDir, "report-my-1.md")
	os.WriteFile(oldTxt, []byte("txt"), 0644)
	os.WriteFile(oldMd, []byte("md"), 0644)
	past := time.Now().Add(-40 * 24 * time.Hour)
	os.Chtimes(oldTxt, past, past)
	os.Chtimes(oldMd, past, past)

	// when
	result, err := ArchivePrune(dir, 30, false)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates = %d, want 1 (only .md)", len(result.Candidates))
	}
	if result.Candidates[0] != "report-my-1.md" {
		t.Errorf("candidate = %q, want report-my-1.md", result.Candidates[0])
	}
}

func TestArchivePrune_AllRecent_NoCandidates(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, ".expedition", "archive")
	os.MkdirAll(archiveDir, 0755)

	// given: only recent files
	os.WriteFile(filepath.Join(archiveDir, "report-my-1.md"), []byte("new"), 0644)
	os.WriteFile(filepath.Join(archiveDir, "report-my-2.md"), []byte("new"), 0644)

	// when
	result, err := ArchivePrune(dir, 30, false)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Errorf("candidates = %d, want 0 (all recent)", len(result.Candidates))
	}
}

func TestArchivePrune_NegativeDays_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	_, err := ArchivePrune(dir, -7, false)
	if err == nil {
		t.Fatal("expected error for negative days, got nil")
	}
}

func TestArchivePrune_ZeroDays_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	_, err := ArchivePrune(dir, 0, false)
	if err == nil {
		t.Fatal("expected error for zero days, got nil")
	}
}

func TestArchivePrune_CustomDays(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, ".expedition", "archive")
	os.MkdirAll(archiveDir, 0755)

	// given: file 10 days old
	f := filepath.Join(archiveDir, "report-my-1.md")
	os.WriteFile(f, []byte("data"), 0644)
	os.Chtimes(f, time.Now().Add(-10*24*time.Hour), time.Now().Add(-10*24*time.Hour))

	// when: threshold is 7 days — file should be a candidate
	result7, err := ArchivePrune(dir, 7, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result7.Candidates) != 1 {
		t.Errorf("days=7: candidates = %d, want 1", len(result7.Candidates))
	}

	// when: threshold is 15 days — file should NOT be a candidate
	result15, err := ArchivePrune(dir, 15, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result15.Candidates) != 0 {
		t.Errorf("days=15: candidates = %d, want 0", len(result15.Candidates))
	}
}
