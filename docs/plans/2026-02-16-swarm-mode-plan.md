# Swarm Mode Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Parallelize expedition execution with N worker goroutines, each running in an isolated worktree.

**Architecture:** Fan-out N goroutines from `Paintress.Run()` via `errgroup`. Each worker loops independently: claim atomic expedition number, acquire worktree, run expedition, process result, release worktree. Shared state synchronized via `sync/atomic` counters and `sync.Mutex`.

**Tech Stack:** Go 1.26, `golang.org/x/sync/errgroup`, `sync/atomic.Int64`, `sync.Mutex`

**Design Doc:** `docs/plans/2026-02-16-swarm-mode-design.md`

---

### Task 1: Add errgroup dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: Add golang.org/x/sync**

Run:
```bash
cd /Users/nino/paintress && go get golang.org/x/sync
```

**Step 2: Verify**

Run: `go build ./...`
Expected: BUILD OK

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: add golang.org/x/sync for errgroup"
```

---

### Task 2: [STRUCTURAL] Convert Paintress counters to atomic + add mutex fields

This is a Tidy First structural change. All existing behavior is preserved — the only difference is that counters use atomic operations and a mutex protects Flag writes. Tests must pass before and after.

**Files:**
- Modify: `paintress.go:16-28` (struct definition)
- Modify: `paintress.go:63-304` (Run method — counter access sites)
- Modify: `paintress.go:450-468` (handleSuccess — totalBugs)
- Modify: `paintress.go:478-499` (printSummary — counter reads)

**Step 1: Run existing tests to confirm baseline**

Run: `go test ./... -count=1 -run 'Test[^W]' -timeout=60s`
Expected: all `review_loop_test.go` tests PASS (skip container tests with `[^W]` prefix filter)

Also run: `go vet ./...`
Expected: clean

**Step 2: Modify Paintress struct**

In `paintress.go`, change the struct definition from:

```go
type Paintress struct {
	config    Config
	logDir    string
	devServer *DevServer
	gradient  *GradientGauge
	reserve   *ReserveParty
	pool      *WorktreePool // nil when --workers=0

	totalSuccess int
	totalSkipped int
	totalFailed  int
	totalBugs    int
}
```

To:

```go
type Paintress struct {
	config    Config
	logDir    string
	devServer *DevServer
	gradient  *GradientGauge
	reserve   *ReserveParty
	pool      *WorktreePool // nil when --workers=0

	// Swarm Mode: atomic counters for concurrent worker access
	expCounter          atomic.Int64 // global expedition number generator
	totalSuccess        atomic.Int64
	totalSkipped        atomic.Int64
	totalFailed         atomic.Int64
	totalBugs           atomic.Int64
	consecutiveFailures atomic.Int64

	// Swarm Mode: mutex-protected shared resources
	flagMu sync.Mutex
}
```

Add imports: `"sync"` and `"sync/atomic"` to the import block.

Note: `"sync"` might already be imported elsewhere. Check before adding duplicates.

**Step 3: Update all counter access sites in Run()**

Replace every counter read/write in `paintress.go`. The local variable `consecutiveFailures` is removed — use the struct field instead.

Changes in `Run()`:

1. Remove `consecutiveFailures := 0` (line ~124) — struct field starts at zero.

2. Every `p.totalFailed++` → `p.totalFailed.Add(1)`
   - In the `err != nil` block (~line 221)
   - In `StatusParseError` (~line 238)
   - In `StatusFailed` (~line 269)

3. Every `consecutiveFailures++` → `p.consecutiveFailures.Add(1)`
   - In the `err != nil` block (~line 220)
   - In `StatusParseError` (~line 237)
   - In `StatusFailed` (~line 268)

4. `consecutiveFailures = 0` → `p.consecutiveFailures.Store(0)` (in `StatusSuccess`, ~line 253)

5. `p.totalSuccess++` → `p.totalSuccess.Add(1)` (~line 254)

6. `p.totalSkipped++` → `p.totalSkipped.Add(1)` (~line 261)

7. `consecutiveFailures >= maxConsecutiveFailures` → `p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures)` (~line 274)

8. Wrap every `WriteFlag(...)` call with mutex:
   ```go
   p.flagMu.Lock()
   WriteFlag(p.config.Continent, exp, ...)
   p.flagMu.Unlock()
   ```
   There are 5 WriteFlag calls in Run(): lines ~214, ~229, ~251, ~259, ~265.

**Step 4: Update handleSuccess()**

In `handleSuccess()`:

```go
// Before:
p.totalBugs += report.BugsFound

