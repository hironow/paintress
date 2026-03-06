package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestParseKV_Normal(t *testing.T) {
	k, v, ok := session.ExportParseKV("last_expedition: 5")
	if !ok {
		t.Fatal("expected ok")
	}
	if k != "last_expedition" || v != "5" {
		t.Errorf("got k=%q v=%q", k, v)
	}
}

func TestParseKV_Comment(t *testing.T) {
	_, _, ok := session.ExportParseKV("# this is a comment")
	if ok {
		t.Error("comments should return false")
	}
}

func TestParseKV_Empty(t *testing.T) {
	_, _, ok := session.ExportParseKV("")
	if ok {
		t.Error("empty line should return false")
	}
}

func TestParseKV_NoColon(t *testing.T) {
	_, _, ok := session.ExportParseKV("no colon here")
	if ok {
		t.Error("line without colon should return false")
	}
}

func TestParseKV_ValueWithColon(t *testing.T) {
	k, v, ok := session.ExportParseKV("last_updated: 2024-01-01 12:00:00")
	if !ok {
		t.Fatal("expected ok")
	}
	if k != "last_updated" {
		t.Errorf("key = %q", k)
	}
	if v != "2024-01-01 12:00:00" {
		t.Errorf("value = %q, want time string with colon", v)
	}
}

func TestParseKV_WhitespaceOnly(t *testing.T) {
	_, _, ok := session.ExportParseKV("   ")
	if ok {
		t.Error("whitespace-only line should return false")
	}
}

func TestReadFlag_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	f := session.ReadFlag(dir)
	if f.Remaining != "?" {
		t.Errorf("default Remaining should be '?', got %q", f.Remaining)
	}
	if f.LastExpedition != 0 {
		t.Errorf("default LastExpedition should be 0, got %d", f.LastExpedition)
	}
}

func TestFlagPath(t *testing.T) {
	p := domain.FlagPath("/some/repo")
	want := filepath.Join("/some/repo", ".expedition", ".run", "flag.md")
	if p != want {
		t.Errorf("FlagPath = %q, want %q", p, want)
	}
}

func TestWriteFlag_AllFields(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	session.WriteFlag(dir, 10, "AWE-99", "success", "0", 0)
	f := session.ReadFlag(dir)

	if f.LastExpedition != 10 {
		t.Errorf("LastExpedition = %d, want 10", f.LastExpedition)
	}
	if f.LastIssue != "AWE-99" {
		t.Errorf("LastIssue = %q, want AWE-99", f.LastIssue)
	}
	if f.LastStatus != "success" {
		t.Errorf("LastStatus = %q, want success", f.LastStatus)
	}
	if f.Remaining != "0" {
		t.Errorf("Remaining = %q, want 0", f.Remaining)
	}
	if f.LastUpdated == "" {
		t.Error("LastUpdated should not be empty")
	}
}

func TestWriteFlag_Overwrite(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	session.WriteFlag(dir, 1, "AWE-1", "success", "10", 0)
	session.WriteFlag(dir, 2, "AWE-2", "failed", "9", 0)

	f := session.ReadFlag(dir)
	if f.LastExpedition != 2 {
		t.Errorf("should reflect latest write, got %d", f.LastExpedition)
	}
	if f.LastIssue != "AWE-2" {
		t.Errorf("LastIssue = %q, want AWE-2", f.LastIssue)
	}
}

func TestReadFlag_ValueWithColonAndSpaces(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	content := `last_expedition: 7
remaining_issues: 10 (approx): 3 left
`
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	f := session.ReadFlag(dir)
	if f.LastExpedition != 7 {
		t.Errorf("LastExpedition = %d, want 7", f.LastExpedition)
	}
	if f.Remaining != "10 (approx): 3 left" {
		t.Errorf("Remaining = %q, want %q", f.Remaining, "10 (approx): 3 left")
	}
}

