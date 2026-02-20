package paintress

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMain strips git environment variables that leak from parent processes
// (e.g. pre-push hooks set GIT_DIR). Without this, exec.Command("git",...)
// in both test helpers and production code under test would target the parent
// repo instead of the test's temp dir, corrupting worktree state.
func TestMain(m *testing.M) {
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	InitTracer("paintress-test", "test")
	os.Exit(m.Run())
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping test: cannot bind local port: %v", err)
	}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Listener = ln
	srv.Start()
	return srv
}

func TestPaintressRun_DryRun_FirstRun_StartsAtExpedition1(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	cfg := Config{
		Continent:      dir,
		MaxExpeditions: 5,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "echo ok",
		DevURL:         "http://localhost:3000",
		DryRun:         true,
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}

	// Dry-run should create expedition-001-prompt.md (starts at 1)
	promptFile := filepath.Join(p.logDir, "expedition-001-prompt.md")
	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("prompt file not created: %v", err)
	}

	if !containsStr(string(content), "Expedition #1") {
		t.Error("prompt should contain 'Expedition #1'")
	}
}

func TestPaintressRun_DryRun_ResumeFromFlag(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	// Plant a flag indicating expedition 7 was the last
	WriteFlag(dir, 7, "AWE-50", "success", "3")

	cfg := Config{
		Continent:      dir,
		MaxExpeditions: 5,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "echo ok",
		DevURL:         "http://localhost:3000",
		DryRun:         true,
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}

	// Should resume at expedition 8, not 1
	promptFile := filepath.Join(p.logDir, "expedition-008-prompt.md")
	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("prompt file expedition-008-prompt.md not created: %v", err)
	}

	if !containsStr(string(content), "Expedition #8") {
		t.Error("prompt should contain 'Expedition #8' (resumed from flag)")
	}

	// expedition-001-prompt.md should NOT exist
	oldPrompt := filepath.Join(p.logDir, "expedition-001-prompt.md")
	if _, err := os.Stat(oldPrompt); !os.IsNotExist(err) {
		t.Error("expedition-001-prompt.md should not exist on resume")
	}
}

func TestPaintressRun_DryRun_PreservesExistingJournals(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	// Simulate 3 previous expeditions with journals
	for i := 1; i <= 3; i++ {
		WriteJournal(dir, &ExpeditionReport{
			Expedition: i, IssueID: "AWE-" + string(rune('0'+i)),
			IssueTitle: "past", MissionType: "implement",
			Status: "success", Reason: "done", PRUrl: "none", BugIssues: "none",
		})
	}
	WriteFlag(dir, 3, "AWE-3", "success", "5")

	// Capture original content of journal 001
	original001, err := os.ReadFile(filepath.Join(jDir, "001.md"))
	if err != nil {
		t.Fatalf("pre-existing journal 001.md missing: %v", err)
	}

	cfg := Config{
		Continent:      dir,
		MaxExpeditions: 5,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "echo ok",
		DevURL:         "http://localhost:3000",
		DryRun:         true,
	}

	p := NewPaintress(cfg)
	p.Run(context.Background())

	// Verify original journals are untouched
	after001, err := os.ReadFile(filepath.Join(jDir, "001.md"))
	if err != nil {
		t.Fatal("journal 001.md was deleted")
	}
	if string(original001) != string(after001) {
		t.Error("journal 001.md was overwritten")
	}
}

