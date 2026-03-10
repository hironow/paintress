package session_test

import (
	"fmt"
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
	session.WriteGommageInsight(w, 5, 3, t.TempDir())

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
	session.WriteGommageInsight(w, 7, 3, t.TempDir())

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
	session.WriteGommageInsight(w, 5, 3, t.TempDir())
	session.WriteGommageInsight(w, 5, 3, t.TempDir())

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
	session.WriteGommageInsight(w, 5, 3, t.TempDir())
	session.WriteGommageInsight(w, 8, 3, t.TempDir())

	// then
	file, _ := w.Read("gommage.md")
	if len(file.Entries) != 2 {
		t.Errorf("expected 2 entries for different expeditions, got %d", len(file.Entries))
	}
}

func TestWriteGommageInsight_IncludesFailureReasons(t *testing.T) {
	// given — continent with journal files containing failure reasons
	continent := t.TempDir()
	journalDir := filepath.Join(continent, ".expedition", "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatal(err)
	}

	reasons := []string{"compile error", "test timeout", "lint failure"}
	for i, reason := range reasons {
		content := fmt.Sprintf("# Expedition #%d — Journal\n- **Status**: failed\n- **Reason**: %s\n", i+1, reason)
		path := filepath.Join(journalDir, fmt.Sprintf("%03d.md", i+1))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)

	// when
	session.WriteGommageInsight(w, 10, 3, continent)

	// then
	file, err := w.Read("gommage.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	entry := file.Entries[0]

	if !strings.Contains(entry.Why, "compile error") {
		t.Errorf("why should contain 'compile error', got %q", entry.Why)
	}
	if !strings.Contains(entry.Why, "test timeout") {
		t.Errorf("why should contain 'test timeout', got %q", entry.Why)
	}
	if !strings.Contains(entry.Why, "lint failure") {
		t.Errorf("why should contain 'lint failure', got %q", entry.Why)
	}
	if !strings.HasPrefix(entry.Why, "Recent failure reasons:") {
		t.Errorf("why should start with 'Recent failure reasons:', got %q", entry.Why)
	}
}

func TestWriteGommageInsight_DeduplicatesReasons(t *testing.T) {
	// given — journals with duplicate reasons
	continent := t.TempDir()
	journalDir := filepath.Join(continent, ".expedition", "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 3; i++ {
		content := fmt.Sprintf("# Expedition #%d — Journal\n- **Status**: failed\n- **Reason**: compile error\n", i)
		path := filepath.Join(journalDir, fmt.Sprintf("%03d.md", i))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)

	// when
	session.WriteGommageInsight(w, 10, 3, continent)

	// then — "compile error" should appear only once
	file, _ := w.Read("gommage.md")
	entry := file.Entries[0]

	count := strings.Count(entry.Why, "compile error")
	if count != 1 {
		t.Errorf("expected 'compile error' once in Why, found %d times: %q", count, entry.Why)
	}
}
