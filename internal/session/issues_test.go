package session

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
	issues, err := FetchIssues(context.Background(), server.URL, "test-api-key", "MY", "", nil)
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
	_, err := FetchIssues(context.Background(), server.URL, "test-api-key", "INVALID", "", nil)

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
	_, err := FetchIssues(context.Background(), "http://localhost:9999", "", "MY", "", nil)
	// then
	if err == nil {
		t.Fatal("expected error for empty API key")
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
	issues, err := FetchIssues(context.Background(), server.URL, "test-api-key", "MY", "", []string{"done"})
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
	_, err := FetchIssues(context.Background(), server.URL, "test-api-key", "MY", "", nil)
	if err != nil {
		t.Fatalf("FetchIssues: %v", err)
	}

	// then — GraphQL filter SHOULD contain "nin" to exclude completed/canceled
	if !strings.Contains(string(capturedBody), "nin") {
		t.Error("GraphQL filter should exclude completed/canceled by default")
	}
}

func TestFetchIssues_PaginatesMultiplePages(t *testing.T) {
	// given — API returns 2 pages
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ := io.ReadAll(r.Body)
		callCount++

		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(string(capturedBody), `"after":"cursor-page1"`) {
			// Second page — no more pages
			w.Write([]byte(`{
				"data": {
					"issues": {
						"nodes": [
							{"identifier": "MY-3", "title": "Third issue", "priority": 3, "state": {"name": "Todo"}, "labels": {"nodes": []}}
						],
						"pageInfo": {"hasNextPage": false, "endCursor": "cursor-page2"}
					}
				}
			}`))
		} else {
			// First page — has next page
			w.Write([]byte(`{
				"data": {
					"issues": {
						"nodes": [
							{"identifier": "MY-1", "title": "First issue", "priority": 1, "state": {"name": "Todo"}, "labels": {"nodes": []}},
							{"identifier": "MY-2", "title": "Second issue", "priority": 2, "state": {"name": "In Progress"}, "labels": {"nodes": []}}
						],
						"pageInfo": {"hasNextPage": true, "endCursor": "cursor-page1"}
					}
				}
			}`))
		}
	}))
	defer server.Close()

	// when
	issues, err := FetchIssues(context.Background(), server.URL, "test-api-key", "MY", "", nil)
	if err != nil {
		t.Fatalf("FetchIssues: %v", err)
	}

	// then — all 3 issues from both pages
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues across 2 pages, got %d", len(issues))
	}
	if issues[0].ID != "MY-1" {
		t.Errorf("issues[0].ID = %q, want MY-1", issues[0].ID)
	}
	if issues[2].ID != "MY-3" {
		t.Errorf("issues[2].ID = %q, want MY-3", issues[2].ID)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
}
