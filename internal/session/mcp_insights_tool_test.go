package session_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// refs issue 0034 (P4): get_insights exposes the learning loop to the
// session — live Lumina recomputation from journals (revives the
// dormant ScanJournalsForLumina) + persisted insight-file parsing.

func writeJournal(t *testing.T, continent, name, body string) {
	t.Helper()
	dir := filepath.Join(continent, ".expedition", "journal")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir journal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write journal: %v", err)
	}
}

func callInsights(t *testing.T, continent, args string) map[string]any {
	t.Helper()
	req := `{"jsonrpc":"2.0","id":70,"method":"tools/call","params":{"name":"get_insights","arguments":` + args + `}}` + "\n"
	var out bytes.Buffer
	srv := session.NewMCPServer(strings.NewReader(req), &out, nil).WithContinent(continent)
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	return decodeDMailToolJSON(t, out.Bytes())
}

func TestMCPServer_ToolsList_IncludesGetInsights(t *testing.T) {
	// given
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	tools := resp["result"].(map[string]any)["tools"].([]any)
	found := false
	for _, t0 := range tools {
		entry, _ := t0.(map[string]any)
		if entry["name"] == "get_insights" {
			found = true
		}
	}
	if !found {
		t.Error("missing get_insights in tools/list")
	}
}

func TestMCPServer_GetInsights_EmptyStateIsNotAnError(t *testing.T) {
	// given: initialized continent without journals or insight files
	continent := t.TempDir()

	// when
	body := callInsights(t, continent, `{}`)

	// then
	if body["initialized"] != true {
		t.Fatalf("initialized = %v, want true (body=%v)", body["initialized"], body)
	}
	insights, _ := body["insights"].([]any)
	if len(insights) != 0 {
		t.Errorf("insights = %v, want empty array", body["insights"])
	}
}

func TestMCPServer_GetInsights_LiveLuminaFromJournals(t *testing.T) {
	// given: two failed expeditions sharing an insight (failure-pattern
	// threshold is 2) — the dormant Lumina scan must run live
	continent := t.TempDir()
	journal := `# Expedition %s

- **Status**: failed
- **Reason**: tests failed
- **Insight**: forgot to run just check before push
`
	writeJournal(t, continent, "001.md", strings.Replace(journal, "%s", "1", 1))
	writeJournal(t, continent, "002.md", strings.Replace(journal, "%s", "2", 1))

	// when
	body := callInsights(t, continent, `{}`)

	// then
	live, _ := body["live_lumina"].([]any)
	if len(live) == 0 {
		t.Fatalf("live_lumina empty, want failure pattern (body=%v)", body)
	}
	first, _ := live[0].(map[string]any)
	if first["source"] != "failure-pattern" {
		t.Errorf("live_lumina[0].source = %v, want failure-pattern", first["source"])
	}
	if !strings.Contains(first["pattern"].(string), "just check") {
		t.Errorf("live_lumina[0].pattern = %v, want the shared insight", first["pattern"])
	}
	if int(first["uses"].(float64)) != 2 {
		t.Errorf("live_lumina[0].uses = %v, want 2", first["uses"])
	}
}

func TestMCPServer_GetInsights_ReadsPersistedInsightFiles(t *testing.T) {
	// given: a persisted lumina.md in the insights ledger
	continent := t.TempDir()
	insightsDir := filepath.Join(continent, ".expedition", "insights")
	runDir := filepath.Join(continent, ".expedition", ".run")
	for _, d := range []string{insightsDir, runDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	w := session.NewInsightWriter(insightsDir, runDir)
	if err := w.Append("lumina.md", "lumina", "paintress", luminaEntryFixture()); err != nil {
		t.Fatalf("seed insight: %v", err)
	}

	// when
	body := callInsights(t, continent, `{"kind":"lumina"}`)

	// then
	insights, _ := body["insights"].([]any)
	if len(insights) != 1 {
		t.Fatalf("insights = %v, want 1 file (body=%v)", body["insights"], body)
	}
	file, _ := insights[0].(map[string]any)
	if file["kind"] != "lumina" {
		t.Errorf("kind = %v, want lumina", file["kind"])
	}
	entries, _ := file["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("entries = %v, want 1", file["entries"])
	}
	entry, _ := entries[0].(map[string]any)
	if entry["title"] == "" {
		t.Errorf("entry missing title: %v", entry)
	}
}

func TestMCPServer_GetInsights_UninitializedWithoutContinent(t *testing.T) {
	// given
	req := `{"jsonrpc":"2.0","id":71,"method":"tools/call","params":{"name":"get_insights","arguments":{}}}` + "\n"
	var out bytes.Buffer
	srv := session.NewMCPServer(strings.NewReader(req), &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	body := decodeDMailToolJSON(t, out.Bytes())
	if body["initialized"] != false {
		t.Errorf("initialized = %v, want false", body["initialized"])
	}
}

// luminaEntryFixture builds a minimal valid insight entry.
func luminaEntryFixture() domain.InsightEntry {
	return domain.InsightEntry{
		Title: "lumina-test", What: "w", Why: "y", How: "h",
		When: "always", Who: "paintress", Constraints: "none",
	}
}
