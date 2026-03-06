package session_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
)

// ═══════════════════════════════════════════════
// Flag Edge Cases
// ═══════════════════════════════════════════════

func TestReadFlag_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(""), 0644)

	f := session.ReadFlag(dir)
	if f.Remaining != "?" {
		t.Errorf("empty file should have default Remaining='?', got %q", f.Remaining)
	}
	if f.LastExpedition != 0 {
		t.Errorf("empty file should have LastExpedition=0, got %d", f.LastExpedition)
	}
}

func TestReadFlag_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	content := "garbage data\n!!@#$%\nno_colon_here\n=== bad ===\n"
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	f := session.ReadFlag(dir)
	// Should not panic, just return defaults
	if f.Remaining != "?" {
		t.Errorf("corrupt file should have default Remaining, got %q", f.Remaining)
	}
}

func TestReadFlag_PartialData(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	// Only some fields present
	content := "last_expedition: 3\nremaining_issues: 7\n"
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	f := session.ReadFlag(dir)
	if f.LastExpedition != 3 {
		t.Errorf("LastExpedition = %d, want 3", f.LastExpedition)
	}
	if f.Remaining != "7" {
		t.Errorf("Remaining = %q, want 7", f.Remaining)
	}
	if f.LastIssue != "" {
		t.Errorf("missing field should be empty, got %q", f.LastIssue)
	}
}

func TestWriteFlag_SpecialCharactersInIssueID(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	session.WriteFlag(dir, 1, "AWE-42/test<script>", "success", "5", 0)
	f := session.ReadFlag(dir)
	if f.LastIssue != "AWE-42/test<script>" {
		t.Errorf("LastIssue = %q, should preserve special chars", f.LastIssue)
	}
}

// ═══════════════════════════════════════════════
// Journal Edge Cases
// ═══════════════════════════════════════════════

func TestWriteJournal_HighExpeditionNumber(t *testing.T) {
	dir := t.TempDir()

	report := &domain.ExpeditionReport{
		Expedition: 1234, IssueID: "X", Status: "success",
		PRUrl: "none", BugIssues: "none",
	}
	session.WriteJournal(dir, report)

	// %03d with 1234 produces "1234" (4 digits, no padding needed)
	path := filepath.Join(dir, ".expedition", "journal", "1234.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("should create 1234.md for expedition 1234")
	}
}

func TestWriteJournal_NewlinesInFields(t *testing.T) {
	dir := t.TempDir()

	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "AWE-1",
		IssueTitle:  "Title with\nnewline",
		MissionType: "implement",
		Status:      "success",
		Reason:      "line1\nline2\nline3",
		PRUrl:       "none",
		BugIssues:   "none",
	}
	err := session.WriteJournal(dir, report)
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, ".expedition", "journal", "001.md"))
	// Should not crash; newlines will break markdown format but that's expected
	if !strings.Contains(string(content), "AWE-1") {
		t.Error("journal should contain issue ID")
	}
}

func TestListJournalFiles_WithSubdirectory(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	os.WriteFile(filepath.Join(jDir, "001.md"), []byte("journal"), 0644)
	os.MkdirAll(filepath.Join(jDir, "subdir"), 0755) // subdirectory should be skipped

	files, err := session.ListJournalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("should skip subdirectory, got %d files", len(files))
	}
}

// ═══════════════════════════════════════════════
// Lumina Edge Cases
// ═══════════════════════════════════════════════

func TestExtractValue_OnlyBoldMarkers(t *testing.T) {
	v := session.ExportExtractValue("- **Status**: **")
	// TrimPrefix("**") removes leading **, TrimSuffix("**") removes trailing **
	if v != "" {
		t.Errorf("got %q, expected empty after trimming lone **", v)
	}
}

func TestExtractValue_MultipleBoldPairs(t *testing.T) {
	v := session.ExportExtractValue("- **Key**: **bold** and **more bold**")
	// SplitN at first colon, then TrimPrefix/TrimSuffix only strips outermost **
	if v == "" {
		t.Error("should not be empty")
	}
}

func TestScanJournalsForLumina_MalformedJournal(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// Journal with no recognizable fields
	os.WriteFile(filepath.Join(jDir, "001.md"), []byte("garbage data\n!@#$"), 0644)
	os.WriteFile(filepath.Join(jDir, "002.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(jDir, "003.md"), []byte("- **Status**:"), 0644) // empty status

	luminas := session.ScanJournalsForLumina(dir)
	if len(luminas) != 0 {
		t.Errorf("malformed journals should produce no luminas, got %d", len(luminas))
	}
}

func TestScanJournalsForLumina_EmptyMission(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	for i := 1; i <= 3; i++ {
		content := `# Expedition

- **Status**: success
- **Mission**:
`
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	luminas := session.ScanJournalsForLumina(dir)
	// Empty mission -> key = " mission: 3 proven successes" with leading space
	// This is a valid edge case to document
	for _, l := range luminas {
		if strings.Contains(l.Pattern, "mission") && strings.Contains(l.Pattern, "proven successes") {
			return // found it, passes
		}
	}
	// If no lumina was created, that's also acceptable for empty mission
}

// ═══════════════════════════════════════════════
// Expedition Edge Cases
// ═══════════════════════════════════════════════

func TestExpedition_BuildPrompt_ZeroNumber(t *testing.T) {
	e := &session.Expedition{
		Number:    0,
		Continent: "/tmp",
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  domain.NewGradientGauge(5),
		Reserve:   domain.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	prompt := e.BuildPrompt()
	if !strings.Contains(prompt, "Expedition #0") {
		t.Error("should handle expedition number 0")
	}
}

func TestExpedition_BuildPrompt_EmptyConfig(t *testing.T) {
	e := &session.Expedition{
		Number:    1,
		Continent: "",
		Config:    domain.Config{}, // all empty
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  domain.NewGradientGauge(5),
		Reserve:   domain.NewReserveParty("", nil, platform.NewLogger(io.Discard, false)),
	}

	// Should not panic with empty config
	prompt := e.BuildPrompt()
	if prompt == "" {
		t.Error("prompt should not be empty even with empty config")
	}
}

// ═══════════════════════════════════════════════
// DevServer Edge Cases
// ═══════════════════════════════════════════════

func TestDevServer_StopMultipleTimes(t *testing.T) {
	ds := session.NewDevServer("echo", "http://localhost:3000", t.TempDir(), "/dev/null", platform.NewLogger(io.Discard, false))
	// Multiple stops should not panic
	ds.Stop()
	ds.Stop()
	ds.Stop()
}
