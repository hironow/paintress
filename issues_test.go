package paintress

import (
	"encoding/json"
	"io"
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
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("Authorization header = %q, want %q", auth, "Bearer test-api-key")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(graphqlResponse))
	}))
	defer server.Close()

	// when
	issues, err := FetchIssues(server.URL, "test-api-key", "MY", "", nil)
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
	_, err := FetchIssues(server.URL, "test-api-key", "INVALID", "", nil)

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
	_, err := FetchIssues("http://localhost:9999", "", "MY", "", nil)
	// then
	if err == nil {
		t.Fatal("expected error for empty API key")
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

func TestFetchIssues_IncludesCompletedWhenStateFilterRequests(t *testing.T) {
	// given — API returns a completed issue
	graphqlResponse := `{
		"data": {
			"issues": {
				"nodes": [
					{
						"identifier": "MY-50",
						"title": "Already done",
						"priority": 3,
						"state": {"name": "Done"},
						"labels": {"nodes": []}
					}
				]
			}
		}
	}`

	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(graphqlResponse))
	}))
	defer server.Close()

	// when — state filter includes a completed state
	issues, err := FetchIssues(server.URL, "test-api-key", "MY", "", []string{"done"})
	if err != nil {
		t.Fatalf("FetchIssues: %v", err)
	}

	// then — completed issue is returned (not excluded by GraphQL filter)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].ID != "MY-50" {
		t.Errorf("issues[0].ID = %q, want MY-50", issues[0].ID)
	}

	// then — GraphQL filter should NOT contain "nin" for state.type
	if strings.Contains(string(capturedBody), "nin") {
		t.Error("GraphQL filter should not exclude completed/canceled when stateFilter is provided")
	}
}

func TestFetchIssues_ExcludesCompletedByDefault(t *testing.T) {
	// given
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"issues":{"nodes":[]}}}`))
	}))
	defer server.Close()

	// when — no state filter (default)
	_, err := FetchIssues(server.URL, "test-api-key", "MY", "", nil)
	if err != nil {
		t.Fatalf("FetchIssues: %v", err)
	}

	// then — GraphQL filter SHOULD contain "nin" to exclude completed/canceled
	if !strings.Contains(string(capturedBody), "nin") {
		t.Error("GraphQL filter should exclude completed/canceled by default")
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
