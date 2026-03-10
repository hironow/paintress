package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestWriteLuminaInsights_CreatesFile(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)
	luminas := []domain.Lumina{
		{Pattern: "[WARN] Avoid — failed 3 times: lint error", Source: "failure-pattern", Uses: 3},
		{Pattern: "[OK] Proven approach (4x successful): TDD cycle", Source: "success-pattern", Uses: 4},
	}

	// when
	session.WriteLuminaInsights(w, luminas)

	// then
	data, err := os.ReadFile(filepath.Join(insightsDir, "lumina.md"))
	if err != nil {
		t.Fatalf("read lumina.md: %v", err)
	}
	file, err := domain.UnmarshalInsightFile(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(file.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(file.Entries))
	}
	if file.Kind != "lumina" {
		t.Errorf("kind = %q, want lumina", file.Kind)
	}
	if file.Tool != "paintress" {
		t.Errorf("tool = %q, want paintress", file.Tool)
	}
}

func TestWriteLuminaInsights_FailurePatternMapping(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)
	luminas := []domain.Lumina{
		{Pattern: "[WARN] Avoid — failed 2 times: tests timeout", Source: "failure-pattern", Uses: 2},
	}

	// when
	session.WriteLuminaInsights(w, luminas)

	// then
	file, _ := w.Read("lumina.md")
	entry := file.Entries[0]

	if !strings.Contains(entry.How, "defensive") {
		t.Errorf("how should mention defensive strategy, got %q", entry.How)
	}
	if entry.Extra["failure-type"] != "failure-pattern" {
		t.Errorf("extra failure-type = %q, want failure-pattern", entry.Extra["failure-type"])
	}
	if !strings.Contains(entry.Constraints, "2") {
		t.Errorf("constraints should mention count, got %q", entry.Constraints)
	}
}

func TestWriteLuminaInsights_SuccessPatternMapping(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)
	luminas := []domain.Lumina{
		{Pattern: "[OK] Proven approach (3x successful): implement", Source: "success-pattern", Uses: 3},
	}

	// when
	session.WriteLuminaInsights(w, luminas)

	// then
	file, _ := w.Read("lumina.md")
	entry := file.Entries[0]

	if !strings.Contains(strings.ToLower(entry.How), "continue") {
		t.Errorf("how should mention continue doing, got %q", entry.How)
	}
}

func TestWriteLuminaInsights_HighSeverityAlertMapping(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)
	luminas := []domain.Lumina{
		{Pattern: "[ALERT] HIGH severity D-Mail in past expedition: alert-critical", Source: "high-severity-alert", Uses: 1},
	}

	// when
	session.WriteLuminaInsights(w, luminas)

	// then
	file, _ := w.Read("lumina.md")
	entry := file.Entries[0]

	if !strings.Contains(entry.How, "immediate") {
		t.Errorf("how should mention immediate attention, got %q", entry.How)
	}
	if entry.Extra["failure-type"] != "high-severity-alert" {
		t.Errorf("extra failure-type = %q, want high-severity-alert", entry.Extra["failure-type"])
	}
}

func TestWriteLuminaInsights_EmptyLuminas(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)

	// when
	session.WriteLuminaInsights(w, nil)

	// then — no file created
	_, err := os.ReadFile(filepath.Join(insightsDir, "lumina.md"))
	if err == nil {
		t.Error("expected no lumina.md for empty luminas")
	}
}

func TestWriteLuminaInsights_Idempotent(t *testing.T) {
	// given
	dir := t.TempDir()
	insightsDir := filepath.Join(dir, "insights")
	runDir := filepath.Join(dir, ".run")
	os.MkdirAll(insightsDir, 0o755)
	os.MkdirAll(runDir, 0o755)

	w := session.NewInsightWriter(insightsDir, runDir)
	luminas := []domain.Lumina{
		{Pattern: "[WARN] Avoid — failed 2 times: lint error", Source: "failure-pattern", Uses: 2},
	}

	// when — write twice
	session.WriteLuminaInsights(w, luminas)
	session.WriteLuminaInsights(w, luminas)

	// then — still only 1 entry (idempotent by title)
	file, _ := w.Read("lumina.md")
	if len(file.Entries) != 1 {
		t.Errorf("expected 1 entry (idempotent), got %d", len(file.Entries))
	}
}