// After:
p.totalBugs.Add(int64(report.BugsFound))
```

**Step 5: Update printSummary()**

Change `printSummary` to compute expedition count from atomics instead of taking a parameter:

```go
// Before:
func (p *Paintress) printSummary(expeditions int) {
	// ...
	LogInfo("%s", fmt.Sprintf(Msg("expeditions_sent"), expeditions))
	LogOK("%s", fmt.Sprintf(Msg("success_count"), p.totalSuccess))
	LogWarn("%s", fmt.Sprintf(Msg("skipped_count"), p.totalSkipped))
	LogError("%s", fmt.Sprintf(Msg("failed_count"), p.totalFailed))
	if p.totalBugs > 0 {
		LogQA("%s", fmt.Sprintf(Msg("bugs_count"), p.totalBugs))
	}
```

After:

```go
func (p *Paintress) printSummary() {
	total := p.totalSuccess.Load() + p.totalFailed.Load() + p.totalSkipped.Load()
	// ...
	LogInfo("%s", fmt.Sprintf(Msg("expeditions_sent"), total))
	LogOK("%s", fmt.Sprintf(Msg("success_count"), p.totalSuccess.Load()))
	LogWarn("%s", fmt.Sprintf(Msg("skipped_count"), p.totalSkipped.Load()))
	LogError("%s", fmt.Sprintf(Msg("failed_count"), p.totalFailed.Load()))
	if p.totalBugs.Load() > 0 {
		LogQA("%s", fmt.Sprintf(Msg("bugs_count"), p.totalBugs.Load()))
	}
```

Update all call sites of `printSummary` to remove the argument:
- `p.printSummary(exp - startExp)` → `p.printSummary()`
- `p.printSummary(exp - startExp + 1)` → `p.printSummary()`
- `p.printSummary(p.config.MaxExpeditions)` → `p.printSummary()`

**Step 6: Verify structural change preserves behavior**

Run: `go vet ./...`
Expected: clean

Run: `go test ./... -count=1 -run 'Test[^W]' -timeout=60s`
Expected: all tests PASS (same as Step 1)

**Step 7: Commit**

```bash
git add paintress.go
git commit -m "refactor: convert Paintress counters to atomic + add flag mutex

[STRUCTURAL] No behavioral change. Prepares for concurrent worker access
in Swarm Mode. All existing tests pass unchanged."
```

---

### Task 3: [BEHAVIORAL] Define sentinel errors for worker signaling

**Files:**
- Modify: `paintress.go` (add error variables + import `"errors"`)
- Create: `paintress_test.go` (new test file)

**Step 1: Write the failing test**

Create `paintress_test.go`:

```go
package main

import (
	"errors"
	"testing"
)

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
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -count=1 -run TestSentinelErrors -timeout=30s`
Expected: FAIL — `errGommage` and `errComplete` are undefined

**Step 3: Write minimal implementation**

In `paintress.go`, add at the top (after the const block):

```go
import "errors"

var (
	errGommage  = errors.New("gommage: consecutive failures exceeded threshold")
	errComplete = errors.New("expedition complete: no remaining issues")
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./... -count=1 -run TestSentinelErrors -timeout=30s`
Expected: PASS

**Step 5: Commit**

```bash
git add paintress.go paintress_test.go
git commit -m "feat: add sentinel errors for Swarm Mode worker signaling"
```

---

### Task 4: [BEHAVIORAL] Test + implement Swarm Mode DryRun (runWorker + errgroup)

This is the core task. Extract the expedition loop body into `runWorker()` and refactor `Run()` to launch workers via `errgroup`.

**Files:**
- Modify: `paintress.go:63-304` (Run method → orchestrator + runWorker)
- Modify: `paintress_test.go` (add DryRun integration test)

**Step 1: Write the failing test**

Add to `paintress_test.go`:

```go
import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a minimal git repo for Paintress tests.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "test"},
		{"git", "config", "commit.gpgsign", "false"},
		{"git", "checkout", "-b", "main"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup (%v) failed: %s", args, string(out))
		}
	}
	// Create .expedition/journal/ directory
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	return dir
}

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

	// Each of the 3 workers should create exactly 1 prompt file (DryRun exits after 1)
	logDir := filepath.Join(dir, ".expedition", ".logs")
	prompts, err := filepath.Glob(filepath.Join(logDir, "expedition-*-prompt.md"))
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(prompts) != 3 {
		t.Errorf("expected 3 prompt files, got %d: %v", len(prompts), prompts)
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
		Workers:        0, // direct execution, single goroutine
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

	logDir := filepath.Join(dir, ".expedition", ".logs")
	prompts, _ := filepath.Glob(filepath.Join(logDir, "expedition-*-prompt.md"))
	if len(prompts) != 1 {
		t.Errorf("expected 1 prompt file, got %d", len(prompts))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -count=1 -run 'TestSwarmMode_DryRun' -timeout=120s`
Expected: FAIL — with Workers=3, current sequential code creates only 1 prompt file (first worker writes, breaks, loop ends).

Note: The single-worker test might actually pass since it matches current behavior.

**Step 3: Implement runWorker()**

Extract the for-loop body from `Run()` into a new method. This is the expedition loop that each worker goroutine executes.

Add import: `"golang.org/x/sync/errgroup"`

```go
func (p *Paintress) runWorker(ctx context.Context, workerID int, startExp int, luminas []Lumina) error {
	for {
		// Check if another worker triggered stop (gommage/complete/signal)
		if ctx.Err() != nil {
			return nil
		}

		// Claim expedition number atomically
		exp := int(p.expCounter.Add(1) - 1)
		if exp >= startExp+p.config.MaxExpeditions {
			return nil // budget exhausted
		}

		LogExp("%s", fmt.Sprintf(Msg("departing"), exp))

		// Reserve recovery (thread-safe)
		p.reserve.TryRecoverPrimary()
		LogInfo("%s", fmt.Sprintf(Msg("gradient_info"), p.gradient.FormatForPrompt()))
		LogInfo("%s", fmt.Sprintf(Msg("party_info"), p.reserve.Status()))

		// Acquire worktree (blocks until available)
		var workDir string
		if p.pool != nil {
			workDir = p.pool.Acquire()
		}
		releaseWorkDir := func() {
			if p.pool != nil && workDir != "" {
				rCtx, rCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer rCancel()
				if err := p.pool.Release(rCtx, workDir); err != nil {
					LogWarn("worktree release: %v", err)
				}
				workDir = ""
			}
		}

		expedition := &Expedition{
			Number:    exp,
			Continent: p.config.Continent,
			WorkDir:   workDir,
			Config:    p.config,
			LogDir:    p.logDir,
			Luminas:   luminas,
			Gradient:  p.gradient,
			Reserve:   p.reserve,
		}

		// DryRun: write prompt and return
		if p.config.DryRun {
			promptFile := filepath.Join(p.logDir, fmt.Sprintf("expedition-%03d-prompt.md", exp))
			os.WriteFile(promptFile, []byte(expedition.BuildPrompt()), 0644)
			LogWarn("%s", fmt.Sprintf(Msg("dry_run_prompt"), promptFile))
			releaseWorkDir()
			return nil // DryRun: each worker produces one prompt and exits
		}

		// Run expedition
		LogInfo("%s", fmt.Sprintf(Msg("sending"), p.reserve.ActiveModel()))
		expStart := time.Now()
		output, err := expedition.Run(ctx)
		expElapsed := time.Since(expStart)

		// Process result
		if err != nil {
			if ctx.Err() == context.Canceled {
				releaseWorkDir()
				return nil
			}

			LogError("%s", fmt.Sprintf(Msg("exp_failed"), exp, err))
			if strings.Contains(err.Error(), "timeout") {
				p.reserve.ForceReserve()
			}
			p.gradient.Discharge()

			p.flagMu.Lock()
			WriteFlag(p.config.Continent, exp, "error", "failed", "?")
			p.flagMu.Unlock()

			WriteJournal(p.config.Continent, &ExpeditionReport{
				Expedition: exp, IssueID: "?", IssueTitle: "?",
				MissionType: "?", Status: "failed", Reason: err.Error(),
				PRUrl: "none", BugIssues: "none",
			})
			p.consecutiveFailures.Add(1)
			p.totalFailed.Add(1)
		} else {
			report, status := ParseReport(output, exp)

			switch status {
			case StatusComplete:
				releaseWorkDir()
				LogOK("%s", Msg("all_complete"))
				p.flagMu.Lock()
				WriteFlag(p.config.Continent, exp, "all", "complete", "0")
				p.flagMu.Unlock()
				return errComplete

			case StatusParseError:
				LogWarn("%s", Msg("report_parse_fail"))
				LogWarn("%s", fmt.Sprintf(Msg("output_check"), p.logDir, exp))
				p.gradient.Decay()
				p.consecutiveFailures.Add(1)
				p.totalFailed.Add(1)

			case StatusSuccess:
				p.handleSuccess(report)
				p.gradient.Charge()
				if report.PRUrl != "" && report.PRUrl != "none" && p.config.ReviewCmd != "" {
					totalTimeout := time.Duration(p.config.TimeoutSec) * time.Second
					remaining := totalTimeout - expElapsed
					if remaining > 0 {
						p.runReviewLoop(ctx, report, remaining, workDir)
					}
				}
				p.flagMu.Lock()
				WriteFlag(p.config.Continent, exp, report.IssueID, "success", report.Remaining)
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, report)
				p.consecutiveFailures.Store(0)
				p.totalSuccess.Add(1)

			case StatusSkipped:
				LogWarn("%s", fmt.Sprintf(Msg("issue_skipped"), report.IssueID, report.Reason))
				p.gradient.Decay()
				p.flagMu.Lock()
				WriteFlag(p.config.Continent, exp, report.IssueID, "skipped", report.Remaining)
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, report)
				p.totalSkipped.Add(1)

			case StatusFailed:
				LogError("%s", fmt.Sprintf(Msg("issue_failed"), report.IssueID, report.Reason))
				p.gradient.Discharge()
				p.flagMu.Lock()
				WriteFlag(p.config.Continent, exp, report.IssueID, "failed", report.Remaining)
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, report)
				p.consecutiveFailures.Add(1)
				p.totalFailed.Add(1)
			}
		}

		// Gommage check
		if p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures) {
			releaseWorkDir()
			LogError("%s", fmt.Sprintf(Msg("gommage"), maxConsecutiveFailures))
			return errGommage
		}

		// Release worktree
		releaseWorkDir()
		if p.pool == nil {
			gitCmd := exec.CommandContext(ctx, "git", "checkout", p.config.BaseBranch)
			gitCmd.Dir = p.config.Continent
			_ = gitCmd.Run()
		}

		// Cooldown
		LogInfo("%s", Msg("cooldown"))
		select {
		case <-time.After(10 * time.Second):
		case <-ctx.Done():
			return nil
		}
	}
}
```

**Step 4: Refactor Run() to use errgroup**

Replace the expedition for-loop in `Run()` with errgroup-based worker launch:

```go
func (p *Paintress) Run(ctx context.Context) int {
	// ... existing init code (log, mission, banner) unchanged ...

	monolith := ReadFlag(p.config.Continent)

	// ... existing banner/logging unchanged ...

	// Start dev server (unchanged)
	if !p.config.DryRun {
		if err := p.devServer.Start(ctx); err != nil {
			LogWarn("%s", fmt.Sprintf(Msg("devserver_warn"), err))
		}
		defer p.devServer.Stop()
	}

	// Initialize worktree pool (unchanged)
	if p.config.Workers > 0 {
		p.pool = NewWorktreePool(
			&localGitExecutor{},
			p.config.Continent,
			p.config.BaseBranch,
			p.config.SetupCmd,
			p.config.Workers,
		)
		if err := p.pool.Init(ctx); err != nil {
			LogError("worktree pool init failed: %v", err)
			return 1
		}
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()
			p.pool.Shutdown(shutdownCtx)
		}()
	}

	// === Swarm Mode: launch workers ===
	startExp := monolith.LastExpedition + 1
	p.expCounter.Store(int64(startExp))

	// Pre-flight Lumina scan (once, before workers start)
	luminas := ScanJournalsForLumina(p.config.Continent)
	if len(luminas) > 0 {
		LogOK("%s", fmt.Sprintf(Msg("lumina_extracted"), len(luminas)))
		WriteLumina(p.config.Continent, luminas)
	}

	g, gCtx := errgroup.WithContext(ctx)
	workerCount := max(p.config.Workers, 1)

	for i := range workerCount {
		g.Go(func() error {
			return p.runWorker(gCtx, i, startExp, luminas)
		})
	}

	err := g.Wait()

	fmt.Println()
	p.printSummary()

	// Determine exit code from sentinel errors
	switch {
	case errors.Is(err, errComplete):
		return 0
	case errors.Is(err, errGommage):
		return 1
	case ctx.Err() == context.Canceled:
		return 130
	case err != nil:
		return 1
	default:
		return 0
	}
}
```

**Important:** Remove the old for-loop, `consecutiveFailures` local variable, and the Lumina scan that was inside the loop. The Lumina scan moves to before worker launch.

**Step 5: Run test to verify it passes**

Run: `go test ./... -count=1 -run 'TestSwarmMode_DryRun' -timeout=120s`
Expected: PASS — both tests (3 workers = 3 prompts, single worker = 1 prompt)

Run: `go vet ./...`
Expected: clean

Run: `go test ./... -count=1 -run 'Test[^W]' -timeout=60s`
Expected: existing review_loop tests still PASS

**Step 6: Commit**

```bash
git add paintress.go paintress_test.go
git commit -m "feat: implement Swarm Mode parallel expedition workers

