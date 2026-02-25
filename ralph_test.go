package paintress

import (
	"io"
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
	rp := NewReserveParty("opus", []string{"sonnet", "haiku"}, NewLogger(io.Discard, false))
	if rp.ActiveModel() != "opus" {
		t.Errorf("ActiveModel = %q, want opus", rp.ActiveModel())
	}
}

func TestReserve_RateLimitSwitch(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet", "haiku"}, NewLogger(io.Discard, false))

	detected := rp.CheckOutput("Error: rate limit exceeded, try again later")
	if !detected {
		t.Error("should detect rate limit")
	}
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should switch to sonnet, got %q", rp.ActiveModel())
	}
}

func TestReserve_NoFalsePositive(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	detected := rp.CheckOutput("The implementation looks correct")
	if detected {
		t.Error("should not detect rate limit in normal output")
	}
	if rp.ActiveModel() != "opus" {
		t.Error("should stay on opus")
	}
}

func TestReserve_NoReserveAvailable(t *testing.T) {
	rp := NewReserveParty("opus", nil, NewLogger(io.Discard, false)) // no reserves
	rp.CheckOutput("rate limit reached")
	// Should stay on opus since no reserve available
	if rp.ActiveModel() != "opus" {
		t.Errorf("should stay opus with no reserves, got %q", rp.ActiveModel())
	}
}

func TestReserve_ForceReserve(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.ForceReserve()
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("got %q, want sonnet", rp.ActiveModel())
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
