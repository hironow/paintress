package paintress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// FetchIssues queries Linear GraphQL API and returns issues for the given team.
// endpoint can be overridden for testing; pass LinearAPIEndpoint for production.
// When stateFilter is non-empty, completed/canceled issues are included in the
// GraphQL query so that local filtering can match them.
func FetchIssues(endpoint, apiKey, teamKey, project string, stateFilter []string) ([]Issue, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("LINEAR_API_KEY is required")
	}

	query := `query($filter: IssueFilter) {
		issues(filter: $filter) {
			nodes {
				identifier
				title
				priority
				state { name }
				labels { nodes { name } }
			}
		}
	}`

	filter := map[string]any{
		"team": map[string]any{"key": map[string]any{"eq": teamKey}},
	}
	if len(stateFilter) == 0 {
		filter["state"] = map[string]any{
			"type": map[string]any{"nin": []string{"completed", "canceled"}},
		}
	}
	if project != "" {
		filter["project"] = map[string]any{
			"name": map[string]any{"eq": project},
		}
	}

	reqBody, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]any{"filter": filter},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Linear API error (%d): %s", resp.StatusCode, string(body))
	}

	var gqlResp struct {
		Data struct {
			Issues struct {
				Nodes []struct {
					Identifier string `json:"identifier"`
					Title      string `json:"title"`
					Priority   int    `json:"priority"`
					State      struct {
						Name string `json:"name"`
					} `json:"state"`
					Labels struct {
						Nodes []struct {
							Name string `json:"name"`
						} `json:"nodes"`
					} `json:"labels"`
				} `json:"nodes"`
			} `json:"issues"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		msgs := make([]string, 0, len(gqlResp.Errors))
		for _, e := range gqlResp.Errors {
			msgs = append(msgs, e.Message)
		}
		return nil, fmt.Errorf("GraphQL errors: %s", strings.Join(msgs, "; "))
	}

	issues := make([]Issue, 0, len(gqlResp.Data.Issues.Nodes))
	for _, node := range gqlResp.Data.Issues.Nodes {
		labels := make([]string, 0, len(node.Labels.Nodes))
		for _, l := range node.Labels.Nodes {
			labels = append(labels, l.Name)
		}
		issues = append(issues, Issue{
			ID:       node.Identifier,
			Title:    node.Title,
			Priority: node.Priority,
			Status:   node.State.Name,
			Labels:   labels,
		})
	}

	return issues, nil
}

// FormatIssuesJSONL returns issues as JSONL (one JSON object per line).
func FormatIssuesJSONL(issues []Issue) string {
	if len(issues) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, issue := range issues {
		data, err := json.Marshal(issue)
		if err != nil {
			continue
		}
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.Write(data)
	}
	return sb.String()
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

// FormatIssuesJSON returns issues as a JSON array string.
func FormatIssuesJSON(issues []Issue) (string, error) {
	data, err := json.Marshal(issues)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
