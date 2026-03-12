package session

// white-box-reason: session internals: tests unexported newTestPaintress helper and env isolation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// TestMain strips git environment variables that leak from parent processes
// (e.g. pre-push hooks set GIT_DIR). Without this, exec.Command("git",...)
// in both test helpers and production code under test would target the parent
// repo instead of the test's temp dir, corrupting worktree state.
func TestMain(m *testing.M) {
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// Shorten cooldowns so SwarmMode tests don't accumulate minutes of idle waits.
	expeditionCooldown = 10 * time.Millisecond
	worktreeReleaseTimeout = 2 * time.Second
	devServerReadyTimeout = 5 * time.Second
	devServerStopTimeout = 1 * time.Second

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

// streamJSONScript creates a bash script that emits the given text as
// stream-json NDJSON (assistant message + result), matching the format
// expected by StreamReader after switching to --output-format stream-json.
func streamJSONScript(t *testing.T, dir, name, text string) string {
	t.Helper()
	assistantMsg := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"id":    "msg_test",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{"type": "text", "text": text},
			},
		},
	}
	resultMsg := map[string]any{
		"type":   "result",
		"result": text,
	}
	aJSON, _ := json.Marshal(assistantMsg)
	rJSON, _ := json.Marshal(resultMsg)

	scriptPath := filepath.Join(dir, name)
	content := fmt.Sprintf("#!/bin/bash\necho '%s'\necho '%s'\n", string(aJSON), string(rJSON))
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	return scriptPath
}

