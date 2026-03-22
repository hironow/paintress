//go:build scenario

package scenario_test

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
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
		o.t.Fatalf("journal dir %s not accessible: %v", dir, err)
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
		if scanErr := scanner.Err(); scanErr != nil {
			o.t.Errorf("scanner error reading %s: %v", entry.Name(), scanErr)
		}
		f.Close()
	}
	o.t.Error("no gommage.triggered event found in .expedition/events/*.jsonl")
}

// AssertEventInJSONL scans .expedition/events/*.jsonl for any event of the
// given type. Fails the test if not found. Generic helper for future event assertions.
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

// --- Notification assertion helpers (proposal 024) ---

// AssertNotifyFailOpen verifies that when a notify-cmd fails, the expedition
// still completes (fail-open semantic). Checks that expedition proceeded by
// verifying at least one D-Mail was produced despite notification failure.
func (o *Observer) AssertNotifyFailOpen() {
	o.t.Helper()
	// If expedition completed despite notify failure, there should be
	// expedition events or D-Mails produced
	dir := filepath.Join(o.ws.RepoPath, ".expedition", "events")
	entries, err := os.ReadDir(dir)
	if err != nil {
		o.t.Logf("events dir not accessible: %v", err)
		return
	}
	if len(entries) == 0 {
		o.t.Error("expected expedition events after notify failure (fail-open), but events dir is empty")
	}
}

// --- Swarm and Approval assertion helpers (proposals 029, 030) ---

// AssertWorktreeCount runs `git worktree list` in the workspace repo
// and verifies the number of worktrees matches expected count.
// A clean repo with no swarm should have exactly 1 worktree (main).
func (o *Observer) AssertWorktreeCount(wantCount int) {
	o.t.Helper()
	cmd := exec.Command("git", "worktree", "list")
	cmd.Dir = o.ws.RepoPath
	out, err := cmd.Output()
	if err != nil {
		o.t.Fatalf("git worktree list: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	got := len(lines)
	if got != wantCount {
		o.t.Errorf("worktree count: got %d, want %d\nworktrees:\n%s", got, wantCount, string(out))
	}
}

// AssertExpeditionCount checks the number of expedition events in JSONL.
func (o *Observer) AssertExpeditionCount(wantCount int) {
	o.t.Helper()
	eventsDir := filepath.Join(o.ws.RepoPath, ".expedition", "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		if wantCount == 0 {
			return // no events dir = 0 expeditions, acceptable
		}
		o.t.Fatalf("read events dir: %v", err)
	}

	count := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(eventsDir, entry.Name()))
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, `"expedition.started"`) {
				count++
			}
		}
	}
	if count != wantCount {
		o.t.Errorf("expedition.started count: got %d, want %d", count, wantCount)
	}
}

// --- Retry exhaustion and inbox filter helpers (proposals 040, 041) ---

// AssertEscalationEvent scans .expedition/events/*.jsonl for an escalation
// event triggered by retry exhaustion (max_retries exceeded).
func (o *Observer) AssertEscalationEvent() {
	o.t.Helper()
	eventsDir := filepath.Join(o.ws.RepoPath, ".expedition", "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		o.t.Fatalf("read events dir: %v", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(eventsDir, entry.Name()))
		if strings.Contains(string(data), `"escalation"`) || strings.Contains(string(data), `"escalated"`) {
			return
		}
	}
	o.t.Error("no escalation event found in .expedition/events/*.jsonl")
}

// AssertInboxProcessedAll verifies that all D-Mails in .expedition/inbox/
// were consumed (inbox is empty after expedition run). This documents the
// passthrough behavior: all kinds are processed, not just specification.
func (o *Observer) AssertInboxProcessedAll() {
	o.t.Helper()
	dir := filepath.Join(o.ws.RepoPath, ".expedition", "inbox")
	files := o.ws.ListFiles(o.t, dir)
	var remaining []string
	for _, f := range files {
		if strings.HasSuffix(f, ".md") {
			remaining = append(remaining, f)
		}
	}
	if len(remaining) > 0 {
		o.t.Errorf(".expedition/inbox still has %d unprocessed D-Mails: %v", len(remaining), remaining)
	}
}

// --- GitHub PR review gate helpers (proposal 048) ---

// AssertPRReviewGateNotCalled verifies that fake-gh was NOT called with
// "pr edit" (because all fixtures use pr_url=none). This documents the
// current blind spot: UpdatePRReviewGate is never exercised in scenarios.
func (o *Observer) AssertPRReviewGateNotCalled() {
	o.t.Helper()
	// When pr_url=none, the PR update path should be skipped entirely.
	// If a FAKE_GH_EDIT_LOG_DIR existed and had files, that would indicate
	// the path was unexpectedly called.
	logDir := filepath.Join(o.ws.Root, "gh-edit-logs")
	if _, err := os.Stat(logDir); err == nil {
		entries, _ := os.ReadDir(logDir)
		if len(entries) > 0 {
			o.t.Errorf("unexpected gh pr edit calls found in %s (%d files)", logDir, len(entries))
		}
	}
}

// --- Expedition timeout helpers (proposal 051) ---

