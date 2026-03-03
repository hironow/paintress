package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

// === Flag Tests ===

func TestReadWriteFlag(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	WriteFlag(dir, 5, "AWE-42", "success", "3", 0)
	f := ReadFlag(dir)
	if f.LastExpedition != 5 {
		t.Errorf("LastExpedition = %d", f.LastExpedition)
	}
	if f.Remaining != "3" {
		t.Errorf("Remaining = %q", f.Remaining)
	}
}

// === Journal Tests ===

func TestWriteJournal(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	report := &domain.ExpeditionReport{
		Expedition: 3, IssueID: "AWE-10", IssueTitle: "Test",
		MissionType: "implement", Status: "success", Reason: "done",
		PRUrl: "https://example.com/pr/1", BugIssues: "none",
	}
	if err := WriteJournal(dir, report); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, ".expedition", "journal", "003.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(content), "AWE-10") {
		t.Error("missing issue ID")
	}
}

// === Lumina Tests ===

func TestLumina_ScanEmpty(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	luminas := ScanJournalsForLumina(dir)
	if len(luminas) != 0 {
		t.Errorf("expected 0 luminas from empty journals, got %d", len(luminas))
	}
}

func TestLumina_ScanWithFailures(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// Create journals with repeating failure
	for i := 1; i <= 3; i++ {
		content := `# Expedition #` + string(rune('0'+i)) + ` — Journal

- **Status**: failed
- **Reason**: テストが3回修正しても通らない
- **Mission**: implement
`
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	luminas := ScanJournalsForLumina(dir)
	found := false
	for _, l := range luminas {
		if containsStr(l.Pattern, "テストが3回") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected lumina from repeated failure, got: %v", luminas)
	}
}

func TestLumina_ScanWithSuccesses(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	for i := 1; i <= 4; i++ {
		content := `# Expedition

- **Status**: success
- **Mission**: implement
`
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	luminas := ScanJournalsForLumina(dir)
	found := false
	for _, l := range luminas {
		if containsStr(l.Pattern, "implement") && containsStr(l.Pattern, "Proven approach") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected success lumina, got: %v", luminas)
	}
}
