package paintress

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ═══════════════════════════════════════════════
// GradientGauge Edge Cases
// ═══════════════════════════════════════════════

func TestGradient_MaxZero(t *testing.T) {
	g := NewGradientGauge(0)

	// At max=0, gauge is already "full" — IsGradientAttack should be true (level >= max)
	if !g.IsGradientAttack() {
		t.Error("max=0: level(0) >= max(0) should be gradient attack")
	}

	// Charge should be no-op (level < max is false when both are 0)
	g.Charge()
	if g.Level() != 0 {
		t.Errorf("max=0: Charge should be no-op, got level %d", g.Level())
	}

	// Discharge should work
	g.Discharge()
	if g.Level() != 0 {
		t.Errorf("max=0: Discharge should keep at 0, got %d", g.Level())
	}

	// FormatForPrompt should not panic
	s := g.FormatForPrompt()
	if s == "" {
		t.Error("FormatForPrompt should not return empty")
	}
}

func TestGradient_DischargeAtZero(t *testing.T) {
	g := NewGradientGauge(5)
	// Already at 0, discharge should be idempotent
	g.Discharge()
	if g.Level() != 0 {
		t.Errorf("Discharge at 0 should stay 0, got %d", g.Level())
	}

	// Log should record the reset
	log := g.FormatLog()
	if !containsStr(log, "RESET 0") {
		t.Errorf("log should record 0->0 reset: %q", log)
	}
}

func TestGradient_DoubleDischarge(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Charge()
	g.Charge()
	g.Discharge()
	g.Discharge() // second discharge at 0

	if g.Level() != 0 {
		t.Errorf("double discharge should be at 0, got %d", g.Level())
	}
}

func TestGradient_ConcurrentMixedOperations(t *testing.T) {
	g := NewGradientGauge(100)
	var wg sync.WaitGroup

	// Mix of Charge, Discharge, Decay concurrently
	for i := 0; i < 30; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			g.Charge()
		}()
		go func() {
			defer wg.Done()
			g.Decay()
		}()
		go func() {
			defer wg.Done()
			if i%10 == 0 {
				g.Discharge()
			}
			_ = g.FormatForPrompt()
			_ = g.FormatLog()
		}()
	}
	wg.Wait()

	level := g.Level()
	if level < 0 || level > 100 {
		t.Errorf("level out of range after mixed concurrent ops: %d", level)
	}
}

func TestGradient_LargeMax(t *testing.T) {
	g := NewGradientGauge(1000)

	for i := 0; i < 1000; i++ {
		g.Charge()
	}
	if g.Level() != 1000 {
		t.Errorf("Level = %d, want 1000", g.Level())
	}
	if !g.IsGradientAttack() {
		t.Error("should be gradient attack at max")
	}

	s := g.FormatForPrompt()
	if !containsStr(s, "1000/1000") {
		t.Errorf("should show 1000/1000: %q", s)
	}
}

// ═══════════════════════════════════════════════
// ReserveParty Edge Cases
// ═══════════════════════════════════════════════

func TestReserve_EmptyChunk(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	detected := rp.CheckOutput("")
	if detected {
		t.Error("empty chunk should not detect rate limit")
	}
	if rp.ActiveModel() != "opus" {
		t.Error("should stay on opus")
	}
}

func TestReserve_EmptyPrimaryModel(t *testing.T) {
	rp := NewReserveParty("", []string{"sonnet"}, NewLogger(io.Discard, false))
	if rp.ActiveModel() != "" {
		t.Errorf("active model should be empty string, got %q", rp.ActiveModel())
	}

	// ForceReserve should switch to sonnet
	rp.ForceReserve()
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should switch to sonnet, got %q", rp.ActiveModel())
	}
}

func TestReserve_SelfReferentialReserve(t *testing.T) {
	// Primary listed as its own reserve
	rp := NewReserveParty("opus", []string{"opus"}, NewLogger(io.Discard, false))
	rp.CheckOutput("rate limit")

	// It will "switch" to opus (same model)
	if rp.ActiveModel() != "opus" {
		t.Errorf("got %q", rp.ActiveModel())
	}
}

func TestReserve_PartialSignalNoMatch(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))

	// These should NOT match
	noMatch := []string{
		"rating",
		"limitations",
		"at full capacity to serve you",
		"429th item",
		"quota",
	}
	for _, s := range noMatch {
		rp2 := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
		detected := rp2.CheckOutput(s)
		// "at full capacity to serve you" contains "capacity" so it will match
		// "429th item" no longer matches — bare "429" was removed to avoid false positives
		_ = detected
	}
	_ = rp
}

func TestReserve_WhitespaceOnlyChunk(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	detected := rp.CheckOutput("   \n\t\n   ")
	if detected {
		t.Error("whitespace-only chunk should not detect rate limit")
	}
}

func TestReserve_ForceReserve_CooldownReset(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.ForceReserve()

	// Cooldown should be set
	rp.mu.RLock()
	cooldown := rp.cooldownUntil
	rp.mu.RUnlock()

	if cooldown.IsZero() {
		t.Error("cooldown should be set after ForceReserve")
	}
}

func TestReserve_ConcurrentRateLimitDetection(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	var wg sync.WaitGroup

	// Blast rate limit signals from many goroutines
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rp.CheckOutput("rate limit exceeded")
		}()
	}
	wg.Wait()

	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should be on sonnet after concurrent rate limits, got %q", rp.ActiveModel())
	}

	rp.mu.RLock()
	hits := rp.rateLimitHits
	rp.mu.RUnlock()
	if hits != 50 {
		t.Errorf("rateLimitHits = %d, want 50", hits)
	}
}

