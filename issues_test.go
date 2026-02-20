package paintress

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	out := FormatIssuesJSONL(issues)

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

func TestFetchIssues_ParsesGraphQLResponse(t *testing.T) {
	// given — mock Linear GraphQL API
	graphqlResponse := `{
		"data": {
			"issues": {
				"nodes": [
					{
						"identifier": "MY-42",
						"title": "Fix login bug",
						"priority": 2,
						"state": {"name": "Todo"},
						"labels": {"nodes": [{"name": "Bug"}]}
					},
					{
						"identifier": "MY-43",
						"title": "Add dark mode",
						"priority": 3,
						"state": {"name": "In Progress"},
						"labels": {"nodes": [{"name": "Enhancement"}]}
					}
				]
			}
		}
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(graphqlResponse))
	}))
	defer server.Close()

	// when
	issues, err := FetchIssues(server.URL, "test-api-key", "MY", "")
	if err != nil {
		t.Fatalf("FetchIssues: %v", err)
	}

	// then
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].ID != "MY-42" {
		t.Errorf("issues[0].ID = %q, want %q", issues[0].ID, "MY-42")
	}
	if issues[0].Status != "Todo" {
		t.Errorf("issues[0].Status = %q, want %q", issues[0].Status, "Todo")
	}
	if len(issues[0].Labels) != 1 || issues[0].Labels[0] != "Bug" {
		t.Errorf("issues[0].Labels = %v, want [Bug]", issues[0].Labels)
	}
	if issues[1].Title != "Add dark mode" {
		t.Errorf("issues[1].Title = %q, want %q", issues[1].Title, "Add dark mode")
	}
}

func TestFetchIssues_GraphQLErrorResponse(t *testing.T) {
	// given — API returns 200 but with errors array
	graphqlResponse := `{
		"data": null,
		"errors": [
			{"message": "You do not have access to this team", "extensions": {"code": "FORBIDDEN"}}
		]
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(graphqlResponse))
	}))
	defer server.Close()

	// when
	_, err := FetchIssues(server.URL, "test-api-key", "INVALID", "")

	// then — must return an error, not silently succeed
	if err == nil {
		t.Fatal("expected error for GraphQL error response, got nil")
	}
	if !strings.Contains(err.Error(), "You do not have access to this team") {
		t.Errorf("error should contain GraphQL message, got: %v", err)
	}
}

func TestFetchIssues_MissingAPIKey(t *testing.T) {
	// given — empty API key
	// when
	_, err := FetchIssues("http://localhost:9999", "", "MY", "")
	// then
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestFormatIssuesJSONL_EmptySlice(t *testing.T) {
	// given
	issues := []Issue{}

	// when
	out := FormatIssuesJSONL(issues)

	// then
	if out != "" {
		t.Errorf("expected empty string for empty issues, got %q", out)
	}
}