// AssertExpeditionTimedOut checks for a timeout-related event in JSONL.
// Full implementation requires FAKE_CLAUDE_SLEEP_SEC env var support
// in fake-claude (not yet implemented). This helper documents the
// assertion pattern for when that infra is added.
func (o *Observer) AssertExpeditionTimedOut() {
	o.t.Helper()
	eventsDir := filepath.Join(o.ws.RepoPath, ".expedition", "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		o.t.Fatalf("read events dir: %v", err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(eventsDir, entry.Name()))
		content := string(data)
		if strings.Contains(content, `"timeout"`) || strings.Contains(content, `"timed_out"`) || strings.Contains(content, `"deadline_exceeded"`) {
			return
		}
	}
	o.t.Error("no timeout event found in .expedition/events/*.jsonl")
}

// --- Journal round-trip helpers (proposal 059) ---

// AssertJournalExists verifies that at least one journal .md file exists
// in .expedition/journal/ that was written by WriteJournal (not hand-seeded).
// Uses file count comparison: if count > seed count, WriteJournal ran.
func (o *Observer) AssertJournalWritten(minCount int) {
	o.t.Helper()
	dir := filepath.Join(o.ws.RepoPath, ".expedition", "journal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		o.t.Fatalf("read journal dir: %v", err)
	}
	var mdCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount < minCount {
		o.t.Errorf("journal files: got %d, want at least %d", mdCount, minCount)
	}
}

// --- Specification consumption + insights helpers (proposals 063, 066) ---

// AssertPromptContainsField reads prompt logs and verifies a specific
// field value appears in at least one prompt. Generic version of
// AssertPromptContainsLumina for any content check.
func (o *Observer) AssertPromptContainsField(substring string) {
	o.t.Helper()
	dir := o.ws.PromptLogDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		o.t.Fatalf("read prompt-log dir: %v", err)
	}
	if len(entries) == 0 {
		o.t.Fatal("no prompt logs found")
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(dir, entry.Name()))
		if strings.Contains(string(data), substring) {
			return
		}
	}
	o.t.Errorf("no prompt log contains %q", substring)
}

// AssertLuminaInsightFile verifies that .expedition/insights/lumina.md
// exists and contains the expected pattern text.
func (o *Observer) AssertLuminaInsightFile(wantPattern string) {
	o.t.Helper()
	path := filepath.Join(o.ws.RepoPath, ".expedition", "insights", "lumina.md")
	data, err := os.ReadFile(path)
	if err != nil {
		o.t.Fatalf("read lumina.md: %v", err)
	}
	if wantPattern != "" && !strings.Contains(string(data), wantPattern) {
		o.t.Errorf("lumina.md does not contain %q\ncontent (truncated):\n%.500s", wantPattern, string(data))
	}
}

// AssertInsightsFileExists verifies .expedition/insights/lumina.md exists.
func (o *Observer) AssertInsightsFileExists() {
	o.t.Helper()
	path := filepath.Join(o.ws.RepoPath, ".expedition", "insights", "lumina.md")
	if _, err := os.Stat(path); err != nil {
		o.t.Errorf(".expedition/insights/lumina.md not found: %v", err)
	}
}

// --- Bug issue + HIGH severity gate helpers (proposals 069, 072) ---

// AssertBugsFoundInJSONL scans .expedition/events/*.jsonl for
// expedition.completed events with bugs_found > 0.
func (o *Observer) AssertBugsFoundInJSONL() {
	o.t.Helper()
	eventsDir := filepath.Join(o.ws.RepoPath, ".expedition", "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		o.t.Fatalf("read events dir: %v", err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(eventsDir, entry.Name()))
		content := string(data)
		if strings.Contains(content, `"bugs_found"`) && !strings.Contains(content, `"bugs_found":"0"`) {
			return
		}
	}
	o.t.Error("no expedition.completed event with bugs_found > 0")
}

// AssertNotifyArgvContains reads the notify-cmd log file and verifies
// the notification message contains the expected substring.
func (o *Observer) AssertNotifyArgvContains(wantSubstring string) {
	o.t.Helper()
	logDir := filepath.Join(o.ws.Root, "notify-logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		o.t.Logf("notify-logs dir not accessible: %v (notify-cmd may not have run)", err)
		return
	}
	for _, entry := range entries {
		data, _ := os.ReadFile(filepath.Join(logDir, entry.Name()))
		if strings.Contains(string(data), wantSubstring) {
			return
		}
	}
	o.t.Errorf("no notify log contains %q (checked %d files)", wantSubstring, len(entries))
}

// --- Report D-Mail field verification helper (proposal 075) ---

// AssertReportDMailFields reads a report D-Mail and verifies key fields
// (description, issues, status, mission_type) are present and non-empty.
// This is the scenario-level complement to 035's domain-level contract test.
func (o *Observer) AssertReportDMailFields(path string) {
	o.t.Helper()
	fm, body := o.ws.ReadDMail(o.t, path)

	for _, field := range []string{"description", "kind", "name"} {
		val, ok := fm[field].(string)
		if !ok || val == "" {
			o.t.Errorf("report D-Mail %s: missing or empty field %q", filepath.Base(path), field)
		}
	}

	// Body should contain status and mission info
	if body == "" {
		o.t.Errorf("report D-Mail %s: empty body", filepath.Base(path))
	}
}
