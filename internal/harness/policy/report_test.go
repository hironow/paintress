package policy_test

import (
	"os"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/policy"
)

func TestParseReport_FailureType_Blocker(t *testing.T) {
	output := `some output
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-99
issue_title: Fix auth
mission_type: implement
branch: feat/auth
pr_url: none
status: failed
reason: dependency not available
failure_type: blocker
remaining_issues: 5
bugs_found: 0
bug_issues: none
insight: External service was down
__EXPEDITION_END__
trailing output`

	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusFailed {
		t.Fatalf("expected domain.StatusFailed, got %v", status)
	}
	if report.FailureType != "blocker" {
		t.Errorf("expected failure_type='blocker', got %q", report.FailureType)
	}
}

func TestParseReport_FailureType_Empty_OnSuccess(t *testing.T) {
	output := `__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-10
issue_title: Add feature
mission_type: implement
branch: feat/add
pr_url: https://github.com/org/repo/pull/1
status: success
reason: implemented
failure_type: none
remaining_issues: 3
bugs_found: 0
bug_issues: none
insight: Clean approach
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusSuccess {
		t.Fatalf("expected domain.StatusSuccess, got %v", status)
	}
	if report.FailureType != "none" {
		t.Errorf("expected failure_type='none', got %q", report.FailureType)
	}
}

func TestParseReport_SkippedStatus(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-77
issue_title: Cleanup old API
mission_type: implement
branch: none
pr_url: none
status: skipped
reason: 複雑すぎるためスキップ
remaining_issues: 10
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusSkipped {
		t.Fatalf("got %v, want domain.StatusSkipped", status)
	}
	if report.IssueID != "AWE-77" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
	if report.Reason != "複雑すぎるためスキップ" {
		t.Errorf("Reason = %q", report.Reason)
	}
}

func TestParseReport_BugsFound(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 4
issue_id: AWE-88
issue_title: Verify login
mission_type: verify
branch: feat/AWE-88
pr_url: none
status: success
reason: verified
remaining_issues: 3
bugs_found: 2
bug_issues: AWE-89,AWE-90
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 4)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.BugsFound != 2 {
		t.Errorf("BugsFound = %d, want 2", report.BugsFound)
	}
	if report.BugIssues != "AWE-89,AWE-90" {
		t.Errorf("BugIssues = %q", report.BugIssues)
	}
	if report.MissionType != "verify" {
		t.Errorf("MissionType = %q", report.MissionType)
	}
}

func TestParseReport_AllFields(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 7
issue_id: AWE-100
issue_title: Add user settings page
mission_type: implement
branch: feat/AWE-100-user-settings
pr_url: https://github.com/org/repo/pull/55
status: success
reason: テスト全件 Pass、PR 作成済み
remaining_issues: 2
bugs_found: 0
bug_issues: none
insight: This project uses Tailwind CSS v4 with @apply directives in module CSS files
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 7)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}

	if report.Expedition != 7 {
		t.Errorf("Expedition = %d", report.Expedition)
	}
	if report.IssueID != "AWE-100" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
	if report.IssueTitle != "Add user settings page" {
		t.Errorf("IssueTitle = %q", report.IssueTitle)
	}
	if report.MissionType != "implement" {
		t.Errorf("MissionType = %q", report.MissionType)
	}
	if report.Branch != "feat/AWE-100-user-settings" {
		t.Errorf("Branch = %q", report.Branch)
	}
	if report.PRUrl != "https://github.com/org/repo/pull/55" {
		t.Errorf("PRUrl = %q", report.PRUrl)
	}
	if report.Remaining != "2" {
		t.Errorf("Remaining = %q", report.Remaining)
	}
	if report.Insight != "This project uses Tailwind CSS v4 with @apply directives in module CSS files" {
		t.Errorf("Insight = %q", report.Insight)
	}
}

func TestParseReport_WithInsight(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 3
issue_id: AWE-50
issue_title: Fix auth flow
mission_type: fix
branch: fix/AWE-50-auth
pr_url: none
status: failed
reason: test timeout
remaining_issues: 8
bugs_found: 0
bug_issues: none
insight: The auth module requires Redis connection and tests need REDIS_URL env var set
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 3)
	if status != domain.StatusFailed {
		t.Fatalf("got %v, want domain.StatusFailed", status)
	}
	if report.Insight != "The auth module requires Redis connection and tests need REDIS_URL env var set" {
		t.Errorf("Insight = %q", report.Insight)
	}
}

func TestParseReport_InsightWithColons(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 2
issue_id: AWE-30
issue_title: Setup CI
mission_type: implement
branch: feat/AWE-30-ci
pr_url: https://github.com/org/repo/pull/10
status: success
reason: done
remaining_issues: 5
bugs_found: 0
bug_issues: none
insight: Dev server config: host must be 0.0.0.0: not localhost for Docker
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 2)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.Insight != "Dev server config: host must be 0.0.0.0: not localhost for Docker" {
		t.Errorf("Insight = %q", report.Insight)
	}
}

func TestParseReport_EmptyOutput(t *testing.T) {
	_, status := domain.ParseReport("", 1)
	if status != domain.StatusParseError {
		t.Fatalf("got %v, want domain.StatusParseError", status)
	}
}

func TestParseReport_OnlyStartMarker(t *testing.T) {
	_, status := domain.ParseReport("__EXPEDITION_REPORT__\nsome data", 1)
	if status != domain.StatusParseError {
		t.Fatalf("got %v, want domain.StatusParseError for missing end marker", status)
	}
}

func TestParseReport_OnlyEndMarker(t *testing.T) {
	_, status := domain.ParseReport("some data\n__EXPEDITION_END__", 1)
	if status != domain.StatusParseError {
		t.Fatalf("got %v, want domain.StatusParseError for missing start marker", status)
	}
}

func TestParseReport_MarkersReversed(t *testing.T) {
	output := "__EXPEDITION_END__\ndata\n__EXPEDITION_REPORT__"
	_, status := domain.ParseReport(output, 1)
	if status != domain.StatusParseError {
		t.Fatalf("got %v, want domain.StatusParseError for reversed markers", status)
	}
}

func TestParseReport_UnknownStatus(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
status: unknown_status
__EXPEDITION_END__`

	_, status := domain.ParseReport(output, 1)
	if status != domain.StatusParseError {
		t.Fatalf("got %v, want domain.StatusParseError for unknown status", status)
	}
}

