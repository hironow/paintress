package main

import "testing"

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

	report, status := ParseReport(output, 1)
	if status != StatusSkipped {
		t.Fatalf("got %v, want StatusSkipped", status)
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

	report, status := ParseReport(output, 4)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
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

	report, status := ParseReport(output, 7)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
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

	report, status := ParseReport(output, 3)
	if status != StatusFailed {
		t.Fatalf("got %v, want StatusFailed", status)
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

	report, status := ParseReport(output, 2)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
	}
	if report.Insight != "Dev server config: host must be 0.0.0.0: not localhost for Docker" {
		t.Errorf("Insight = %q", report.Insight)
	}
}

func TestParseReport_EmptyOutput(t *testing.T) {
	_, status := ParseReport("", 1)
	if status != StatusParseError {
		t.Fatalf("got %v, want StatusParseError", status)
	}
}

func TestParseReport_OnlyStartMarker(t *testing.T) {
	_, status := ParseReport("__EXPEDITION_REPORT__\nsome data", 1)
	if status != StatusParseError {
		t.Fatalf("got %v, want StatusParseError for missing end marker", status)
	}
}

func TestParseReport_OnlyEndMarker(t *testing.T) {
	_, status := ParseReport("some data\n__EXPEDITION_END__", 1)
	if status != StatusParseError {
		t.Fatalf("got %v, want StatusParseError for missing start marker", status)
	}
}

func TestParseReport_MarkersReversed(t *testing.T) {
	output := "__EXPEDITION_END__\ndata\n__EXPEDITION_REPORT__"
	_, status := ParseReport(output, 1)
	if status != StatusParseError {
		t.Fatalf("got %v, want StatusParseError for reversed markers", status)
	}
}

func TestParseReport_UnknownStatus(t *testing.T) {
	output := `
__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-1
status: unknown_status
__EXPEDITION_END__`

	_, status := ParseReport(output, 1)
	if status != StatusParseError {
		t.Fatalf("got %v, want StatusParseError for unknown status", status)
	}
}

func TestParseReport_CompleteOverridesReport(t *testing.T) {
	// If both COMPLETE and REPORT markers exist, COMPLETE takes precedence
	output := `__EXPEDITION_COMPLETE__
__EXPEDITION_REPORT__
expedition: 1
status: success
__EXPEDITION_END__`

	_, status := ParseReport(output, 1)
	if status != StatusComplete {
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

	report, status := ParseReport(output, 5)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
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

	report, status := ParseReport(output, 1)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
	}
	if report.IssueID != "AWE-1" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
}
