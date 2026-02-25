package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hironow/paintress"
)

// FetchIssues queries Linear GraphQL API and returns issues for the given team.
// endpoint can be overridden for testing; pass paintress.LinearAPIEndpoint for production.
// When stateFilter is non-empty, completed/canceled issues are included in the
// GraphQL query so that local filtering can match them.
func FetchIssues(ctx context.Context, endpoint, apiKey, teamKey, project string, stateFilter []string) ([]paintress.Issue, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("LINEAR_API_KEY is required")
	}

	query := `query($filter: IssueFilter, $first: Int, $after: String) {
		issues(filter: $filter, first: $first, after: $after) {
			nodes {
				identifier
				title
				priority
				state { name }
				labels { nodes { name } }
			}
			pageInfo {
				hasNextPage
				endCursor
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

	var allIssues []paintress.Issue
	var cursor string

	for {
		vars := map[string]any{
			"filter": filter,
			"first":  250,
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		reqBody, err := json.Marshal(map[string]any{
			"query":     query,
			"variables": vars,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
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
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
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

		for _, node := range gqlResp.Data.Issues.Nodes {
			labels := make([]string, 0, len(node.Labels.Nodes))
			for _, l := range node.Labels.Nodes {
				labels = append(labels, l.Name)
			}
			allIssues = append(allIssues, paintress.Issue{
				ID:       node.Identifier,
				Title:    node.Title,
				Priority: node.Priority,
				Status:   node.State.Name,
				Labels:   labels,
			})
		}

		if !gqlResp.Data.Issues.PageInfo.HasNextPage {
			break
		}
		cursor = gqlResp.Data.Issues.PageInfo.EndCursor
	}

	return allIssues, nil
}