func TestPaintressRun_DryRun_FirstRun_StartsAtExpedition1(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	cfg := domain.Config{
		Continent:      dir,
		MaxExpeditions: 5,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "echo ok",
		DevURL:         "http://localhost:3000",
		DryRun:         true,
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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
	WriteFlag(dir, 7, "AWE-50", "success", "3", 0)

	cfg := domain.Config{
		Continent:      dir,
		MaxExpeditions: 5,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "echo ok",
		DevURL:         "http://localhost:3000",
		DryRun:         true,
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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
	if _, err := os.Stat(oldPrompt); !errors.Is(err, fs.ErrNotExist) {
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
		WriteJournal(dir, &domain.ExpeditionReport{
			Expedition: i, IssueID: "AWE-" + string(rune('0'+i)),
			IssueTitle: "past", MissionType: "implement",
			Status: "success", Reason: "done", PRUrl: "none", BugIssues: "none",
		})
	}
	WriteFlag(dir, 3, "AWE-3", "success", "5", 0)

	// Capture original content of journal 001
	original001, err := os.ReadFile(filepath.Join(jDir, "001.md"))
	if err != nil {
		t.Fatalf("pre-existing journal 001.md missing: %v", err)
	}

	cfg := domain.Config{
		Continent:      dir,
		MaxExpeditions: 5,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "echo ok",
		DevURL:         "http://localhost:3000",
		DryRun:         true,
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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
				WriteFlag(dir, 5, "AWE-10", "success", "8", 0)
			},
			wantLastExp:   5,
			wantStartExp:  6,
			wantRemaining: "8",
		},
		{
			name: "flag at expedition 20",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)
				WriteFlag(dir, 20, "AWE-99", "failed", "2", 0)
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
	report := &domain.ExpeditionReport{
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
	if _, err := os.Stat(path008); errors.Is(err, fs.ErrNotExist) {
		t.Fatal("expected 008.md to be created")
	}

	path001 := filepath.Join(dir, ".expedition", "journal", "001.md")
	if _, err := os.Stat(path001); !errors.Is(err, fs.ErrNotExist) {
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

	cfg := domain.Config{
		Continent:      dir,
		Workers:        3,
		MaxExpeditions: 3,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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

	cfg := domain.Config{
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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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

	cfg := domain.Config{
		Continent:      dir,
		Workers:        3,
		MaxExpeditions: 2, // fewer than workers
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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

	cfg := domain.Config{
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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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
	WriteFlag(dir, 4, "AWE-10", "success", "10", 0)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        3,
		MaxExpeditions: 3,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	logDir := filepath.Join(dir, ".expedition", ".run", "logs")

	// Should have prompts for expeditions 5, 6, 7 (not 1, 2, 3)
	for _, expNum := range []int{5, 6, 7} {
		promptFile := filepath.Join(logDir, fmt.Sprintf("expedition-%03d-prompt.md", expNum))
		if _, err := os.Stat(promptFile); errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected prompt file for expedition %d, not found", expNum)
		}
	}

	// Should NOT have prompts for expeditions 1-4
	for _, expNum := range []int{1, 2, 3, 4} {
		promptFile := filepath.Join(logDir, fmt.Sprintf("expedition-%03d-prompt.md", expNum))
		if _, err := os.Stat(promptFile); !errors.Is(err, fs.ErrNotExist) {
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

	cfg := domain.Config{
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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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

	cfg := domain.Config{
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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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

	cfg := domain.Config{
		Continent:      dir,
		Workers:        1, // single worker WITH worktree pool
		MaxExpeditions: 2,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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

	// Create a script that outputs __EXPEDITION_COMPLETE__ as stream-json
	completeScript := streamJSONScript(t, dir, "complete.sh", "__EXPEDITION_COMPLETE__")

	cfg := domain.Config{
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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
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

	cfg := domain.Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 2,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)

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

	cfg := domain.Config{
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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	p.Run(context.Background())

	// Journal entry should exist for expedition 1
	journalPath := filepath.Join(dir, ".expedition", "journal", "001.md")
	if _, err := os.Stat(journalPath); errors.Is(err, fs.ErrNotExist) {
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
	cfg := domain.Config{Continent: dir, BaseBranch: "main", Model: "opus"}
	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)

	// Write flag for expedition 5
	p.writeFlag(dir, 5, "ISS-5", "success", "10", 0)

	// Attempt to write flag for expedition 3 (out-of-order completion)
	p.writeFlag(dir, 3, "ISS-3", "success", "12", 0)

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

	cfg := domain.Config{
		Continent:      dir,
		MaxExpeditions: 1,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		NoDev:          true,
		DryRun:         true,
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)

	// devServer should be nil — no panic during Run
	if p.devServer != nil {
		t.Fatal("devServer should be nil when NoDev=true")
	}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}
}

func TestFormatSummaryJSON(t *testing.T) {
	// given
	summary := domain.RunSummary{
		Total:    5,
		Success:  4,
		Skipped:  0,
		Failed:   1,
		Bugs:     0,
		Gradient: "3/5",
	}

	// when
	out, err := domain.FormatSummaryJSON(summary)
	if err != nil {
		t.Fatalf("FormatSummaryJSON: %v", err)
	}

	// then — must be valid JSON
	var parsed domain.RunSummary
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, out)
	}
	if parsed.Total != 5 {
		t.Errorf("total = %d, want 5", parsed.Total)
	}
	if parsed.Success != 4 {
		t.Errorf("success = %d, want 4", parsed.Success)
	}
	if parsed.Failed != 1 {
		t.Errorf("failed = %d, want 1", parsed.Failed)
	}
	if parsed.Gradient != "3/5" {
		t.Errorf("gradient = %q, want %q", parsed.Gradient, "3/5")
	}
}

func TestFormatSummaryJSON_MidHighSeverity(t *testing.T) {
	// given
	summary := domain.RunSummary{
		Total:           3,
		Success:         2,
		Failed:          1,
		MidHighSeverity: 4,
		Gradient:        "2/3",
	}

	// when
	out, err := domain.FormatSummaryJSON(summary)
	if err != nil {
		t.Fatalf("FormatSummaryJSON: %v", err)
	}

	// then — mid_high_severity must appear in JSON
	var parsed domain.RunSummary
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, out)
	}
	if parsed.MidHighSeverity != 4 {
		t.Errorf("mid_high_severity = %d, want 4", parsed.MidHighSeverity)
	}
}

// ===============================================
// Workers>1 Integration Tests (MY-362 gap fill)
// ===============================================

// TestSwarmMode_TwoWorkers_Consolidation verifies that post-run consolidation
// writes the max(LastExpedition) from per-worker worktree flag.md files back
// to Continent's flag.md. Workers=2, MaxExpeditions=2: each worker runs 1
// expedition, then reconcileFlags picks the highest and consolidates.
func TestSwarmMode_TwoWorkers_Consolidation(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// fakeClaude outputs a valid success report as stream-json
	reportText := `__EXPEDITION_REPORT__
issue_id: TEST-1
issue_title: consolidation test
mission_type: implement
branch: none
pr_url: none
status: success
reason: done
remaining_issues: 3
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`
	script := streamJSONScript(t, dir, "fakeclaude.sh", reportText)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        2,
		MaxExpeditions: 2,
		BaseBranch:     "main",
		ClaudeCmd:      script,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Consolidation: Continent flag.md should have the highest expedition number
	flag := ReadFlag(dir)
	if flag.LastExpedition < 2 {
		t.Errorf("consolidation: expected LastExpedition >= 2, got %d", flag.LastExpedition)
	}
	if flag.LastStatus != "success" {
		t.Errorf("expected LastStatus=success, got %q", flag.LastStatus)
	}
	if flag.Remaining != "3" {
		t.Errorf("expected Remaining=3, got %q", flag.Remaining)
	}

	// Both expeditions should have completed successfully
	if p.totalSuccess.Load() != 2 {
		t.Errorf("expected totalSuccess=2, got %d", p.totalSuccess.Load())
	}
}

// TestSwarmMode_TwoWorkers_ArchiveIdempotent verifies that when two workers
// both process the same inbox D-Mail and both attempt to archive it on success,
// idempotent ArchiveInboxDMail ensures no errors. One os.Rename succeeds; the
// second gets ENOENT, confirms the destination exists in archive, and returns nil.
func TestSwarmMode_TwoWorkers_ArchiveIdempotent(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// Place a D-Mail in inbox before start
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)
	dmailContent := "---\nname: shared-dmail\nkind: info\ndescription: shared test\n---\n\nShared body\n"
	if err := os.WriteFile(filepath.Join(inboxDir, "shared-dmail.md"), []byte(dmailContent), 0644); err != nil {
		t.Fatal(err)
	}

	// fakeClaude outputs a valid success report as stream-json
	archiveReportText := `__EXPEDITION_REPORT__
issue_id: TEST-1
issue_title: archive test
mission_type: implement
branch: none
pr_url: none
status: success
reason: done
remaining_issues: 5
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`
	script := streamJSONScript(t, dir, "fakeclaude.sh", archiveReportText)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        2,
		MaxExpeditions: 2,
		BaseBranch:     "main",
		ClaudeCmd:      script,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Inbox should be empty: both workers tried to archive, one succeeded,
	// the other got ENOENT but confirmed dst in archive (idempotent nil).
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".md" {
			t.Errorf("inbox should be empty after archive, found: %s", e.Name())
		}
	}

	// Archive should contain the D-Mail
	archivePath := filepath.Join(dir, ".expedition", "archive", "shared-dmail.md")
	if _, err := os.Stat(archivePath); errors.Is(err, fs.ErrNotExist) {
		t.Error("expected shared-dmail.md in archive/")
	}
}

// TestSwarmMode_TwoWorkers_MidHighSeverityAggregation verifies that
// totalMidHighSeverity correctly aggregates HIGH severity D-Mails detected
// mid-expedition across multiple workers. Each worker's script writes a unique
// HIGH severity D-Mail during execution; the inbox watcher detects them.
func TestSwarmMode_TwoWorkers_MidHighSeverityAggregation(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	// Script writes a unique HIGH severity D-Mail (using PID for uniqueness)
	// to Continent's inbox mid-execution, waits for watcher, then outputs report.
	highReportText := `__EXPEDITION_REPORT__
issue_id: TEST-1
issue_title: high severity test
mission_type: implement
branch: none
pr_url: none
status: success
reason: done
remaining_issues: 3
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`
	highAssistant := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"id": "msg_test", "role": "assistant", "model": "claude-sonnet-4-20250514",
			"content": []map[string]any{{"type": "text", "text": highReportText}},
		},
	}
	highResult := map[string]any{"type": "result", "result": highReportText}
	highAJSON, _ := json.Marshal(highAssistant)
	highRJSON, _ := json.Marshal(highResult)

	script := filepath.Join(dir, "fakeclaude-high.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
DMAIL_NAME="high-$$"
cat > %s/$DMAIL_NAME.md << DMEOF
---
name: $DMAIL_NAME
kind: alert
description: high severity test
severity: high
---

High severity body
DMEOF
# Wait for inbox watcher (fsnotify) to detect the new file
sleep 2
echo '%s'
echo '%s'
`, inboxDir, string(highAJSON), string(highRJSON))
	if err := os.WriteFile(script, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := domain.Config{
		Continent:      dir,
		Workers:        2,
		MaxExpeditions: 2,
		BaseBranch:     "main",
		ClaudeCmd:      script,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Each worker should have detected at least 1 HIGH severity D-Mail.
	// Both watchers may see both D-Mails (timing-dependent), so total >= 2.
	total := p.totalMidHighSeverity.Load()
	if total < 2 {
		t.Errorf("expected totalMidHighSeverity >= 2, got %d", total)
	}

	// Flag should reflect the mid-high severity count
	flag := ReadFlag(dir)
	if flag.MidHighSeverity == 0 {
		t.Error("consolidated flag.md should have MidHighSeverity > 0")
	}
}

// TestSwarmMode_TwoWorkers_StatusComplete_WritesFlag verifies that the
// StatusComplete path writes the "all/complete" flag checkpoint before
// releasing the worktree back to the pool. If writeFlag ran after
// releaseWorkDir, another worker could reclaim the worktree and overwrite
// the flag.md, losing the completion checkpoint.
func TestSwarmMode_TwoWorkers_StatusComplete_WritesFlag(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// fakeClaude outputs __EXPEDITION_COMPLETE__ which triggers StatusComplete
	script := streamJSONScript(t, dir, "fakeclaude.sh", "__EXPEDITION_COMPLETE__")

	cfg := domain.Config{
		Continent:      dir,
		Workers:        2,
		MaxExpeditions: 2,
		BaseBranch:     "main",
		ClaudeCmd:      script,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Consolidated flag must reflect the StatusComplete checkpoint.
	// If writeFlag ran after releaseWorkDir (the bug), reconcileFlags
	// could miss the "complete" status or see stale data from a reused worktree.
	flag := ReadFlag(dir)
	if flag.LastExpedition == 0 {
		t.Fatal("consolidated flag.md has LastExpedition=0; writeFlag may not have run before release")
	}
	if flag.LastStatus != "complete" {
		t.Errorf("expected LastStatus=complete, got %q", flag.LastStatus)
	}
	if flag.LastIssue != "all" {
		t.Errorf("expected LastIssue=all, got %q", flag.LastIssue)
	}
	if flag.Remaining != "0" {
		t.Errorf("expected Remaining=0, got %q", flag.Remaining)
	}
}

// TestSwarmMode_StaleWorktreeFlag_IgnoredAfterInit verifies that stale
// flag.md files from a crashed prior run do not advance the resume point.
// WorktreePool.Init force-removes old worktrees (and their flag.md files),
// and reconcileFlags runs after Init, so stale checkpoints are invisible.
func TestSwarmMode_StaleWorktreeFlag_IgnoredAfterInit(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// Plant a stale worktree with a real git worktree and a flag.md at exp 99
	// to simulate a prior crash that left behind worktree state.
	stalePath := filepath.Join(dir, ".expedition", ".run", "worktrees", "worker-001")
	cmd := exec.Command("git", "worktree", "add", "--detach", stalePath, "main")
	cmd.Dir = dir
	cmd.Env = gitIsolatedEnv(dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}
	WriteFlag(stalePath, 99, "STALE-1", "success", "0", 0)

	// Continent's own flag at exp 2 (the real checkpoint)
	WriteFlag(dir, 2, "MY-1", "success", "5", 0)

	// fakeClaude outputs a success report — if stale flag is read,
	// startExp would be 100 and this expedition would not run.
	staleReportText := `__EXPEDITION_REPORT__
issue_id: MY-2
issue_title: not stale
mission_type: implement
branch: none
pr_url: none
status: success
reason: done
remaining_issues: 4
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`
	script := streamJSONScript(t, dir, "fakeclaude.sh", staleReportText)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 3, // startExp=3 from continent flag (exp 2), runs exp 3,4,5
		BaseBranch:     "main",
		ClaudeCmd:      script,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	var logBuf bytes.Buffer
	p := NewPaintress(cfg, platform.NewLogger(&logBuf, false), &logBuf, io.Discard, nil, nil)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d\nlog:\n%s", code, logBuf.String())
	}

	// If the stale flag (exp 99) was read, startExp would be 100 and
	// nothing would run (MaxExpeditions=3 < 100). The test proves
	// reconcileFlags ignores stale worktree flags after Init cleans them.
	if p.totalSuccess.Load() == 0 {
		t.Fatal("no expeditions ran; stale worktree flag.md likely advanced startExp past MaxExpeditions")
	}

	flag := ReadFlag(dir)
	// startExp = continent flag (2) + 1 = 3, runs 3 expeditions: exp 3, 4, 5
	if flag.LastExpedition != 5 {
		t.Errorf("expected LastExpedition=5, got %d", flag.LastExpedition)
	}
	if flag.LastIssue != "MY-2" {
		t.Errorf("expected LastIssue=MY-2, got %q", flag.LastIssue)
	}
}

// TestStatusComplete_ArchivesInboxDMails verifies that inbox D-Mails are
// archived even when the expedition returns StatusComplete (all issues done).
// Regression test: previously archive only happened in StatusSuccess, causing
// an infinite loop in waiting mode where the same D-Mails were re-read.
//
// Golden test data: testdata/implementation-feedback-057.md (real D-Mail from
// a production vsano expedition that triggered the infinite loop).
func TestStatusComplete_ArchivesInboxDMails(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// Place golden D-Mail in inbox
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	golden, err := os.ReadFile("testdata/implementation-feedback-057.md")
	if err != nil {
		t.Fatalf("read golden testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "implementation-feedback-057.md"), golden, 0644); err != nil {
		t.Fatal(err)
	}

	// fakeClaude outputs __EXPEDITION_COMPLETE__ (no remaining issues)
	script := streamJSONScript(t, dir, "fakeclaude.sh", "__EXPEDITION_COMPLETE__")

	cfg := domain.Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 1,
		BaseBranch:     "main",
		ClaudeCmd:      script,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Inbox must be empty: D-Mail should be archived after StatusComplete
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".md" {
			t.Errorf("inbox should be empty after StatusComplete, found: %s", e.Name())
		}
	}

	// Archive must contain the D-Mail
	archivePath := filepath.Join(dir, ".expedition", "archive", "implementation-feedback-057.md")
	if _, err := os.Stat(archivePath); errors.Is(err, fs.ErrNotExist) {
		t.Error("expected implementation-feedback-057.md in archive/, but not found")
	}
}

// TestExpeditionError_ArchivesInboxDMails verifies that inbox D-Mails are
// archived even when the expedition process fails (handleExpeditionError path).
// Regression: early return from handleExpeditionError bypassed archive logic.
func TestExpeditionError_ArchivesInboxDMails(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// Place golden D-Mail in inbox
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	golden, err := os.ReadFile("testdata/implementation-feedback-057.md")
	if err != nil {
		t.Fatalf("read golden testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "implementation-feedback-057.md"), golden, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := domain.Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 1,
		BaseBranch:     "main",
		ClaudeCmd:      "/bin/false", // always fails → handleExpeditionError path
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	_ = p.Run(context.Background())

	// Inbox must be empty: D-Mail should be archived even on error path
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".md" {
			t.Errorf("inbox should be empty after expedition error, found: %s", e.Name())
		}
	}

	// Archive must contain the D-Mail
	archivePath := filepath.Join(dir, ".expedition", "archive", "implementation-feedback-057.md")
	if _, err := os.Stat(archivePath); errors.Is(err, fs.ErrNotExist) {
		t.Error("expected implementation-feedback-057.md in archive/, but not found")
	}
}

// TestGommage_ArchivesInboxDMails verifies that inbox D-Mails are archived
// even when the worker exits via errGommage (consecutive failure limit).
// Regression: return errGommage bypassed archive logic in dispatchExpeditionResult.
func TestGommage_ArchivesInboxDMails(t *testing.T) {
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	// Place golden D-Mail in inbox
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	golden, err := os.ReadFile("testdata/implementation-feedback-057.md")
	if err != nil {
		t.Fatalf("read golden testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "implementation-feedback-057.md"), golden, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := domain.Config{
		Continent:      dir,
		Workers:        1,
		MaxExpeditions: 20, // enough to trigger gommage
		BaseBranch:     "main",
		ClaudeCmd:      "/bin/false", // always fails → consecutive failures → gommage
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(context.Background())

	if code != 1 {
		t.Fatalf("expected exit code 1 (gommage), got %d", code)
	}

	// Inbox must be empty: D-Mail should be archived even on gommage path
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".md" {
			t.Errorf("inbox should be empty after gommage, found: %s", e.Name())
		}
	}

	// Archive must contain the D-Mail
	archivePath := filepath.Join(dir, ".expedition", "archive", "implementation-feedback-057.md")
	if _, err := os.Stat(archivePath); errors.Is(err, fs.ErrNotExist) {
		t.Error("expected implementation-feedback-057.md in archive/, but not found")
	}
}

// ===============================================
// Infinite Loop Audit — Termination Proof Tests
// ===============================================

// TestTermination_MaxExpeditions_CeilingStopsWorker proves that runWorker
// terminates after exactly MaxExpeditions, even when every expedition succeeds
// (the "keep going" happy path). This is the non-DryRun public API proof that
// the atomic counter ceiling `exp >= startExp + MaxExpeditions` in
// paintress_expedition.go:33 prevents an infinite expedition loop.
func TestTermination_MaxExpeditions_CeilingStopsWorker(t *testing.T) {
	// given: non-DryRun with MaxExpeditions=2, Claude always returns success
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	reportText := `__EXPEDITION_REPORT__
issue_id: TERM-1
issue_title: termination proof
mission_type: implement
branch: none
pr_url: none
status: success
reason: done
remaining_issues: 99
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`
	script := streamJSONScript(t, dir, "fakeclaude.sh", reportText)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0, // single worker
		MaxExpeditions: 2,
		BaseBranch:     "main",
		ClaudeCmd:      script,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	// when
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(ctx)

	// then: Run() must terminate within the test timeout (not loop forever)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if p.totalAttempted.Load() != 2 {
		t.Errorf("expected exactly 2 expeditions (MaxExpeditions=2), got %d", p.totalAttempted.Load())
	}
	if p.totalSuccess.Load() != 2 {
		t.Errorf("expected 2 successes, got %d", p.totalSuccess.Load())
	}
}

// TestTermination_ReviewLoop_CycleCapStopsLoop proves that the review loop
// terminates after maxReviewGateCycles even when the review command always
// fails (always finds comments). This is the public API proof via Run() that
// review.go:103 `for cycle := 1; cycle <= maxReviewGateCycles` prevents an
// infinite review-fix cycle.
func TestTermination_ReviewLoop_CycleCapStopsLoop(t *testing.T) {
	// given: Claude returns success with a PR URL, review always fails
	dir := setupTestRepo(t)
	setupGitRepoWithBranch(t, dir, "feat/term-test")

	srv := newTestServer(t)
	defer srv.Close()

	reportText := `__EXPEDITION_REPORT__
issue_id: TERM-2
issue_title: review termination proof
mission_type: implement
branch: feat/term-test
pr_url: https://github.com/test/test/pull/1
status: success
reason: done
remaining_issues: 5
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`
	claudeScript := streamJSONScript(t, dir, "fakeclaude.sh", reportText)

	// Review script always exits 1 with comments (never passes)
	reviewScript := filepath.Join(dir, "review.sh")
	writeScript(t, reviewScript, "echo '[P2] Always fails'\nexit 1\n")

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		BaseBranch:     "main",
		ClaudeCmd:      claudeScript,
		ReviewCmd:      reviewScript,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	// when
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(ctx)

	// then: Run() must terminate (review loop bounded by maxReviewGateCycles)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if p.totalSuccess.Load() != 1 {
		t.Errorf("expected 1 success, got %d", p.totalSuccess.Load())
	}

	// Journal should record the review not being resolved
	journalPath := filepath.Join(dir, ".expedition", "journal", "001.md")
	content, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("journal not written: %v", err)
	}
	if !containsStr(string(content), "Review not fully resolved") && !containsStr(string(content), "Review") {
		t.Errorf("journal should mention review status, got: %s", string(content))
	}
}

// TestTermination_ConsecutiveSkips_StopsWorker proves that the expedition loop
// terminates when all expeditions are consecutively skipped (e.g. all issues
// are In Review). Without this guard, the loop burns Claude API + Linear API
// tokens indefinitely on each 10-second cycle. This is the public API proof
// via Run() that consecutive skips trigger early exit, analogous to the
// gommage pattern for consecutive failures.
func TestSkipReview_RunsReviewOnPastPRs(t *testing.T) {
	// given: a repo with a past PR in pr-index.jsonl
	dir := setupTestRepo(t)
	srv := newTestServer(t)
	defer srv.Close()

	// Plant a past PR in the index
	pastReport := &domain.ExpeditionReport{
		Expedition: 1,
		IssueID:    "AWE-10",
		PRUrl:      "https://github.com/org/repo/pull/10",
	}
	if err := WritePRIndex(dir, pastReport); err != nil {
		t.Fatalf("WritePRIndex: %v", err)
	}

	// review_cmd: script that creates a marker file and exits 0 (pass)
	reviewMarker := filepath.Join(dir, "review-was-called.marker")
	reviewScript := filepath.Join(dir, "review.sh")
	writeScript(t, reviewScript, fmt.Sprintf("touch %q\nexit 0\n", reviewMarker))

	// Claude returns "skipped" — all issues In Review
	reportText := `__EXPEDITION_REPORT__
issue_id: SKIP-1
issue_title: all blocked on review
mission_type: implement
branch: none
pr_url: none
status: skipped
reason: All issues are In Review with open PRs. No actionable work available.
remaining_issues: 5
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`
	script := streamJSONScript(t, dir, "fakeclaude.sh", reportText)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		BaseBranch:     "main",
		ClaudeCmd:      script,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
		ReviewCmd:      reviewScript,
	}

	// when
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	p.Run(ctx)

	// then: review_cmd should have been called
	if _, err := os.Stat(reviewMarker); os.IsNotExist(err) {
		t.Error("review_cmd was not executed during skip — re-review path not triggered")
	}
}

func TestTermination_ConsecutiveSkips_StopsWorker(t *testing.T) {
	// given: Claude always returns status=skipped (simulating all issues In Review)
	dir := setupTestRepo(t)

	srv := newTestServer(t)
	defer srv.Close()

	reportText := `__EXPEDITION_REPORT__
issue_id: SKIP-1
issue_title: all blocked on review
mission_type: implement
branch: none
pr_url: none
status: skipped
reason: All issues are In Review with open PRs. No actionable work available.
remaining_issues: 5
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`
	script := streamJSONScript(t, dir, "fakeclaude.sh", reportText)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 20, // high ceiling — should NOT reach this
		BaseBranch:     "main",
		ClaudeCmd:      script,
		DevCmd:         "true",
		DevURL:         srv.URL,
		TimeoutSec:     30,
		Model:          "opus",
	}

	// when
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil)
	code := p.Run(ctx)

	// then: Run() must terminate via consecutive skip detection, not MaxExpeditions
	if p.totalSkipped.Load() >= 20 {
		t.Fatalf("loop ran all 20 expeditions — consecutive skip termination did not fire")
	}
	if p.totalSkipped.Load() < 3 {
		t.Errorf("expected at least 3 skips before termination, got %d", p.totalSkipped.Load())
	}
	// exit code 1 indicates abnormal termination (like gommage)
	if code != 1 {
		t.Errorf("expected exit code 1 (consecutive skips), got %d", code)
	}
}
