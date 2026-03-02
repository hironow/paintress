package paintress

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// LinearAPIEndpoint is the default Linear GraphQL API URL.
const LinearAPIEndpoint = "https://api.linear.app/graphql"

// Issue represents a Linear issue for pipe-composable output.
type Issue struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Priority int      `json:"priority"`
	Status   string   `json:"status"`
	Labels   []string `json:"labels"`
}

// FormatIssuesJSONL returns issues as JSONL (one JSON object per line).
func FormatIssuesJSONL(issues []Issue) (string, error) {
	if len(issues) == 0 {
		return "", nil
	}
	var sb strings.Builder
	for i, issue := range issues {
		data, err := json.Marshal(issue)
		if err != nil {
			return "", fmt.Errorf("marshal issue %q: %w", issue.ID, err)
		}
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.Write(data)
	}
	return sb.String(), nil
}

// FormatIssuesTable returns issues as a human-readable aligned table.
func FormatIssuesTable(issues []Issue) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%-12s %4s  %-14s  %s", "ID", "PRI", "STATUS", "TITLE")
	for _, issue := range issues {
		fmt.Fprintf(&sb, "\n%-12s %4d  %-14s  %s", issue.ID, issue.Priority, issue.Status, issue.Title)
	}
	return sb.String()
}

// FilterIssuesByState returns issues matching any of the given state names.
// Comparison is case-insensitive. If states is nil or empty, all issues are returned.
func FilterIssuesByState(issues []Issue, states []string) []Issue {
	if len(states) == 0 {
		return issues
	}
	allowed := make(map[string]bool, len(states))
	for _, s := range states {
		allowed[strings.ToLower(s)] = true
	}
	var filtered []Issue
	for _, issue := range issues {
		if allowed[strings.ToLower(issue.Status)] {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// SortByPriority sorts issues in-place by ascending priority.
// Priority 0 (unset) is treated as lowest and sorted to the end.
// The sort is stable, preserving the original order for equal priorities.
func SortByPriority(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		pi, pj := issues[i].Priority, issues[j].Priority
		if pi == 0 {
			return false
		}
		if pj == 0 {
			return true
		}
		return pi < pj
	})
}

// FormatIssuesJSON returns issues as a JSON array string.
func FormatIssuesJSON(issues []Issue) (string, error) {
	data, err := json.Marshal(issues)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