func TestWriteFlag_IssueIDWithNewline_IsSanitized(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	issueID := "AWE-1\nAWE-2"
	session.WriteFlag(dir, 1, issueID, "success", "5", 0)

	f := session.ReadFlag(dir)
	if f.LastIssue != "AWE-1 AWE-2" {
		t.Errorf("LastIssue = %q, want %q", f.LastIssue, "AWE-1 AWE-2")
	}
}

func TestWriteFlag_SanitizesStatusAndRemaining(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	session.WriteFlag(dir, 1, "AWE-1", "success\nextra", "5\r\nmore", 0)
	f := session.ReadFlag(dir)

	if f.LastStatus != "success extra" {
		t.Errorf("LastStatus = %q, want %q", f.LastStatus, "success extra")
	}
	if f.Remaining != "5  more" {
		t.Errorf("Remaining = %q, want %q", f.Remaining, "5  more")
	}
}

func TestReadFlag_CurrentIssueAndTitle(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	content := `last_expedition: 3
current_issue: MY-239
current_title: flag.md watcher
`
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	f := session.ReadFlag(dir)
	if f.CurrentIssue != "MY-239" {
		t.Errorf("CurrentIssue = %q, want %q", f.CurrentIssue, "MY-239")
	}
	if f.CurrentTitle != "flag.md watcher" {
		t.Errorf("CurrentTitle = %q, want %q", f.CurrentTitle, "flag.md watcher")
	}
}

func TestReadFlag_CurrentIssueAbsent_DefaultsEmpty(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	content := `last_expedition: 1
remaining_issues: 5
`
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	f := session.ReadFlag(dir)
	if f.CurrentIssue != "" {
		t.Errorf("CurrentIssue should default to empty, got %q", f.CurrentIssue)
	}
	if f.CurrentTitle != "" {
		t.Errorf("CurrentTitle should default to empty, got %q", f.CurrentTitle)
	}
}

func TestReadFlag_InvalidAndNegativeExpedition(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	content := `last_expedition: not-a-number
remaining_issues: 1
`
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	f := session.ReadFlag(dir)
	// fmt.Sscanf should leave LastExpedition at zero on parse failure.
	if f.LastExpedition != 0 {
		t.Errorf("LastExpedition = %d, want 0 on parse failure", f.LastExpedition)
	}

	content = `last_expedition: -5
remaining_issues: 1
`
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	f = session.ReadFlag(dir)
	// Negative values are currently accepted by fmt.Sscanf.
	if f.LastExpedition != -5 {
		t.Errorf("LastExpedition = %d, want -5 for negative values", f.LastExpedition)
	}
}

func TestWriteFlag_MidHighSeverity(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	// given — write flag with mid-high severity count
	session.WriteFlag(dir, 5, "MY-42", "success", "3", 2)
	f := session.ReadFlag(dir)

	// then — mid_high_severity should be recorded
	if f.MidHighSeverity != 2 {
		t.Errorf("MidHighSeverity = %d, want 2", f.MidHighSeverity)
	}
	// other fields should be unaffected
	if f.LastExpedition != 5 {
		t.Errorf("LastExpedition = %d, want 5", f.LastExpedition)
	}
}

func TestWriteFlag_MidHighSeverityZero(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	// given — write flag with zero mid-high severity
	session.WriteFlag(dir, 3, "MY-10", "failed", "5", 0)
	f := session.ReadFlag(dir)

	// then — mid_high_severity should be 0 (not omitted)
	if f.MidHighSeverity != 0 {
		t.Errorf("MidHighSeverity = %d, want 0", f.MidHighSeverity)
	}
}

func TestReadFlag_MidHighSeverity(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	// given — flag.md with mid_high_severity field
	content := `last_expedition: 7
last_issue: MY-50
last_status: success
remaining_issues: 2
mid_high_severity: 3
`
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	// when
	f := session.ReadFlag(dir)

	// then
	if f.MidHighSeverity != 3 {
		t.Errorf("MidHighSeverity = %d, want 3", f.MidHighSeverity)
	}
}

