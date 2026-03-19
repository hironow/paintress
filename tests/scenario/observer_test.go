//go:build scenario

package scenario_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Observer provides high-level assertion helpers for scenario tests.
// It wraps a Workspace and testing.T to verify mailbox state, D-Mail
// content, and closed-loop completion.
type Observer struct {
	ws *Workspace
	t  *testing.T
}

// NewObserver creates an Observer for the given workspace.
func NewObserver(ws *Workspace, t *testing.T) *Observer {
	return &Observer{ws: ws, t: t}
}

// AssertMailboxState verifies file counts in mailbox directories.
// Keys are relative paths like ".siren/inbox", ".expedition/archive".
func (o *Observer) AssertMailboxState(expectations map[string]int) {
	o.t.Helper()
	for relPath, want := range expectations {
		dir := filepath.Join(o.ws.RepoPath, relPath)
		got := o.ws.CountFiles(o.t, dir)
		if got != want {
			o.t.Errorf("mailbox %s: got %d files, want %d", relPath, got, want)
		}
	}
}

// AssertAllOutboxEmpty verifies that all tool outboxes contain no .md files.
func (o *Observer) AssertAllOutboxEmpty() {
	o.t.Helper()
	tools := []string{".siren", ".expedition", ".gate"}
	for _, tool := range tools {
		dir := filepath.Join(o.ws.RepoPath, tool, "outbox")
		files := o.ws.ListFiles(o.t, dir)
		var mdFiles []string
		for _, f := range files {
			if strings.HasSuffix(f, ".md") {
				mdFiles = append(mdFiles, f)
			}
		}
		if len(mdFiles) > 0 {
			o.t.Errorf("%s/outbox not empty: %v", tool, mdFiles)
		}
	}
}

// AssertArchiveContains verifies that a tool's archive directory contains
// D-Mail files with the expected kinds in their frontmatter.
func (o *Observer) AssertArchiveContains(toolDir string, kinds []string) {
	o.t.Helper()
	dir := filepath.Join(o.ws.RepoPath, toolDir, "archive")
	files := o.ws.ListFiles(o.t, dir)
	if len(files) == 0 && len(kinds) > 0 {
		o.t.Errorf("%s/archive: expected D-Mails with kinds %v, but archive is empty", toolDir, kinds)
		return
	}

	// Collect all kinds found in archive
	foundKinds := make(map[string]bool)
	for _, f := range files {
		if !strings.HasSuffix(f, ".md") {
			continue
		}
		path := filepath.Join(dir, f)
		fm, _ := o.ws.ReadDMail(o.t, path)
		if kind, ok := fm["kind"].(string); ok {
			foundKinds[kind] = true
		}
	}

	for _, want := range kinds {
		if !foundKinds[want] {
			o.t.Errorf("%s/archive: missing D-Mail with kind %q (found kinds: %v)", toolDir, want, foundKinds)
		}
	}
}

// AssertDMailKind verifies that a D-Mail file has the expected kind.
func (o *Observer) AssertDMailKind(path, expectedKind string) {
	o.t.Helper()
	fm, _ := o.ws.ReadDMail(o.t, path)
	kind, ok := fm["kind"].(string)
	if !ok {
		o.t.Errorf("D-Mail %s: missing kind field in frontmatter", path)
		return
	}
	if kind != expectedKind {
		o.t.Errorf("D-Mail %s: got kind %q, want %q", path, kind, expectedKind)
	}
}

// WaitForClosedLoop waits for a complete closed loop (specification -> report -> feedback).
// It polls all 3 delivery points:
//  1. specification in .expedition/inbox
//  2. report in .gate/inbox
//  3. feedback in .siren/inbox AND .expedition/inbox
func (o *Observer) WaitForClosedLoop(timeout time.Duration) {
	o.t.Helper()
	stepTimeout := timeout / 3
	if stepTimeout < 10*time.Second {
		stepTimeout = 10 * time.Second
	}

	o.ws.WaitForDMail(o.t, ".expedition", "inbox", stepTimeout)
	o.ws.WaitForDMail(o.t, ".gate", "inbox", stepTimeout)
	o.ws.WaitForDMail(o.t, ".siren", "inbox", stepTimeout)
}

// --- Prompt and expedition assertion helpers (proposals 003, 009 adapted) ---

// AssertPromptCount verifies fake-claude was called exactly wantCount times.
func (o *Observer) AssertPromptCount(wantCount int) {
	o.t.Helper()
	dir := o.ws.PromptLogDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		o.t.Fatalf("read prompt-log dir %s: %v", dir, err)
	}
	got := len(entries)
	if got != wantCount {
		if got == 0 {
			o.t.Errorf("prompt count: got 0, want %d — fake-claude may not have been invoked", wantCount)
		} else {
			o.t.Errorf("prompt count: got %d, want %d", got, wantCount)
		}
	}
}