Extract runWorker() loop from Run(). Launch N goroutines via errgroup.
Each worker independently claims atomic expedition numbers, acquires
worktrees, runs expeditions, and processes results.

Sentinel errors (errGommage, errComplete) signal cross-worker stops."
```

---

### Task 5: [BEHAVIORAL] Test + implement gommage in parallel mode

Verify that consecutive failures from multiple parallel workers trigger gommage correctly.

**Files:**
- Modify: `paintress_test.go` (add gommage test)

**Step 1: Write the failing test**

Add to `paintress_test.go`:

```go
import "net/http/httptest"

func TestSwarmMode_Gommage_StopsAllWorkers(t *testing.T) {
	dir := setupTestRepo(t)

	// Start a trivial HTTP server so DevServer.Start() succeeds immediately
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
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
```

**Step 2: Run test to verify it passes (or fails for the right reason)**

Run: `go test ./... -count=1 -run 'TestSwarmMode_Gommage' -timeout=120s`

If Task 4 was implemented correctly, this test should PASS because:
- `/bin/false` exits with code 1 → expedition.Run() returns error
- Each failure increments `consecutiveFailures`
- After 3 failures, gommage triggers → errGommage → exit code 1

If it fails, debug and fix.

**Step 3: Commit**

```bash
git add paintress_test.go
git commit -m "test: add Swarm Mode gommage integration test

Verifies parallel workers trigger gommage after consecutive failures
and stop early without running all expeditions."
```

---

### Task 6: Full verification

**Step 1: Run all non-container tests**

Run: `go test ./... -count=1 -run 'Test[^W]' -timeout=120s`
Expected: ALL tests PASS (review_loop + paintress tests)

**Step 2: Run container tests (optional, slower)**

Run: `go test ./... -count=1 -run 'TestWorktreePool' -timeout=300s`
Expected: ALL WorktreePool container tests PASS

**Step 3: Run go vet**

Run: `go vet ./...`
Expected: clean

**Step 4: Run gofmt**

Run: `gofmt -l .`
Expected: no files listed (all formatted)

**Step 5: Commit any fixes if needed**

If any formatting or vet issues:
```bash
gofmt -w .
git add -A
git commit -m "style: fix formatting"
```

---

## Execution Notes

- **Testing strategy**: Host-based tests only (no containers needed). Git worktrees are created in temp dirs by the test helper.
- **DevServer in tests**: Use `httptest.NewServer` to satisfy DevServer.Start()'s health check without running a real dev server.
- **ClaudeCmd in tests**: Use `/bin/false` (always fails) to test failure paths without real Claude CLI.
- **DryRun tests**: No Claude CLI, no DevServer, no review loop — just prompt file generation.
- **Container tests are unchanged**: WorktreePool tests in `worktree_test.go` are independent and unaffected.
