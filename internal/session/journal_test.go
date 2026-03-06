package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestJournalDir(t *testing.T) {
	p := domain.JournalDir("/some/repo")
	want := filepath.Join("/some/repo", ".expedition", "journal")
	if p != want {
		t.Errorf("JournalDir = %q, want %q", p, want)
	}
}

func TestWriteJournal_CreatesDirectoryIfMissing(t *testing.T) {
	dir := t.TempDir()
	// Do not pre-create journal dir — WriteJournal should create it

	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "AWE-1",
		IssueTitle:  "Test Issue",
		MissionType: "implement",
		Status:      "success",
		Reason:      "done",
		PRUrl:       "https://example.com/pr/1",
		BugIssues:   "none",
	}
	if err := session.WriteJournal(dir, report); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, ".expedition", "journal", "001.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("journal file should be created")
	}
}

func TestWriteJournal_ContentFormat(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	report := &domain.ExpeditionReport{
		Expedition:  5,
		IssueID:     "AWE-42",
		IssueTitle:  "Add dark mode",
		MissionType: "implement",
		Status:      "success",
		Reason:      "all tests pass",
		PRUrl:       "https://github.com/org/repo/pull/100",
		BugsFound:   2,
		BugIssues:   "AWE-43,AWE-44",
		Insight:     "Tailwind config uses content paths with glob patterns",
	}

	if err := session.WriteJournal(dir, report); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".expedition", "journal", "005.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	checks := []string{
		"Expedition #5",
		"AWE-42",
		"Add dark mode",
		"implement",
		"success",
		"all tests pass",
		"https://github.com/org/repo/pull/100",
		"Bugs found**: 2",
		"AWE-43,AWE-44",
		"Insight**: Tailwind config uses content paths with glob patterns",
	}
	for _, c := range checks {
		if !strings.Contains(s, c) {
			t.Errorf("journal missing %q", c)
		}
	}
}

func TestWriteJournal_IncludesFailureType(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "AWE-99",
		IssueTitle:  "Fix auth",
		MissionType: "implement",
		Status:      "failed",
		Reason:      "dependency not available",
		FailureType: "blocker",
		PRUrl:       "none",
		BugIssues:   "none",
		Insight:     "External service was down",
	}

	session.WriteJournal(dir, report)

	content, err := os.ReadFile(filepath.Join(dir, ".expedition", "journal", "001.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "**Failure type**: blocker") {
		t.Error("journal should contain failure_type field")
	}
}

func TestWriteJournal_EmptyInsightField(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	report := &domain.ExpeditionReport{
		Expedition:  2,
		IssueID:     "AWE-12",
		IssueTitle:  "No insight",
		MissionType: "implement",
		Status:      "success",
		Reason:      "done",
		PRUrl:       "none",
		BugIssues:   "none",
		Insight:     "",
	}

	session.WriteJournal(dir, report)

	content, err := os.ReadFile(filepath.Join(dir, ".expedition", "journal", "002.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "**Insight**: ") {
		t.Error("journal should include Insight field even when empty")
	}
}

func TestWriteJournal_InsightNotSetDefaultsToEmptyLine(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	report := &domain.ExpeditionReport{
		Expedition:  3,
		IssueID:     "AWE-13",
		IssueTitle:  "Insight omitted",
		MissionType: "implement",
		Status:      "success",
		Reason:      "done",
		PRUrl:       "none",
		BugIssues:   "none",
		// Insight intentionally not set
	}

	session.WriteJournal(dir, report)

	content, err := os.ReadFile(filepath.Join(dir, ".expedition", "journal", "003.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "**Insight**: ") {
		t.Error("journal should include Insight field even when not set")
	}
}

func TestWriteJournal_FilenamePadding(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		expedition int
		filename   string
	}{
		{1, "001.md"},
		{10, "010.md"},
		{100, "100.md"},
	}

	for _, tt := range tests {
		report := &domain.ExpeditionReport{
			Expedition: tt.expedition, IssueID: "X", Status: "success",
			PRUrl: "none", BugIssues: "none",
		}
		session.WriteJournal(dir, report)

		path := filepath.Join(dir, ".expedition", "journal", tt.filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s for expedition %d", tt.filename, tt.expedition)
		}
	}
}

func TestListJournalFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	files, err := session.ListJournalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestListJournalFiles_SkipsZeroFile(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// 000.md should be skipped
	os.WriteFile(filepath.Join(jDir, "000.md"), []byte("skip me"), 0644)
	os.WriteFile(filepath.Join(jDir, "001.md"), []byte("include me"), 0644)

	files, err := session.ListJournalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (000.md excluded), got %d", len(files))
	}
}