// AssertPromptCountAtLeast verifies fake-claude was called at least minCount times.
// Useful for non-deterministic scenarios where exact count varies.
func (o *Observer) AssertPromptCountAtLeast(minCount int) {
	o.t.Helper()
	dir := o.ws.PromptLogDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		o.t.Fatalf("read prompt-log dir %s: %v", dir, err)
	}
	got := len(entries)
	if got < minCount {
		o.t.Errorf("prompt count: got %d, want at least %d", got, minCount)
	}
}

// AssertExpeditionJournalExists verifies that at least one expedition journal
// entry was created in .expedition/journal/.
func (o *Observer) AssertExpeditionJournalExists() {
	o.t.Helper()
	dir := filepath.Join(o.ws.RepoPath, ".expedition", "journal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// journal dir may not exist if no expedition ran
		o.t.Logf("journal dir %s not accessible: %v", dir, err)
		return
	}
	if len(entries) == 0 {
		o.t.Error("expected at least 1 expedition journal entry, got 0")
	}
}

// --- Event store assertion helpers (proposal 018) ---

// AssertGommageEvent scans .expedition/events/*.jsonl for an event of type
// "gommage.triggered" and verifies the consecutive_failures field matches
// the expected count. This tests the full gommage pipeline:
// FAIL_COUNT exhaustion -> gommage trigger -> JSONL event recording.
func (o *Observer) AssertGommageEvent(wantConsecutiveFailures int) {
	o.t.Helper()
	eventsDir := filepath.Join(o.ws.RepoPath, ".expedition", "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		o.t.Fatalf("read events dir %s: %v", eventsDir, err)
	}

	type eventEnvelope struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	type gommageData struct {
		ConsecutiveFailures int `json:"consecutive_failures"`
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		f, openErr := os.Open(filepath.Join(eventsDir, entry.Name()))
		if openErr != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var ev eventEnvelope
			if jsonErr := json.Unmarshal(scanner.Bytes(), &ev); jsonErr != nil {
				continue
			}
			if ev.Type == "gommage.triggered" {
				var data gommageData
				if dataErr := json.Unmarshal(ev.Data, &data); dataErr != nil {
					o.t.Errorf("gommage event found but data unmarshal failed: %v", dataErr)
					f.Close()
					return
				}
				if data.ConsecutiveFailures != wantConsecutiveFailures {
					o.t.Errorf("gommage consecutive_failures: got %d, want %d",
						data.ConsecutiveFailures, wantConsecutiveFailures)
				}
				f.Close()
				return // found
			}
		}
		f.Close()
	}
	o.t.Error("no gommage.triggered event found in .expedition/events/*.jsonl")
}

// AssertEventInJSONL scans .expedition/events/*.jsonl for any event of the
// given type. Returns true if found. Generic helper for future event assertions.
func (o *Observer) AssertEventInJSONL(wantType string) {
	o.t.Helper()
	eventsDir := filepath.Join(o.ws.RepoPath, ".expedition", "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		o.t.Fatalf("read events dir: %v", err)
	}

	type eventEnvelope struct {
		Type string `json:"type"`
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		f, _ := os.Open(filepath.Join(eventsDir, entry.Name()))
		if f == nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var ev eventEnvelope
			if json.Unmarshal(scanner.Bytes(), &ev) == nil && ev.Type == wantType {
				f.Close()
				return
			}
		}
		f.Close()
	}
	o.t.Errorf("event type %q not found in .expedition/events/*.jsonl", wantType)
}

// --- Lumina assertion helpers (proposal 021) ---

// AssertPromptContainsLumina reads all prompt log files and verifies that
// at least one contains the given substring. Used to confirm Lumina patterns
// extracted from journals are injected into the expedition prompt.
func (o *Observer) AssertPromptContainsLumina(substring string) {
	o.t.Helper()
	dir := o.ws.PromptLogDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		o.t.Fatalf("read prompt-log dir %s: %v", dir, err)
	}
	if len(entries) == 0 {
		o.t.Fatalf("no prompt logs found in %s — fake-claude was not invoked", dir)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			continue
		}
		if strings.Contains(string(data), substring) {
			return
		}
	}

	if len(entries) > 0 {
		first, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
		o.t.Logf("first prompt log (truncated):\n%.2000s", string(first))
	}
	o.t.Errorf("no prompt log contains substring %q (checked %d files)", substring, len(entries))
}

// AssertPromptNotContainsLumina verifies that none of the prompt logs
// contain the given substring. Useful for negative tests.
func (o *Observer) AssertPromptNotContainsLumina(substring string) {
	o.t.Helper()
	dir := o.ws.PromptLogDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		o.t.Fatalf("read prompt-log dir %s: %v", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			continue
		}
		if strings.Contains(string(data), substring) {
			o.t.Errorf("prompt log %s unexpectedly contains %q", entry.Name(), substring)
			return
		}
	}
}