func TestReadFlag_ResumeExpeditionNumber(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(dir string)
		wantLastExp   int
		wantStartExp  int
		wantRemaining string
	}{
		{
			name:          "no flag file — fresh start",
			setup:         func(dir string) {},
			wantLastExp:   0,
			wantStartExp:  1,
			wantRemaining: "?",
		},
		{
			name: "flag at expedition 5",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)
				WriteFlag(dir, 5, "AWE-10", "success", "8")
			},
			wantLastExp:   5,
			wantStartExp:  6,
			wantRemaining: "8",
		},
		{
			name: "flag at expedition 20",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)
				WriteFlag(dir, 20, "AWE-99", "failed", "2")
			},
			wantLastExp:   20,
			wantStartExp:  21,
			wantRemaining: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			flag := ReadFlag(dir)
			startExp := flag.LastExpedition + 1

			if flag.LastExpedition != tt.wantLastExp {
				t.Errorf("LastExpedition = %d, want %d", flag.LastExpedition, tt.wantLastExp)
			}
			if startExp != tt.wantStartExp {
				t.Errorf("startExp = %d, want %d", startExp, tt.wantStartExp)
			}
			if flag.Remaining != tt.wantRemaining {
				t.Errorf("Remaining = %q, want %q", flag.Remaining, tt.wantRemaining)
			}
		})
	}
}

func TestWriteJournal_ResumedNumbering(t *testing.T) {
	dir := t.TempDir()

	// Write journal at expedition 8 (simulating a resumed run)
	report := &ExpeditionReport{
		Expedition:  8,
		IssueID:     "AWE-50",
		IssueTitle:  "Fix login",
		MissionType: "fix",
		Status:      "success",
		Reason:      "done",
		PRUrl:       "https://github.com/org/repo/pull/8",
		BugIssues:   "none",
	}
	if err := WriteJournal(dir, report); err != nil {
		t.Fatal(err)
	}

	// Should create 008.md, not 001.md
	path008 := filepath.Join(dir, ".expedition", "journal", "008.md")
	if _, err := os.Stat(path008); os.IsNotExist(err) {
		t.Fatal("expected 008.md to be created")
	}

	path001 := filepath.Join(dir, ".expedition", "journal", "001.md")
	if _, err := os.Stat(path001); !os.IsNotExist(err) {
		t.Error("001.md should not exist — journal should use resumed number")
	}

	content, err := os.ReadFile(path008)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(content), "Expedition #8") {
		t.Error("journal should reference expedition #8")
	}
}

// === Sentinel Errors ===

func TestSentinelErrors_AreDistinct(t *testing.T) {
	if errors.Is(errGommage, errComplete) {
		t.Error("errGommage and errComplete must be distinct errors")
	}
	if errGommage.Error() == "" {
		t.Error("errGommage must have a non-empty message")
	}
	if errComplete.Error() == "" {
		t.Error("errComplete must have a non-empty message")
	}
}

// === Swarm Mode DryRun Integration Tests ===