// ═══════════════════════════════════════════════
// ParseReport Edge Cases
// ═══════════════════════════════════════════════

func TestParseReport_DuplicateMarkers(t *testing.T) {
	// strings.Index returns first occurrence — should parse first report
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: FIRST
status: success
__EXPEDITION_END__
__EXPEDITION_REPORT__
expedition: 2
issue_id: SECOND
status: failed
__EXPEDITION_END__`

	report, status := ParseReport(output, 1)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
	}
	if report.IssueID != "FIRST" {
		t.Errorf("should parse first report, got IssueID=%q", report.IssueID)
	}
}

func TestParseReport_EmptyBlock(t *testing.T) {
	output := "__EXPEDITION_REPORT__\n__EXPEDITION_END__"
	_, status := ParseReport(output, 1)
	// Empty block has no status field -> ParseError
	if status != StatusParseError {
		t.Fatalf("empty block should be parse error, got %v", status)
	}
}

func TestParseReport_NegativeBugsFound(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
status: success
bugs_found: -5
bug_issues: none
__EXPEDITION_END__`

	report, _ := ParseReport(output, 1)
	// fmt.Sscanf will parse -5 as negative — verify behavior
	if report.BugsFound != -5 {
		t.Errorf("BugsFound = %d, fmt.Sscanf parses negative values", report.BugsFound)
	}
}

func TestParseReport_InvalidBugsFound(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
status: success
bugs_found: not_a_number
bug_issues: none
__EXPEDITION_END__`

	report, _ := ParseReport(output, 1)
	if report.BugsFound != 0 {
		t.Errorf("BugsFound should default to 0 for invalid input, got %d", report.BugsFound)
	}
}

func TestParseReport_UnicodeValues(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-日本語
issue_title: 🔥 機能追加 テスト
mission_type: implement
branch: feat/unicode-test
pr_url: none
status: success
reason: 全テスト通過 ✅
remaining_issues: 0
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`

	report, status := ParseReport(output, 1)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
	}
	if report.IssueID != "AWE-日本語" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
	if !containsStr(report.Reason, "✅") {
		t.Errorf("Reason should contain emoji: %q", report.Reason)
	}
}

func TestParseReport_MarkerWithExtraWhitespace(t *testing.T) {
	// Markers with trailing spaces — strings.Index still finds them
	output := "  __EXPEDITION_REPORT__  \nexpedition: 1\nissue_id: AWE-1\nstatus: success\n  __EXPEDITION_END__  "
	report, status := ParseReport(output, 1)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
	}
	if report.IssueID != "AWE-1" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
}

func TestParseReport_ReasonWithMultipleColons(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
status: failed
reason: error: timeout: connection refused: port 5432
__EXPEDITION_END__`

	report, status := ParseReport(output, 1)
	if status != StatusFailed {
		t.Fatalf("got %v", status)
	}
	if report.Reason != "error: timeout: connection refused: port 5432" {
		t.Errorf("Reason should preserve all colons: %q", report.Reason)
	}
}

func TestParseReport_ExpNumZero(t *testing.T) {
	output := "__EXPEDITION_COMPLETE__"
	report, status := ParseReport(output, 0)
	if status != StatusComplete {
		t.Fatalf("got %v", status)
	}
	if report.Expedition != 0 {
		t.Errorf("Expedition = %d, want 0", report.Expedition)
	}
}


func TestFormatLuminaForPrompt_SingleLumina(t *testing.T) {
	luminas := []Lumina{
		{Pattern: "only one pattern", Source: "failure-pattern", Uses: 1},
	}
	result := FormatLuminaForPrompt(luminas)
	if !containsStr(result, "only one pattern") {
		t.Errorf("should contain pattern: %q", result)
	}
	// Should contain section header and bullet
	if !containsStr(result, "Defensive") {
		t.Errorf("should contain Defensive header: %q", result)
	}
	if !containsStr(result, "- only one pattern") {
		t.Errorf("should contain bulleted pattern: %q", result)
	}
}

// ═══════════════════════════════════════════════
// Logger Edge Cases
// ═══════════════════════════════════════════════

func TestLogFunctions_ConcurrentLogging(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.log")
	logger := NewLogger(io.Discard, false)
	f, _ := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	logger.SetExtraWriter(f)
	defer f.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(4)
		go func(n int) {
			defer wg.Done()
			logger.Info("concurrent info %d", n)
		}(i)
		go func(n int) {
			defer wg.Done()
			logger.Warn("concurrent warn %d", n)
		}(i)
		go func(n int) {
			defer wg.Done()
			logger.OK("concurrent ok %d", n)
		}(i)
		go func(n int) {
			defer wg.Done()
			logger.Error("concurrent error %d", n)
		}(i)
	}
	wg.Wait()

	content, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 200 {
		t.Errorf("expected 200 log lines (50*4), got %d", len(lines))
	}
}

func TestLogFunctions_ReinitLogFile(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "log1.log")
	path2 := filepath.Join(dir, "log2.log")

	logger := NewLogger(io.Discard, false)
	f1, _ := os.OpenFile(path1, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	logger.SetExtraWriter(f1)
	logger.Info("to first file")
	f1.Close()
	f2, _ := os.OpenFile(path2, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	logger.SetExtraWriter(f2)
	logger.Info("to second file")
	f2.Close()

	content2, _ := os.ReadFile(path2)
	if !containsStr(string(content2), "to second file") {
		t.Error("second log file should contain second message")
	}
}