func TestParseReport_CompleteOverridesReport(t *testing.T) {
	// If both COMPLETE and REPORT markers exist, COMPLETE takes precedence
	output := `__EXPEDITION_COMPLETE__
__EXPEDITION_REPORT__
expedition: 1
status: success
__EXPEDITION_END__`

	_, status := domain.ParseReport(output, 1)
	if status != domain.StatusComplete {
		t.Fatalf("COMPLETE should take precedence, got %v", status)
	}
}

func TestParseReport_WhitespaceInValues(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 5
issue_id:   AWE-55
issue_title:  Some Title With Spaces
status: success
remaining_issues: 0
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 5)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.IssueID != "AWE-55" {
		t.Errorf("IssueID = %q, should be trimmed", report.IssueID)
	}
}

func TestParseReport_SurroundedByExtraOutput(t *testing.T) {
	output := `Starting expedition...
Analyzing codebase...
Running tests...
All 42 tests passed!

Creating PR...

__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
issue_title: Test
mission_type: implement
branch: feat/AWE-1
pr_url: https://github.com/org/repo/pull/1
status: success
reason: done
remaining_issues: 5
bugs_found: 0
bug_issues: none
__EXPEDITION_END__

Session ended.
Goodbye.`

	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.IssueID != "AWE-1" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
}

func TestParseReport_MissingOptionalFields_Defaults(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 9
issue_id: AWE-9
issue_title: Minimal report
mission_type: implement
branch: feat/AWE-9
pr_url: none
status: success
reason: ok
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 9)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.BugsFound != 0 {
		t.Errorf("BugsFound = %d, want 0 when missing", report.BugsFound)
	}
	if report.BugIssues != "" {
		t.Errorf("BugIssues = %q, want empty when missing", report.BugIssues)
	}
	if report.Remaining != "" {
		t.Errorf("Remaining = %q, want empty when missing", report.Remaining)
	}
}