func TestSwarmMode_DryRun_CreatesUniquePrompts(t *testing.T) {
	dir := setupTestRepo(t)

	cfg := Config{
		Continent:      dir,
		Workers:        3,
		MaxExpeditions: 3,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	logDir := filepath.Join(dir, ".expedition", ".run", "logs")
	prompts, err := filepath.Glob(filepath.Join(logDir, "expedition-*-prompt.md"))
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(prompts) != 3 {
		t.Errorf("expected 3 prompt files, got %d: %v", len(prompts), prompts)
	}

	// Verify summary counters reflect DryRun expeditions
	totalRan := p.totalSuccess.Load() + p.totalFailed.Load() + p.totalSkipped.Load()
	if totalRan != 3 {
		t.Errorf("expected totalRan=3 (DryRun expeditions counted), got %d", totalRan)
	}

	// Verify all expedition numbers are unique
	seen := make(map[string]bool)
	for _, p := range prompts {
		base := filepath.Base(p)
		if seen[base] {
			t.Errorf("duplicate prompt file: %s", base)
		}
		seen[base] = true
	}
}

func TestSwarmMode_DryRun_SingleWorker(t *testing.T) {
	dir := setupTestRepo(t)

	cfg := Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	logDir := filepath.Join(dir, ".expedition", ".run", "logs")
	prompts, _ := filepath.Glob(filepath.Join(logDir, "expedition-*-prompt.md"))
	if len(prompts) != 1 {
		t.Errorf("expected 1 prompt file, got %d", len(prompts))
	}
}

func TestSwarmMode_Gommage_StopsAllWorkers(t *testing.T) {
	dir := setupTestRepo(t)

	// Start a trivial HTTP server so DevServer.Start() succeeds immediately
	srv := newTestServer(t)
	defer srv.Close()

	cfg := Config{
		Continent:      dir,
		Workers:        2,
		MaxExpeditions: 20,
		DryRun:         false,
		BaseBranch:     "main",
		ClaudeCmd:      "/bin/false", // always exits with code 1
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 1 {
		t.Errorf("expected exit code 1 (gommage), got %d", code)
	}

	totalFailed := p.totalFailed.Load()
	if totalFailed < int64(maxConsecutiveFailures) {
		t.Errorf("expected at least %d failures before gommage, got %d",
			maxConsecutiveFailures, totalFailed)
	}

	// Should not have run all 20 expeditions (gommage stops early)
	totalRan := p.totalSuccess.Load() + p.totalFailed.Load() + p.totalSkipped.Load()
	if totalRan >= 20 {
		t.Errorf("gommage should have stopped early, but ran all %d expeditions", totalRan)
	}
}

func TestSwarmMode_MaxExpeditions_LessThan_Workers(t *testing.T) {
	dir := setupTestRepo(t)

	cfg := Config{
		Continent:      dir,
		Workers:        3,
		MaxExpeditions: 2, // fewer than workers
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	logDir := filepath.Join(dir, ".expedition", ".run", "logs")
	prompts, _ := filepath.Glob(filepath.Join(logDir, "expedition-*-prompt.md"))
	if len(prompts) != 2 {
		t.Errorf("expected 2 prompt files (MaxExpeditions=2), got %d: %v", len(prompts), prompts)
	}
}

func TestSwarmMode_ContextCancellation_GracefulShutdown(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	cfg := Config{
		Continent:      dir,
		Workers:        2,
		MaxExpeditions: 100,
		DryRun:         false,
		BaseBranch:     "main",
		ClaudeCmd:      "/bin/false",
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     60,
		Model:          "opus",
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(3 * time.Second)
		cancel()
	}()

	p := NewPaintress(cfg)
	code := p.Run(ctx)

	// Workers fail fast with /bin/false, but cooldown is 10s.
	// After ~3s cancel fires during cooldown, causing graceful exit.
	// Exit code is 130 (interrupted) OR 1 (gommage, if 3 failures happen before cancel).
	// Both are acceptable — the key is no deadlock.
	if code != 130 && code != 1 {
		t.Errorf("expected exit code 130 (interrupted) or 1 (gommage), got %d", code)
	}

	// Verify it didn't run all 100 expeditions
	totalRan := p.totalSuccess.Load() + p.totalFailed.Load() + p.totalSkipped.Load()
	if totalRan >= 100 {
		t.Errorf("should have stopped early, but ran %d expeditions", totalRan)
	}
}

func TestSwarmMode_FlagResume_ParallelNumbering(t *testing.T) {
	dir := setupTestRepo(t)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	// Plant flag at expedition 4
	WriteFlag(dir, 4, "AWE-10", "success", "10")

	cfg := Config{
		Continent:      dir,
		Workers:        3,
		MaxExpeditions: 3,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	logDir := filepath.Join(dir, ".expedition", ".run", "logs")

	// Should have prompts for expeditions 5, 6, 7 (not 1, 2, 3)
	for _, expNum := range []int{5, 6, 7} {
		promptFile := filepath.Join(logDir, fmt.Sprintf("expedition-%03d-prompt.md", expNum))
		if _, err := os.Stat(promptFile); os.IsNotExist(err) {
			t.Errorf("expected prompt file for expedition %d, not found", expNum)
		}
	}

	// Should NOT have prompts for expeditions 1-4
	for _, expNum := range []int{1, 2, 3, 4} {
		promptFile := filepath.Join(logDir, fmt.Sprintf("expedition-%03d-prompt.md", expNum))
		if _, err := os.Stat(promptFile); !os.IsNotExist(err) {
			t.Errorf("prompt file for expedition %d should not exist (resumed from 5)", expNum)
		}
	}
}

// TestSwarmMode_DeadlineExceeded_ReturnsNonZero verifies that a context
// timeout (DeadlineExceeded) is treated as interrupted, not success.
func TestSwarmMode_DeadlineExceeded_ReturnsNonZero(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	cfg := Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 100,
		DryRun:         false,
		BaseBranch:     "main",
		ClaudeCmd:      "sleep 5", // ensure deadline fires mid-expedition
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     60,
		Model:          "opus",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	p := NewPaintress(cfg)
	code := p.Run(ctx)

	// DeadlineExceeded should return non-zero (130), not 0
	if code == 0 {
		t.Errorf("expected non-zero exit code for DeadlineExceeded, got 0")
	}
}

// TestSwarmMode_DeadlineExceeded_NotCountedAsFailure verifies that a
// context deadline during expedition execution does not increment failure
// counters or trigger gommage.
func TestSwarmMode_DeadlineExceeded_NotCountedAsFailure(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// Script that sleeps long enough for the deadline to fire mid-expedition.
	// Use 'exec' so bash replaces itself with sleep (no child process leak).
	sleepScript := filepath.Join(dir, "slowclaude.sh")
	if err := os.WriteFile(sleepScript, []byte("#!/bin/bash\nexec sleep 999\n"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 1,
		DryRun:         false,
		BaseBranch:     "main",
		ClaudeCmd:      sleepScript,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     60, // expedition timeout is long
		Model:          "opus",
	}

	// Context deadline is short — fires while expeditions are running
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	p := NewPaintress(cfg)
	code := p.Run(ctx)

	// Should be interrupted (130), NOT gommage (1)
	if code != 130 {
		t.Errorf("expected exit code 130 (interrupted), got %d", code)
	}

	// Deadline-induced errors should NOT count as real failures
	if p.totalFailed.Load() > 0 {
		t.Errorf("deadline cancellation should not count as failure, got totalFailed=%d",
			p.totalFailed.Load())
	}
}

func TestSwarmMode_SingleWorker_WithWorktreePool(t *testing.T) {
	dir := setupTestRepo(t)

	cfg := Config{
		Continent:      dir,
		Workers:        1, // single worker WITH worktree pool
		MaxExpeditions: 2,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	logDir := filepath.Join(dir, ".expedition", ".run", "logs")
	prompts, _ := filepath.Glob(filepath.Join(logDir, "expedition-*-prompt.md"))

	// DryRun respects MaxExpeditions: single worker loops to create 2 prompts
	if len(prompts) != 2 {
		t.Errorf("expected 2 prompt files (MaxExpeditions=2), got %d", len(prompts))
	}
}

// TestSwarmMode_StatusComplete_CountedInSummary verifies that an expedition
// receiving StatusComplete is counted in the summary totals so printSummary
// does not under-report executed expeditions.
func TestSwarmMode_StatusComplete_CountedInSummary(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// Create a script that outputs __EXPEDITION_COMPLETE__
	completeScript := filepath.Join(dir, "complete.sh")
	if err := os.WriteFile(completeScript, []byte("#!/bin/bash\necho '__EXPEDITION_COMPLETE__'\n"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 5,
		DryRun:         false,
		BaseBranch:     "main",
		ClaudeCmd:      completeScript,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	// errComplete → exit code 0
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// The expedition ran (ClaudeCmd executed, output parsed) — must be counted
	// in totalAttempted, but NOT in any outcome counter (success/failed/skipped).
	if p.totalAttempted.Load() == 0 {
		t.Error("StatusComplete expedition not counted in totalAttempted")
	}
	if p.totalSuccess.Load() != 0 || p.totalFailed.Load() != 0 || p.totalSkipped.Load() != 0 {
		t.Errorf("StatusComplete should not increment outcome counters: success=%d failed=%d skipped=%d",
			p.totalSuccess.Load(), p.totalFailed.Load(), p.totalSkipped.Load())
	}
}

// TestSwarmMode_RunResetsCounters verifies that calling Run() twice on the
// same Paintress resets all run-scoped counters so the second run starts clean.
func TestSwarmMode_RunResetsCounters(t *testing.T) {
	dir := setupTestRepo(t)

	cfg := Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 2,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)

	// First run: 2 DryRun expeditions
	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("first Run() = %d, want 0", code)
	}
	if p.totalAttempted.Load() != 2 {
		t.Fatalf("first run: expected totalAttempted=2, got %d", p.totalAttempted.Load())
	}

	// Second run on same instance: counters should be fresh
	code = p.Run(context.Background())
	if code != 0 {
		t.Fatalf("second Run() = %d, want 0", code)
	}
	if p.totalAttempted.Load() != 2 {
		t.Errorf("second run: expected totalAttempted=2 (reset), got %d", p.totalAttempted.Load())
	}
	if p.totalSuccess.Load() != 2 {
		t.Errorf("second run: expected totalSuccess=2 (reset), got %d", p.totalSuccess.Load())
	}
	if p.consecutiveFailures.Load() != 0 {
		t.Errorf("second run: expected consecutiveFailures=0 (reset), got %d", p.consecutiveFailures.Load())
	}
}

// TestSwarmMode_StatusParseError_WritesJournalAndFlag verifies that when an
// expedition output cannot be parsed (StatusParseError), a journal entry and
// flag checkpoint are still written — matching the behavior of all other
// failure paths (err != nil, StatusFailed).
func TestSwarmMode_StatusParseError_WritesJournalAndFlag(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// Script that outputs garbage (no expedition markers) → StatusParseError
	badScript := filepath.Join(dir, "badreport.sh")
	if err := os.WriteFile(badScript, []byte("#!/bin/bash\necho 'no markers here'\n"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 1,
		DryRun:         false,
		BaseBranch:     "main",
		ClaudeCmd:      badScript,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	p.Run(context.Background())

	// Journal entry should exist for expedition 1
	journalPath := filepath.Join(dir, ".expedition", "journal", "001.md")
	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		t.Error("StatusParseError did not write journal entry")
	}

	// Flag should be updated to expedition 1
	flag := ReadFlag(dir)
	if flag.LastExpedition != 1 {
		t.Errorf("StatusParseError did not update flag: expected last_expedition=1, got %d", flag.LastExpedition)
	}
	if flag.LastStatus != "parse_error" {
		t.Errorf("flag status: expected 'parse_error', got %q", flag.LastStatus)
	}
}

// TestSwarmMode_FlagMonotonic_NoRegression verifies that the flag checkpoint
// is monotonic: a lower-numbered expedition completing after a higher one
// must not overwrite the flag with a smaller expedition number.
func TestSwarmMode_FlagMonotonic_NoRegression(t *testing.T) {
	dir := setupTestRepo(t)

	// Directly test the monotonic guard via Paintress.writeFlag
	cfg := Config{Continent: dir, BaseBranch: "main", Model: "opus"}
	p := NewPaintress(cfg)

	// Write flag for expedition 5
	p.flagMu.Lock()
	p.writeFlag(5, "ISS-5", "success", "10")
	p.flagMu.Unlock()

	// Attempt to write flag for expedition 3 (out-of-order completion)
	p.flagMu.Lock()
	p.writeFlag(3, "ISS-3", "success", "12")
	p.flagMu.Unlock()

	// Flag should still show expedition 5, not 3
	flag := ReadFlag(dir)
	if flag.LastExpedition != 5 {
		t.Errorf("flag regressed: expected last_expedition=5, got %d", flag.LastExpedition)
	}
	if flag.LastIssue != "ISS-5" {
		t.Errorf("flag regressed: expected last_issue=ISS-5, got %s", flag.LastIssue)
	}
}

func TestPaintressRun_NoDev_SkipsDevServer(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	cfg := Config{
		Continent:      dir,
		MaxExpeditions: 1,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		NoDev:          true,
		DryRun:         true,
	}

	p := NewPaintress(cfg)

	// devServer should be nil — no panic during Run
	if p.devServer != nil {
		t.Fatal("devServer should be nil when NoDev=true")
	}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}
}