func TestListJournalFiles_Sorted(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	os.WriteFile(filepath.Join(jDir, "003.md"), []byte("third"), 0644)
	os.WriteFile(filepath.Join(jDir, "001.md"), []byte("first"), 0644)
	os.WriteFile(filepath.Join(jDir, "002.md"), []byte("second"), 0644)

	files, err := session.ListJournalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	for i, want := range []string{"001.md", "002.md", "003.md"} {
		if filepath.Base(files[i]) != want {
			t.Errorf("files[%d] = %q, want %q", i, filepath.Base(files[i]), want)
		}
	}
}

func TestListJournalFiles_SkipsNonMdFiles(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	os.WriteFile(filepath.Join(jDir, "001.md"), []byte("journal"), 0644)
	os.WriteFile(filepath.Join(jDir, "notes.txt"), []byte("not a journal"), 0644)
	os.WriteFile(filepath.Join(jDir, "data.json"), []byte("{}"), 0644)

	files, err := session.ListJournalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 .md file, got %d", len(files))
	}
}

func TestWriteJournal_HighSeverityDMailField(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	report := &domain.ExpeditionReport{
		Expedition:         1,
		IssueID:            "AWE-50",
		IssueTitle:         "Fix login",
		MissionType:        "implement",
		Status:             "success",
		Reason:             "done",
		PRUrl:              "https://example.com/pr/50",
		BugIssues:          "none",
		HighSeverityDMails: "alert-critical, alert-deploy",
	}

	session.WriteJournal(dir, report)

	content, err := os.ReadFile(filepath.Join(dir, ".expedition", "journal", "001.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.Contains(s, "**HIGH severity D-Mail**: alert-critical, alert-deploy") {
		t.Errorf("journal should contain HIGH severity D-Mail field, got:\n%s", s)
	}
}

func TestWriteJournal_HighSeverityDMailEmpty(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	report := &domain.ExpeditionReport{
		Expedition:  2,
		IssueID:     "AWE-51",
		IssueTitle:  "No alerts",
		MissionType: "implement",
		Status:      "success",
		PRUrl:       "none",
		BugIssues:   "none",
	}

	session.WriteJournal(dir, report)

	content, err := os.ReadFile(filepath.Join(dir, ".expedition", "journal", "002.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "**HIGH severity D-Mail**: ") {
		t.Error("journal should include HIGH severity D-Mail field even when empty")
	}
}

func TestListJournalFiles_NoDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := session.ListJournalFiles(dir)
	if err == nil {
		t.Error("expected error when journal dir doesn't exist")
	}
}

func TestWritePRIndex_AppendsEntry(t *testing.T) {
	// given
	continent := t.TempDir()
	report := &domain.ExpeditionReport{
		Expedition: 1,
		IssueID:    "AWE-42",
		PRUrl:      "https://github.com/org/repo/pull/1",
	}

	// when
	if err := session.WritePRIndex(continent, report); err != nil {
		t.Fatalf("WritePRIndex: %v", err)
	}

	// then
	path := filepath.Join(domain.JournalDir(continent), "pr-index.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pr-index: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "https://github.com/org/repo/pull/1") {
		t.Errorf("expected PR URL in index, got: %s", content)
	}
	if !strings.Contains(content, "AWE-42") {
		t.Errorf("expected issue ID in index, got: %s", content)
	}
}

func TestWritePRIndex_SkipsNone(t *testing.T) {
	continent := t.TempDir()
	report := &domain.ExpeditionReport{
		Expedition: 1,
		IssueID:    "AWE-42",
		PRUrl:      "none",
	}
	if err := session.WritePRIndex(continent, report); err != nil {
		t.Fatalf("WritePRIndex: %v", err)
	}
	path := filepath.Join(domain.JournalDir(continent), "pr-index.jsonl")
	if _, err := os.Stat(path); err == nil {
		t.Error("expected no index file for PRUrl=none")
	}
}

func TestWritePRIndex_AppendsMultiple(t *testing.T) {
	continent := t.TempDir()
	for i := 1; i <= 3; i++ {
		report := &domain.ExpeditionReport{
			Expedition: i,
			IssueID:    "AWE-" + string(rune('0'+i)),
			PRUrl:      "https://github.com/org/repo/pull/" + string(rune('0'+i)),
		}
		if err := session.WritePRIndex(continent, report); err != nil {
			t.Fatalf("WritePRIndex #%d: %v", i, err)
		}
	}
	path := filepath.Join(domain.JournalDir(continent), "pr-index.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %s", len(lines), string(data))
	}
}