func TestParseReport_BugsFoundWithoutBugIssues(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 10
issue_id: AWE-10
issue_title: Verify audit trail
mission_type: verify
branch: feat/AWE-10
pr_url: none
status: success
reason: ok
bugs_found: 3
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 10)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.BugsFound != 3 {
		t.Errorf("BugsFound = %d, want 3", report.BugsFound)
	}
	if report.BugIssues != "" {
		t.Errorf("BugIssues = %q, want empty when missing", report.BugIssues)
	}
}

func TestParseReport_IgnoresUnknownFields(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 11
issue_id: AWE-11
issue_title: Unknown fields
mission_type: implement
branch: feat/AWE-11
pr_url: none
status: success
reason: ok
unknown_field: should be ignored
another_unknown: 123
remaining_issues: 4
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 11)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.IssueID != "AWE-11" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
	if report.Remaining != "4" {
		t.Errorf("Remaining = %q", report.Remaining)
	}
}

func TestParseReport_DuplicateKeys_LastWins(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 12
issue_id: AWE-12
issue_title: Duplicate keys
mission_type: implement
branch: feat/AWE-12
pr_url: none
status: success
reason: first reason
reason: final reason
remaining_issues: 9
remaining_issues: 8
bugs_found: 1
bugs_found: 2
bug_issues: AWE-100
bug_issues: AWE-101
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 12)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.Reason != "final reason" {
		t.Errorf("Reason = %q, want %q", report.Reason, "final reason")
	}
	if report.Remaining != "8" {
		t.Errorf("Remaining = %q, want %q", report.Remaining, "8")
	}
	if report.BugsFound != 2 {
		t.Errorf("BugsFound = %d, want 2", report.BugsFound)
	}
	if report.BugIssues != "AWE-101" {
		t.Errorf("BugIssues = %q, want %q", report.BugIssues, "AWE-101")
	}
}

