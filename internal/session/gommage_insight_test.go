package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestWriteGommageInsight_CreatesFile(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)

	// when
	session.WriteGommageInsight(w, 5, 3)

	// then
	data, err := os.ReadFile(filepath.Join(insightsDir, "gommage.md"))
	if err != nil {
		t.Fatalf("read gommage.md: %v", err)
	}
	file, err := domain.UnmarshalInsightFile(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(file.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(file.Entries))
	}
	if file.Kind != "gommage" {
		t.Errorf("kind = %q, want gommage", file.Kind)
	}
	if file.Tool != "paintress" {
		t.Errorf("tool = %q, want paintress", file.Tool)
	}
}

func TestWriteGommageInsight_FieldMapping(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)

	// when
	session.WriteGommageInsight(w, 7, 3)

	// then
	file, _ := w.Read("gommage.md")
	entry := file.Entries[0]

	if !strings.Contains(entry.Title, "gommage") {
		t.Errorf("title should contain gommage, got %q", entry.Title)
	}
	if !strings.Contains(entry.Title, "7") {
		t.Errorf("title should contain expedition number, got %q", entry.Title)
	}
	if !strings.Contains(entry.What, "3") {
		t.Errorf("what should contain failure count, got %q", entry.What)
	}
	if !strings.Contains(entry.Why, "systematic") {
		t.Errorf("why should mention systematic issue, got %q", entry.Why)
	}
	if !strings.Contains(entry.Who, "paintress") {
		t.Errorf("who should mention paintress, got %q", entry.Who)
	}
	if entry.Extra["gradient-level"] != "0" {
		t.Errorf("extra gradient-level = %q, want 0", entry.Extra["gradient-level"])
	}
}

func TestWriteGommageInsight_Idempotent(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)

	// when — write same expedition twice
	session.WriteGommageInsight(w, 5, 3)
	session.WriteGommageInsight(w, 5, 3)

	// then — idempotent by title
	file, _ := w.Read("gommage.md")
	if len(file.Entries) != 1 {
		t.Errorf("expected 1 entry (idempotent), got %d", len(file.Entries))
	}
}

func TestWriteGommageInsight_DifferentExpeditions(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)

	// when — different expeditions create different entries
	session.WriteGommageInsight(w, 5, 3)
	session.WriteGommageInsight(w, 8, 3)

	// then
	file, _ := w.Read("gommage.md")
	if len(file.Entries) != 2 {
		t.Errorf("expected 2 entries for different expeditions, got %d", len(file.Entries))
	}
}
