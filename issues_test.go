package paintress

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatIssuesJSONL(t *testing.T) {
	// given
	issues := []Issue{
		{ID: "MY-42", Title: "Fix login bug", Priority: 2, Status: "Todo", Labels: []string{"Bug"}},
		{ID: "MY-43", Title: "Add dark mode", Priority: 3, Status: "In Progress", Labels: []string{"Enhancement"}},
	}

	// when
	out, err := FormatIssuesJSONL(issues)
	if err != nil {
		t.Fatalf("FormatIssuesJSONL: %v", err)
	}

	// then — each line must be valid JSON
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), out)
	}
	for i, line := range lines {
		var parsed Issue
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Fatalf("line %d is not valid JSON: %v\nraw: %s", i, err, line)
		}
		if parsed.ID != issues[i].ID {
			t.Errorf("line %d: id = %q, want %q", i, parsed.ID, issues[i].ID)
		}
	}
}

func TestFormatIssuesJSON(t *testing.T) {
	// given
	issues := []Issue{
		{ID: "MY-42", Title: "Fix login bug", Priority: 2, Status: "Todo", Labels: []string{"Bug"}},
	}

	// when
	out, err := FormatIssuesJSON(issues)
	if err != nil {
		t.Fatalf("FormatIssuesJSON: %v", err)
	}

	// then — must be valid JSON array
	var parsed []Issue
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON array: %v\nraw: %s", err, out)
	}
	if len(parsed) != 1 {
		t.Errorf("expected 1 issue, got %d", len(parsed))
	}
}

func TestFormatIssuesTable(t *testing.T) {
	// given
	issues := []Issue{
		{ID: "MY-42", Title: "Fix login bug", Priority: 2, Status: "Todo", Labels: []string{"Bug"}},
		{ID: "MY-43", Title: "Add dark mode", Priority: 3, Status: "In Progress", Labels: []string{"Enhancement"}},
	}

	// when
	out := FormatIssuesTable(issues)

	// then — must contain header and both issue rows
	if !strings.Contains(out, "ID") {
		t.Error("table should contain ID header")
	}
	if !strings.Contains(out, "MY-42") {
		t.Error("table should contain MY-42")
	}
	if !strings.Contains(out, "MY-43") {
		t.Error("table should contain MY-43")
	}
	if !strings.Contains(out, "Fix login bug") {
		t.Error("table should contain issue title")
	}
	if !strings.Contains(out, "Todo") {
		t.Error("table should contain status")
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (header + 2 issues), got %d", len(lines))
	}
}

func TestFormatIssuesTable_Empty(t *testing.T) {
	// given
	issues := []Issue{}

	// when
	out := FormatIssuesTable(issues)

	// then — header only
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (header only), got %d", len(lines))
	}
}

func TestFilterIssuesByState(t *testing.T) {
	// given
	issues := []Issue{
		{ID: "MY-1", Status: "Todo"},
		{ID: "MY-2", Status: "In Progress"},
		{ID: "MY-3", Status: "Done"},
		{ID: "MY-4", Status: "Todo"},
	}

	// when — filter to Todo only
	filtered := FilterIssuesByState(issues, []string{"Todo"})

	// then
	if len(filtered) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(filtered))
	}
	if filtered[0].ID != "MY-1" || filtered[1].ID != "MY-4" {
		t.Errorf("filtered = %v, want MY-1 and MY-4", filtered)
	}
}

func TestFilterIssuesByState_MultipleStates(t *testing.T) {
	// given
	issues := []Issue{
		{ID: "MY-1", Status: "Todo"},
		{ID: "MY-2", Status: "In Progress"},
		{ID: "MY-3", Status: "Done"},
	}

	// when — filter to Todo and In Progress
	filtered := FilterIssuesByState(issues, []string{"Todo", "In Progress"})

	// then
	if len(filtered) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(filtered))
	}
}

func TestFilterIssuesByState_CaseInsensitive(t *testing.T) {
	// given
	issues := []Issue{
		{ID: "MY-1", Status: "Todo"},
		{ID: "MY-2", Status: "In Progress"},
	}

	// when — lowercase input
	filtered := FilterIssuesByState(issues, []string{"todo"})

	// then
	if len(filtered) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(filtered))
	}
	if filtered[0].ID != "MY-1" {
		t.Errorf("filtered[0].ID = %q, want MY-1", filtered[0].ID)
	}
}

func TestFilterIssuesByState_EmptyFilter(t *testing.T) {
	// given
	issues := []Issue{
		{ID: "MY-1", Status: "Todo"},
		{ID: "MY-2", Status: "Done"},
	}

	// when — empty filter returns all
	filtered := FilterIssuesByState(issues, nil)

	// then
	if len(filtered) != 2 {
		t.Fatalf("expected 2 issues (no filter), got %d", len(filtered))
	}
}

func TestFormatIssuesJSONL_EmptySlice(t *testing.T) {
	// given
	issues := []Issue{}

	// when
	out, err := FormatIssuesJSONL(issues)
	if err != nil {
		t.Fatalf("FormatIssuesJSONL: %v", err)
	}

	// then
	if out != "" {
		t.Errorf("expected empty string for empty issues, got %q", out)
	}
}