func TestParseReport_MultipleBlocks_FirstInvalid(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
status: unknown
__EXPEDITION_END__
__EXPEDITION_REPORT__
expedition: 2
issue_id: AWE-2
status: success
remaining_issues: 3
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`

	_, status := domain.ParseReport(output, 1)
	if status != domain.StatusParseError {
		t.Fatalf("first invalid block should yield parse error, got %v", status)
	}
}

func TestParseReport_MultipleBlocks_FirstSuccessWins(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
status: success
remaining_issues: 5
bugs_found: 0
bug_issues: none
__EXPEDITION_END__
__EXPEDITION_REPORT__
expedition: 2
issue_id: AWE-2
status: failed
reason: later failure
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.IssueID != "AWE-1" {
		t.Errorf("IssueID = %q, want %q", report.IssueID, "AWE-1")
	}
	if report.Remaining != "5" {
		t.Errorf("Remaining = %q, want %q", report.Remaining, "5")
	}
}

// --- from ralph_test.go ---

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

	report, status := domain.ParseReport(output, 3)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.IssueID != "AWE-123" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
	if report.Remaining != "5" {
		t.Errorf("Remaining = %q", report.Remaining)
	}
}

func TestParseReport_Complete(t *testing.T) {
	_, status := domain.ParseReport("__EXPEDITION_COMPLETE__", 10)
	if status != domain.StatusComplete {
		t.Fatalf("got %v, want domain.StatusComplete", status)
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
	_, status := domain.ParseReport(output, 2)
	if status != domain.StatusFailed {
		t.Fatalf("got %v, want domain.StatusFailed", status)
	}
}

func TestParseReport_ParseError(t *testing.T) {
	_, status := domain.ParseReport("no markers here", 1)
	if status != domain.StatusParseError {
		t.Fatalf("got %v, want domain.StatusParseError", status)
	}
}

// --- from edge_cases_test.go ---

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

	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.IssueID != "FIRST" {
		t.Errorf("should parse first report, got IssueID=%q", report.IssueID)
	}
}

func TestParseReport_EmptyBlock(t *testing.T) {
	output := "__EXPEDITION_REPORT__\n__EXPEDITION_END__"
	_, status := domain.ParseReport(output, 1)
	// Empty block has no status field -> ParseError
	if status != domain.StatusParseError {
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

	report, _ := domain.ParseReport(output, 1)
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

	report, _ := domain.ParseReport(output, 1)
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

	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
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
	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
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

	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusFailed {
		t.Fatalf("got %v", status)
	}
	if report.Reason != "error: timeout: connection refused: port 5432" {
		t.Errorf("Reason should preserve all colons: %q", report.Reason)
	}
}

func TestParseReport_ExpNumZero(t *testing.T) {
	output := "__EXPEDITION_COMPLETE__"
	report, status := domain.ParseReport(output, 0)
	if status != domain.StatusComplete {
		t.Fatalf("got %v", status)
	}
	if report.Expedition != 0 {
		t.Errorf("Expedition = %d, want 0", report.Expedition)
	}
}

func TestFormatLuminaForPrompt_SingleLumina(t *testing.T) {
	luminas := []domain.Lumina{
		{Pattern: "only one pattern", Source: "failure-pattern", Uses: 1},
	}
	result := policy.FormatLuminaForPrompt(luminas)
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

func TestExtractPRURLs_FiltersNone(t *testing.T) {
	// given
	reports := []*domain.ExpeditionReport{
		{Expedition: 1, IssueID: "AWE-1", PRUrl: "https://github.com/org/repo/pull/1"},
		{Expedition: 2, IssueID: "AWE-2", PRUrl: "none"},
		{Expedition: 3, IssueID: "AWE-3", PRUrl: ""},
		{Expedition: 4, IssueID: "AWE-4", PRUrl: "https://github.com/org/repo/pull/4"},
	}

	// when
	entries := domain.ExtractPRURLs(reports)

	// then
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].PRUrl != "https://github.com/org/repo/pull/1" {
		t.Errorf("entry[0].PRUrl = %q", entries[0].PRUrl)
	}
	if entries[1].Expedition != 4 {
		t.Errorf("entry[1].Expedition = %d, want 4", entries[1].Expedition)
	}
}

func TestExtractPRURLs_Empty(t *testing.T) {
	entries := domain.ExtractPRURLs(nil)
	if len(entries) != 0 {
		t.Errorf("expected 0, got %d", len(entries))
	}
}

func TestExtractPRURLs_NilReport(t *testing.T) {
	reports := []*domain.ExpeditionReport{nil, nil}
	entries := domain.ExtractPRURLs(reports)
	if len(entries) != 0 {
		t.Errorf("expected 0, got %d", len(entries))
	}
}

func TestParseReport_AlternativeEndMarker(t *testing.T) {
	// LLMs sometimes hallucinate __END_EXPEDITION_REPORT__ instead of __EXPEDITION_END__.
	// The parser must accept both.
	output := `Starting...
__EXPEDITION_REPORT__
expedition: 60
issue_id: MY-418
issue_title: リンク作成ワークフロー
mission_type: verify
branch: none
pr_url: none
status: success
reason: 全DoD検証Pass
remaining_issues: 4
bugs_found: 0
bug_issues: none
insight: Playwright CLIでKonva Canvas検証が可能
__END_EXPEDITION_REPORT__
Done.`

	report, status := domain.ParseReport(output, 60)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess — alt end marker should be accepted", status)
	}
	if report.IssueID != "MY-418" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
	if report.MissionType != "verify" {
		t.Errorf("MissionType = %q", report.MissionType)
	}
	if report.Insight != "Playwright CLIでKonva Canvas検証が可能" {
		t.Errorf("Insight = %q", report.Insight)
	}
}

func TestParseReport_CanonicalEndMarkerStillWorks(t *testing.T) {
	// Ensure canonical __EXPEDITION_END__ still works after adding alt marker.
	output := `__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
status: success
__EXPEDITION_END__`

	_, status := domain.ParseReport(output, 1)
	if status != domain.StatusSuccess {
		t.Fatalf("canonical end marker broken, got %v", status)
	}
}

func TestParseReport_BothEndMarkersPresent_CanonicalWins(t *testing.T) {
	// When both markers exist, the earlier one (canonical) is used.
	output := `__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
status: success
__EXPEDITION_END__
extra stuff
__END_EXPEDITION_REPORT__`

	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v", status)
	}
	if report.IssueID != "AWE-1" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
}

func TestParseReport_Golden_Expedition060(t *testing.T) {
	// Golden file: real expedition output with __END_EXPEDITION_REPORT__ and
	// YAML-like list syntax. Verifies parser handles messy real-world output.
	data, err := os.ReadFile("testdata/golden/expedition-060-alt-end-marker.txt")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	report, status := domain.ParseReport(string(data), 60)
	if status != domain.StatusSuccess {
		t.Fatalf("golden file should parse as success, got %v", status)
	}
	if report.IssueID != "" {
		// Note: the YAML format uses "target_issue" not "issue_id",
		// so issue_id won't be extracted — but status IS extracted.
		t.Logf("IssueID extracted: %q (field mismatch expected)", report.IssueID)
	}
	if report.Status != "success" {
		t.Errorf("Status = %q, want 'success'", report.Status)
	}
	if report.BugsFound != 0 {
		t.Errorf("BugsFound = %d, want 0", report.BugsFound)
	}
}

func TestParseReport_ConsolidateMissionType(t *testing.T) {
	output := `Starting consolidation...
__EXPEDITION_REPORT__
expedition: 70
issue_id: none
issue_title: PR chain consolidation
mission_type: consolidate
branch: consolidate/chain-a
pr_url: https://github.com/org/repo/pull/99
status: success
reason: Applied #1, #2, #3. New PR #99. Closed #1, #2, #3.
failure_type: none
remaining_issues: 5
bugs_found: 0
bug_issues: none
insight: Cherry-pick ordering matters for stacked PRs - always apply root-first
__EXPEDITION_END__
Done.`

	report, status := domain.ParseReport(output, 70)
	if status != domain.StatusSuccess {
		t.Fatalf("got %v, want domain.StatusSuccess", status)
	}
	if report.MissionType != "consolidate" {
		t.Errorf("MissionType = %q, want 'consolidate'", report.MissionType)
	}
	if report.Branch != "consolidate/chain-a" {
		t.Errorf("Branch = %q", report.Branch)
	}
	if report.PRUrl != "https://github.com/org/repo/pull/99" {
		t.Errorf("PRUrl = %q", report.PRUrl)
	}
	if !containsStr(report.Reason, "Applied #1, #2, #3") {
		t.Errorf("Reason should contain applied PRs: %q", report.Reason)
	}
}

func TestParseReport_ConsolidatePartialCompletion(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 71
issue_id: none
issue_title: PR chain consolidation (partial)
mission_type: consolidate
branch: consolidate/chain-a
pr_url: none
status: skipped
reason: Applied #1, #2, #3. Remaining: #4, #5. Timed out before completion.
failure_type: performance
remaining_issues: 5
bugs_found: 0
bug_issues: none
insight: Large chains need multiple expeditions
__EXPEDITION_END__`

	report, status := domain.ParseReport(output, 71)
	if status != domain.StatusSkipped {
		t.Fatalf("got %v, want domain.StatusSkipped for partial consolidation", status)
	}
	if report.MissionType != "consolidate" {
		t.Errorf("MissionType = %q, want 'consolidate'", report.MissionType)
	}
	if !containsStr(report.Reason, "Remaining: #4, #5") {
		t.Errorf("Reason should contain remaining PRs: %q", report.Reason)
	}
	if report.FailureType != "performance" {
		t.Errorf("FailureType = %q, want 'performance' for timeout", report.FailureType)
	}
}
