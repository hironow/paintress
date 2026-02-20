package paintress

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// === Report Parser Tests ===

func TestParseReport_Success(t *testing.T) {
	output := `Some output...
__EXPEDITION_REPORT__
expedition: 3
issue_id: AWE-123
issue_title: Add login form
mission_type: implement
branch: feat/AWE-123-add-login-form
pr_url: https://github.com/org/repo/pull/42
status: success
reason: テスト全件 Pass
remaining_issues: 5
bugs_found: 0
bug_issues: none
__EXPEDITION_END__
Done.`

	report, status := ParseReport(output, 3)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
	}
	if report.IssueID != "AWE-123" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
	if report.Remaining != "5" {
		t.Errorf("Remaining = %q", report.Remaining)
	}
}

func TestParseReport_Complete(t *testing.T) {
	_, status := ParseReport("__EXPEDITION_COMPLETE__", 10)
	if status != StatusComplete {
		t.Fatalf("got %v, want StatusComplete", status)
	}
}

func TestParseReport_Failed(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 2
issue_id: AWE-50
issue_title: Refactor
mission_type: implement
branch: none
pr_url: none
status: failed
reason: テスト失敗
remaining_issues: 8
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`
	_, status := ParseReport(output, 2)
	if status != StatusFailed {
		t.Fatalf("got %v, want StatusFailed", status)
	}
}

func TestParseReport_ParseError(t *testing.T) {
	_, status := ParseReport("no markers here", 1)
	if status != StatusParseError {
		t.Fatalf("got %v, want StatusParseError", status)
	}
}

// === Flag Tests ===

func TestReadWriteFlag(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	WriteFlag(dir, 5, "AWE-42", "success", "3")
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

	report := &ExpeditionReport{
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

// === Gradient Gauge Tests ===

func TestGradient_Charge(t *testing.T) {
	g := NewGradientGauge(5)
	if g.Level() != 0 {
		t.Fatal("should start at 0")
	}

	g.Charge()
	g.Charge()
	g.Charge()
	if g.Level() != 3 {
		t.Errorf("Level = %d, want 3", g.Level())
	}
	if g.IsGradientAttack() {
		t.Error("should not be gradient attack at 3/5")
	}
}

func TestGradient_Full(t *testing.T) {
	g := NewGradientGauge(3)
	g.Charge()
	g.Charge()
	g.Charge()
	if !g.IsGradientAttack() {
		t.Error("should be gradient attack at 3/3")
	}
	// Should not exceed max
	g.Charge()
	if g.Level() != 3 {
		t.Errorf("should cap at max, got %d", g.Level())
	}
}

func TestGradient_Discharge(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Charge()
	g.Charge()
	g.Discharge()
	if g.Level() != 0 {
		t.Errorf("should reset to 0, got %d", g.Level())
	}
}

func TestGradient_Decay(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Charge()
	g.Decay()
	if g.Level() != 1 {
		t.Errorf("Level = %d, want 1", g.Level())
	}
	// Decay at 0 should stay 0
	g.Decay()
	g.Decay()
	if g.Level() != 0 {
		t.Errorf("should not go below 0, got %d", g.Level())
	}
}

func TestGradient_PriorityHint(t *testing.T) {
	g := NewGradientGauge(5)

	hint := g.PriorityHint()
	if !containsStr(hint, "Gauge empty") {
		t.Errorf("at 0, hint should suggest small issues: %s", hint)
	}

	g.Charge()
	hint = g.PriorityHint()
	if !containsStr(hint, "Normal") {
		t.Errorf("at 1, hint should be normal: %s", hint)
	}

	g.Charge()
	g.Charge()
	g.Charge() // level 4
	hint = g.PriorityHint()
	if !containsStr(hint, "Gauge high") {
		t.Errorf("at 4, hint should mention high priority: %s", hint)
	}

	g.Charge() // level 5 = max
	hint = g.PriorityHint()
	if !containsStr(hint, "GRADIENT ATTACK") {
		t.Errorf("at max, hint should say gradient attack: %s", hint)
	}
}

// === Reserve Party Tests ===

func TestReserve_DefaultModel(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet", "haiku"})
	if rp.ActiveModel() != "opus" {
		t.Errorf("ActiveModel = %q, want opus", rp.ActiveModel())
	}
}

func TestReserve_RateLimitSwitch(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet", "haiku"})

	detected := rp.CheckOutput("Error: rate limit exceeded, try again later")
	if !detected {
		t.Error("should detect rate limit")
	}
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should switch to sonnet, got %q", rp.ActiveModel())
	}
}

func TestReserve_NoFalsePositive(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"})
	detected := rp.CheckOutput("The implementation looks correct")
	if detected {
		t.Error("should not detect rate limit in normal output")
	}
	if rp.ActiveModel() != "opus" {
		t.Error("should stay on opus")
	}
}

func TestReserve_NoReserveAvailable(t *testing.T) {
	rp := NewReserveParty("opus", nil) // no reserves
	rp.CheckOutput("rate limit reached")
	// Should stay on opus since no reserve available
	if rp.ActiveModel() != "opus" {
		t.Errorf("should stay opus with no reserves, got %q", rp.ActiveModel())
	}
}

func TestReserve_ForceReserve(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"})
	rp.ForceReserve()
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("got %q, want sonnet", rp.ActiveModel())
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

// === Helpers ===

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