// === reconcileFlags Tests ===

func TestReconcileFlags_FindsMaxAcrossWorktrees(t *testing.T) {
	// given — continent flag (exp 5), worker-001 (exp 7), worker-002 (exp 3)
	continent := t.TempDir()

	// Continent's own flag.md
	os.MkdirAll(filepath.Join(continent, ".expedition", ".run"), 0755)
	session.WriteFlag(continent, 5, "MY-10", "success", "5", 0)

	// worker-001 flag.md — the highest
	w1 := filepath.Join(continent, ".expedition", ".run", "worktrees", "worker-001")
	os.MkdirAll(filepath.Join(w1, ".expedition", ".run"), 0755)
	session.WriteFlag(w1, 7, "MY-20", "success", "3", 1)

	// worker-002 flag.md — lower than both
	w2 := filepath.Join(continent, ".expedition", ".run", "worktrees", "worker-002")
	os.MkdirAll(filepath.Join(w2, ".expedition", ".run"), 0755)
	session.WriteFlag(w2, 3, "MY-5", "failed", "7", 0)

	// when
	best := session.ExportReconcileFlags(continent, 2)

	// then — should return worker-001's flag (exp 7)
	if best.LastExpedition != 7 {
		t.Errorf("LastExpedition = %d, want 7", best.LastExpedition)
	}
	if best.LastIssue != "MY-20" {
		t.Errorf("LastIssue = %q, want MY-20", best.LastIssue)
	}
	if best.MidHighSeverity != 1 {
		t.Errorf("MidHighSeverity = %d, want 1", best.MidHighSeverity)
	}
}

func TestReconcileFlags_NoWorktrees(t *testing.T) {
	// given — only continent flag.md, no worktrees
	continent := t.TempDir()
	os.MkdirAll(filepath.Join(continent, ".expedition", ".run"), 0755)
	session.WriteFlag(continent, 5, "MY-10", "success", "5", 0)

	// when
	best := session.ExportReconcileFlags(continent, 2)

	// then — should return continent's flag
	if best.LastExpedition != 5 {
		t.Errorf("LastExpedition = %d, want 5", best.LastExpedition)
	}
	if best.LastIssue != "MY-10" {
		t.Errorf("LastIssue = %q, want MY-10", best.LastIssue)
	}
}

func TestReconcileFlags_NoFlagFiles(t *testing.T) {
	// given — empty directory, no flag.md anywhere
	continent := t.TempDir()

	// when
	best := session.ExportReconcileFlags(continent, 2)

	// then — zero-value defaults
	if best.LastExpedition != 0 {
		t.Errorf("LastExpedition = %d, want 0", best.LastExpedition)
	}
	if best.Remaining != "?" {
		t.Errorf("Remaining = %q, want '?'", best.Remaining)
	}
}

func TestReconcileFlags_Workers0_IgnoresWorktreeFlags(t *testing.T) {
	// given — continent flag (exp 5), stale worktree flag (exp 99)
	continent := t.TempDir()
	os.MkdirAll(filepath.Join(continent, ".expedition", ".run"), 0755)
	session.WriteFlag(continent, 5, "MY-10", "success", "5", 0)

	w1 := filepath.Join(continent, ".expedition", ".run", "worktrees", "worker-001")
	os.MkdirAll(filepath.Join(w1, ".expedition", ".run"), 0755)
	session.WriteFlag(w1, 99, "STALE-1", "success", "0", 0)

	// when — workers=0 means no pool init, stale worktree flags are ignored
	best := session.ExportReconcileFlags(continent, 0)

	// then — only continent flag is considered
	if best.LastExpedition != 5 {
		t.Errorf("LastExpedition = %d, want 5 (should ignore stale worktree exp 99)", best.LastExpedition)
	}
	if best.LastIssue != "MY-10" {
		t.Errorf("LastIssue = %q, want MY-10", best.LastIssue)
	}
}
