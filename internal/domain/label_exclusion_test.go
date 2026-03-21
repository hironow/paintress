package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

// TestExcludeIssuesByLabel_EmptyExcludeList verifies all issues pass through with no exclusions.
func TestExcludeIssuesByLabel_EmptyExcludeList(t *testing.T) {
	// given
	issues := []domain.Issue{
		{ID: "MY-1", Title: "Issue 1", Labels: []string{"bug"}},
		{ID: "MY-2", Title: "Issue 2", Labels: []string{"feature"}},
	}
	excludeLabels := []string{}

	// when
	result := domain.ExcludeIssuesByLabel(issues, excludeLabels)

	// then
	if len(result) != 2 {
		t.Errorf("ExcludeIssuesByLabel with empty exclude = %d issues, want 2", len(result))
	}
}

// TestExcludeIssuesByLabel_ExcludesSingleLabel verifies a single label exclusion.
func TestExcludeIssuesByLabel_ExcludesSingleLabel(t *testing.T) {
	// given
	issues := []domain.Issue{
		{ID: "MY-1", Title: "Blocked issue", Labels: []string{"blocked"}},
		{ID: "MY-2", Title: "Normal issue", Labels: []string{"bug"}},
	}

	// when
	result := domain.ExcludeIssuesByLabel(issues, []string{"blocked"})

	// then: only MY-2 passes through
	if len(result) != 1 {
		t.Fatalf("ExcludeIssuesByLabel = %d issues, want 1", len(result))
	}
	if result[0].ID != "MY-2" {
		t.Errorf("remaining issue ID = %q, want MY-2", result[0].ID)
	}
}

// TestExcludeIssuesByLabel_CaseInsensitive verifies label matching is case-insensitive.
func TestExcludeIssuesByLabel_CaseInsensitive(t *testing.T) {
	// given
	issues := []domain.Issue{
		{ID: "MY-1", Title: "Issue with uppercase label", Labels: []string{"Blocked"}},
		{ID: "MY-2", Title: "Normal issue", Labels: []string{"bug"}},
	}

	// when: exclude "blocked" (lowercase) should match "Blocked" (title case)
	result := domain.ExcludeIssuesByLabel(issues, []string{"blocked"})

	// then
	if len(result) != 1 {
		t.Fatalf("ExcludeIssuesByLabel case-insensitive = %d issues, want 1", len(result))
	}
	if result[0].ID != "MY-2" {
		t.Errorf("remaining issue = %q, want MY-2", result[0].ID)
	}
}

// TestExcludeIssuesByLabel_MultipleExcludeLabels verifies multiple labels can be excluded.
func TestExcludeIssuesByLabel_MultipleExcludeLabels(t *testing.T) {
	// given
	issues := []domain.Issue{
		{ID: "MY-1", Labels: []string{"blocked"}},
		{ID: "MY-2", Labels: []string{"wontfix"}},
		{ID: "MY-3", Labels: []string{"bug"}},
	}

	// when: exclude both "blocked" and "wontfix"
	result := domain.ExcludeIssuesByLabel(issues, []string{"blocked", "wontfix"})

	// then: only MY-3 passes through
	if len(result) != 1 {
		t.Fatalf("ExcludeIssuesByLabel multi-label = %d issues, want 1", len(result))
	}
	if result[0].ID != "MY-3" {
		t.Errorf("remaining issue = %q, want MY-3", result[0].ID)
	}
}

// TestExcludeIssuesByLabel_IssueWithMultipleLabels verifies an issue is excluded
// if any of its labels matches the exclude list.
func TestExcludeIssuesByLabel_IssueWithMultipleLabels(t *testing.T) {
	// given
	issues := []domain.Issue{
		{ID: "MY-1", Labels: []string{"bug", "blocked"}},
		{ID: "MY-2", Labels: []string{"feature"}},
	}

	// when: exclude "blocked" — MY-1 has it plus "bug"
	result := domain.ExcludeIssuesByLabel(issues, []string{"blocked"})

	// then: MY-1 excluded, MY-2 remains
	if len(result) != 1 {
		t.Fatalf("ExcludeIssuesByLabel multi-label issue = %d issues, want 1", len(result))
	}
	if result[0].ID != "MY-2" {
		t.Errorf("remaining issue = %q, want MY-2", result[0].ID)
	}
}

// TestExcludeIssuesByLabel_NoIssuesHaveExcludedLabel verifies all issues pass when none match.
func TestExcludeIssuesByLabel_NoIssuesHaveExcludedLabel(t *testing.T) {
	// given
	issues := []domain.Issue{
		{ID: "MY-1", Labels: []string{"bug"}},
		{ID: "MY-2", Labels: []string{"feature"}},
	}

	// when: exclude a label that no issue has
	result := domain.ExcludeIssuesByLabel(issues, []string{"wontfix"})

	// then: all issues pass through
	if len(result) != 2 {
		t.Errorf("ExcludeIssuesByLabel no matches = %d issues, want 2", len(result))
	}
}

// TestExcludeIssuesByLabel_NilLabels verifies nil excludeLabels returns all issues.
func TestExcludeIssuesByLabel_NilLabels(t *testing.T) {
	// given
	issues := []domain.Issue{
		{ID: "MY-1", Labels: []string{"bug"}},
	}

	// when
	result := domain.ExcludeIssuesByLabel(issues, nil)

	// then
	if len(result) != 1 {
		t.Errorf("ExcludeIssuesByLabel(nil) = %d issues, want 1", len(result))
	}
}
