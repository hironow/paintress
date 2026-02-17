package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJournalDir(t *testing.T) {
	p := JournalDir("/some/repo")
	want := filepath.Join("/some/repo", ".expedition", "journal")
	if p != want {
		t.Errorf("JournalDir = %q, want %q", p, want)
	}
}

func TestWriteJournal_CreatesDirectoryIfMissing(t *testing.T) {
	dir := t.TempDir()
	// Do not pre-create journal dir — WriteJournal should create it

	report := &ExpeditionReport{
		Expedition:  1,
		IssueID:     "AWE-1",
		IssueTitle:  "Test Issue",
		MissionType: "implement",
		Status:      "success",
		Reason:      "done",
		PRUrl:       "https://example.com/pr/1",
		BugIssues:   "none",
	}
	if err := WriteJournal(dir, report); err != nil {
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

	report := &ExpeditionReport{
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

	if err := WriteJournal(dir, report); err != nil {
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
		if !containsStr(s, c) {
			t.Errorf("journal missing %q", c)
		}
	}
}

func TestWriteJournal_IncludesFailureType(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	report := &ExpeditionReport{
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

	WriteJournal(dir, report)

	content, err := os.ReadFile(filepath.Join(dir, ".expedition", "journal", "001.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "**Failure type**: blocker") {
		t.Error("journal should contain failure_type field")
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
		report := &ExpeditionReport{
			Expedition: tt.expedition, IssueID: "X", Status: "success",
			PRUrl: "none", BugIssues: "none",
		}
		WriteJournal(dir, report)

		path := filepath.Join(dir, ".expedition", "journal", tt.filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s for expedition %d", tt.filename, tt.expedition)
		}
	}
}

func TestListJournalFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	files, err := ListJournalFiles(dir)
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

	files, err := ListJournalFiles(dir)
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

	files, err := ListJournalFiles(dir)
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

	files, err := ListJournalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 .md file, got %d", len(files))
	}
}

func TestListJournalFiles_NoDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := ListJournalFiles(dir)
	if err == nil {
		t.Error("expected error when journal dir doesn't exist")
	}
}
