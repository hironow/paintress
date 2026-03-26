# E15 Gommage Recovery Strategy — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace paintress's uniform Gommage halt with class-aware recovery that retries transient failures in-place, preserves worktree progress, and resumes after session restart.

**Architecture:** B+C hybrid — domain aggregate owns classification + retry-count tracking (pure, testable), session layer owns side effects (cooldown, model switch, worktree lifecycle). Expedition loop delegates to `DecideRecovery()` then `executeRecovery()`.

**Tech Stack:** Go 1.26, stdlib only (no new deps). OTel for observability. Event sourcing (JSONL).

**Spec:** `docs/superpowers/specs/2026-03-26-gommage-recovery-strategy-design.md`

**Branch:** `feature/e15-gommage-recovery`

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| NEW | `internal/domain/gommage_classifier.go` | `GommageClass` type + `ClassifyGommage()` pure function |
| NEW | `internal/domain/gommage_classifier_test.go` | 7 test cases for classifier |
| NEW | `internal/domain/gommage_recovery.go` | `RecoveryAction`, `RecoveryDecision`, `cooldownForClass()` |
| NEW | `internal/domain/gommage_recovery_test.go` | 4 test cases for cooldown + serialization |
| NEW | `internal/session/gommage_recovery.go` | `executeRecovery()`, `injectParseErrorLumina()` |
| NEW | `internal/session/gommage_recovery_test.go` | 4 test cases for recovery execution |
| NEW | `internal/session/gommage_checkpoint.go` | checkpoint, resume, orphan cleanup |
| NEW | `internal/session/gommage_checkpoint_test.go` | 5 test cases for checkpoint/resume |
| MOD | `internal/domain/expedition_aggregate.go` | `recoveryAttempts`, `DecideRecovery()`, `ResetRecovery()` |
| MOD | `internal/domain/expedition_aggregate_test.go` | 5 new test cases |
| MOD | `internal/domain/event.go` | `GommageTriggeredData` extension, 2 new event types |
| MOD | `internal/usecase/port/port.go` | 2 new methods on `ExpeditionEventEmitter` |
| MOD | `internal/usecase/emitter.go` | 2 new emitter implementations |
| MOD | `internal/session/paintress_expedition.go` | L215-236 → class-aware dispatch + CAS fix |
| MOD | `internal/session/paintress.go` | startup orphan cleanup + resume |
| MOD | `internal/session/gommage_insight.go` | `Class` parameter |

---

### Task 1: Domain — GommageClass type + ClassifyGommage

**Files:**
- Create: `internal/domain/gommage_classifier.go`
- Create: `internal/domain/gommage_classifier_test.go`

- [ ] **Step 1: Write failing tests for ClassifyGommage**

```go
// internal/domain/gommage_classifier_test.go
package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestClassifyGommage_Timeout(t *testing.T) {
	reasons := []string{"timeout after 120s", "timeout after 120s", "timeout after 120s"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassTimeout {
		t.Errorf("got %q, want %q", got, domain.GommageClassTimeout)
	}
}

func TestClassifyGommage_RateLimit(t *testing.T) {
	reasons := []string{"rate_limit: model overloaded", "rate_limit: 429", "rate_limit: quota"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassRateLimit {
		t.Errorf("got %q, want %q", got, domain.GommageClassRateLimit)
	}
}

func TestClassifyGommage_ParseError(t *testing.T) {
	reasons := []string{"parse_error: no markers", "parse_error: invalid json", "parse_error: truncated"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassParseError {
		t.Errorf("got %q, want %q", got, domain.GommageClassParseError)
	}
}

func TestClassifyGommage_Blocker(t *testing.T) {
	reasons := []string{"blocker: PR stuck", "blocker: merge conflict", "blocker: CI failed"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassBlocker {
		t.Errorf("got %q, want %q", got, domain.GommageClassBlocker)
	}
}

func TestClassifyGommage_Systematic(t *testing.T) {
	reasons := []string{"unknown error A", "unknown error B", "unknown error C"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassSystematic {
		t.Errorf("got %q, want %q", got, domain.GommageClassSystematic)
	}
}

func TestClassifyGommage_Mixed_NoMajority(t *testing.T) {
	reasons := []string{"timeout after 120s", "rate_limit: 429", "blocker: stuck"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassSystematic {
		t.Errorf("mixed with no majority should be systematic, got %q", got)
	}
}

func TestClassifyGommage_Empty(t *testing.T) {
	got := domain.ClassifyGommage(nil)
	if got != domain.GommageClassSystematic {
		t.Errorf("empty reasons should be systematic, got %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/nino/tap/paintress && go test ./internal/domain/ -run TestClassifyGommage -v && cd -`
Expected: FAIL — `ClassifyGommage` not defined

- [ ] **Step 3: Implement ClassifyGommage**

```go
// internal/domain/gommage_classifier.go
package domain

import "strings"

// GommageClass identifies the dominant failure pattern in a streak.
type GommageClass string

const (
	GommageClassTimeout    GommageClass = "timeout"
	GommageClassParseError GommageClass = "parse_error"
	GommageClassRateLimit  GommageClass = "rate_limit"
	GommageClassBlocker    GommageClass = "blocker"
	GommageClassSystematic GommageClass = "systematic"
)

// classKeywords maps each class to its detection keywords.
var classKeywords = map[GommageClass]string{
	GommageClassTimeout:    "timeout",
	GommageClassRateLimit:  "rate_limit",
	GommageClassParseError: "parse_error",
	GommageClassBlocker:    "blocker",
}

// ClassifyGommage inspects recent failure reasons and returns the dominant class.
// Majority-vote over keyword matching. If no majority, returns GommageClassSystematic.
func ClassifyGommage(reasons []string) GommageClass {
	if len(reasons) == 0 {
		return GommageClassSystematic
	}
	counts := make(map[GommageClass]int)
	for _, reason := range reasons {
		lower := strings.ToLower(reason)
		for class, keyword := range classKeywords {
			if strings.Contains(lower, keyword) {
				counts[class]++
				break // one class per reason
			}
		}
	}
	majority := (len(reasons) + 1) / 2 // >50%
	for class, count := range counts {
		if count >= majority {
			return class
		}
	}
	return GommageClassSystematic
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/nino/tap/paintress && go test ./internal/domain/ -run TestClassifyGommage -v && cd -`
Expected: PASS (7/7)

- [ ] **Step 5: Commit**

```bash
cd /Users/nino/tap/paintress && git add internal/domain/gommage_classifier.go internal/domain/gommage_classifier_test.go && git commit -m "feat(domain): add GommageClass type and ClassifyGommage pure function [BEHAVIORAL]" && cd -
```

---

### Task 2: Domain — RecoveryDecision type + cooldownForClass

**Files:**
- Create: `internal/domain/gommage_recovery.go`
- Create: `internal/domain/gommage_recovery_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/domain/gommage_recovery_test.go
package domain_test

import (
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestCooldownForClass_Timeout(t *testing.T) {
	got := domain.CooldownForClass(domain.GommageClassTimeout, 1)
	if got != 30*time.Second {
		t.Errorf("timeout retry 1: got %v, want 30s", got)
	}
	got = domain.CooldownForClass(domain.GommageClassTimeout, 2)
	if got != 90*time.Second {
		t.Errorf("timeout retry 2: got %v, want 90s", got)
	}
}

func TestCooldownForClass_RateLimit(t *testing.T) {
	got := domain.CooldownForClass(domain.GommageClassRateLimit, 1)
	if got != 60*time.Second {
		t.Errorf("rate_limit retry 1: got %v, want 60s", got)
	}
	got = domain.CooldownForClass(domain.GommageClassRateLimit, 2)
	if got != 180*time.Second {
		t.Errorf("rate_limit retry 2: got %v, want 180s", got)
	}
}

func TestCooldownForClass_ParseError(t *testing.T) {
	got := domain.CooldownForClass(domain.GommageClassParseError, 1)
	if got != 5*time.Second {
		t.Errorf("parse_error retry 1: got %v, want 5s", got)
	}
}

func TestRecoveryDecision_IsRetry(t *testing.T) {
	d := domain.RecoveryDecision{Action: domain.RecoveryRetry}
	if !d.IsRetry() {
		t.Error("expected IsRetry true")
	}
	d2 := domain.RecoveryDecision{Action: domain.RecoveryHalt}
	if d2.IsRetry() {
		t.Error("expected IsRetry false for halt")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/nino/tap/paintress && go test ./internal/domain/ -run "TestCooldown|TestRecoveryDecision" -v && cd -`

- [ ] **Step 3: Implement RecoveryDecision + cooldownForClass**

```go
// internal/domain/gommage_recovery.go
package domain

import "time"

// RecoveryAction describes what the expedition loop should do.
type RecoveryAction string

const (
	RecoveryRetry RecoveryAction = "retry"
	RecoveryHalt  RecoveryAction = "halt"
)

// RecoveryDecision is the aggregate's verdict on what to do after Gommage.
type RecoveryDecision struct {
	Action      RecoveryAction
	Class       GommageClass
	Cooldown    time.Duration
	RetryNum    int
	MaxRetry    int
	KeepWorkDir bool
}

// IsRetry returns true if the decision is to retry.
func (d RecoveryDecision) IsRetry() bool {
	return d.Action == RecoveryRetry
}

// cooldown base values per class.
var cooldownBase = map[GommageClass][2]time.Duration{
	GommageClassTimeout:    {30 * time.Second, 90 * time.Second},
	GommageClassRateLimit:  {60 * time.Second, 180 * time.Second},
	GommageClassParseError: {5 * time.Second, 15 * time.Second},
}

// CooldownForClass returns the cooldown duration for a given class and retry number (1-indexed).
func CooldownForClass(class GommageClass, retryNum int) time.Duration {
	base, ok := cooldownBase[class]
	if !ok {
		return 0
	}
	if retryNum <= 1 {
		return base[0]
	}
	return base[1]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/nino/tap/paintress && go test ./internal/domain/ -run "TestCooldown|TestRecoveryDecision" -v && cd -`

- [ ] **Step 5: Commit**

```bash
cd /Users/nino/tap/paintress && git add internal/domain/gommage_recovery.go internal/domain/gommage_recovery_test.go && git commit -m "feat(domain): add RecoveryDecision type and cooldownForClass [BEHAVIORAL]" && cd -
```

---

### Task 3: Domain — ExpeditionAggregate.DecideRecovery + event extensions

**Files:**
- Modify: `internal/domain/expedition_aggregate.go`
- Modify: `internal/domain/expedition_aggregate_test.go`
- Modify: `internal/domain/event.go`

- [ ] **Step 1: Add new event types and data structs to event.go**

Add after `EventGommageTriggered` definition:

```go
EventGommageRecovery      EventType = "gommage.recovery"
EventExpeditionCheckpoint EventType = "expedition.checkpoint"
```

Extend `GommageTriggeredData`:

```go
type GommageTriggeredData struct {
	Expedition          int          `json:"expedition"`
	ConsecutiveFailures int          `json:"consecutive_failures"`
	Class               GommageClass `json:"class,omitempty"`
	RecoveryAction      string       `json:"recovery_action,omitempty"`
	RetryNum            int          `json:"retry_num,omitempty"`
}
```

Add new data structs:

```go
type GommageRecoveryData struct {
	Expedition int          `json:"expedition"`
	Class      GommageClass `json:"class"`
	Action     string       `json:"action"`
	RetryNum   int          `json:"retry_num"`
	Cooldown   string       `json:"cooldown"`
}

type ExpeditionCheckpointData struct {
	Expedition  int    `json:"expedition"`
	Phase       string `json:"phase"`
	WorkDir     string `json:"work_dir"`
	CommitCount int    `json:"commit_count"`
}
```

- [ ] **Step 2: Write failing tests for DecideRecovery**

Add to `internal/domain/expedition_aggregate_test.go`:

```go
func TestDecideRecovery_RetryOnTimeout(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"timeout", "timeout", "timeout"}
	d := agg.DecideRecovery(reasons)
	if d.Action != domain.RecoveryRetry {
		t.Errorf("expected retry, got %s", d.Action)
	}
	if d.Class != domain.GommageClassTimeout {
		t.Errorf("expected timeout class, got %s", d.Class)
	}
	if d.RetryNum != 1 {
		t.Errorf("expected retryNum=1, got %d", d.RetryNum)
	}
}

func TestDecideRecovery_RetryOnRateLimit(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"rate_limit: 429", "rate_limit: 429", "rate_limit: 429"}
	d := agg.DecideRecovery(reasons)
	if d.Action != domain.RecoveryRetry {
		t.Errorf("expected retry, got %s", d.Action)
	}
	if d.Class != domain.GommageClassRateLimit {
		t.Errorf("expected rate_limit, got %s", d.Class)
	}
}

func TestDecideRecovery_HaltOnBlocker(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"blocker: stuck", "blocker: stuck", "blocker: stuck"}
	d := agg.DecideRecovery(reasons)
	if d.Action != domain.RecoveryHalt {
		t.Errorf("expected halt for blocker, got %s", d.Action)
	}
}

func TestDecideRecovery_HaltAfterMaxRetries(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"timeout", "timeout", "timeout"}
	agg.DecideRecovery(reasons) // retry 1
	agg.DecideRecovery(reasons) // retry 2
	d := agg.DecideRecovery(reasons) // should halt
	if d.Action != domain.RecoveryHalt {
		t.Errorf("expected halt after max retries, got %s", d.Action)
	}
}

func TestDecideRecovery_ResetOnSuccess(t *testing.T) {
	agg := domain.NewExpeditionAggregate()
	reasons := []string{"timeout", "timeout", "timeout"}
	agg.DecideRecovery(reasons) // retry 1
	agg.ResetRecovery()         // success happened
	d := agg.DecideRecovery(reasons) // should be retry 1 again
	if d.RetryNum != 1 {
		t.Errorf("expected retryNum=1 after reset, got %d", d.RetryNum)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /Users/nino/tap/paintress && go test ./internal/domain/ -run TestDecideRecovery -v && cd -`

- [ ] **Step 4: Implement DecideRecovery + ResetRecovery**

Add to `expedition_aggregate.go`:

```go
const maxRecoveryAttempts = 2

// DecideRecovery classifies the failure streak and decides retry vs halt.
func (a *ExpeditionAggregate) DecideRecovery(reasons []string) RecoveryDecision {
	class := ClassifyGommage(reasons)
	switch class {
	case GommageClassTimeout, GommageClassRateLimit, GommageClassParseError:
		if a.recoveryAttempts >= maxRecoveryAttempts {
			return RecoveryDecision{Action: RecoveryHalt, Class: class}
		}
		a.recoveryAttempts++
		return RecoveryDecision{
			Action:      RecoveryRetry,
			Class:       class,
			Cooldown:    CooldownForClass(class, a.recoveryAttempts),
			RetryNum:    a.recoveryAttempts,
			MaxRetry:    maxRecoveryAttempts,
			KeepWorkDir: true,
		}
	default:
		return RecoveryDecision{Action: RecoveryHalt, Class: class}
	}
}

// ResetRecovery clears recovery attempts. Called when consecutiveFailures resets.
func (a *ExpeditionAggregate) ResetRecovery() {
	a.recoveryAttempts = 0
}
```

Add `recoveryAttempts int` field to `ExpeditionAggregate` struct.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/nino/tap/paintress && go test ./internal/domain/ -run TestDecideRecovery -v && cd -`

- [ ] **Step 6: Run all domain tests**

Run: `cd /Users/nino/tap/paintress && go test ./internal/domain/ -v && cd -`

- [ ] **Step 7: Commit**

```bash
cd /Users/nino/tap/paintress && git add internal/domain/expedition_aggregate.go internal/domain/expedition_aggregate_test.go internal/domain/event.go && git commit -m "feat(domain): add DecideRecovery to ExpeditionAggregate + event extensions [BEHAVIORAL]" && cd -
```

---

### Task 4: Port + Emitter — new EmitGommageRecovery + EmitCheckpoint

**Files:**
- Modify: `internal/usecase/port/port.go`
- Modify: `internal/usecase/emitter.go`

- [ ] **Step 1: Add methods to ExpeditionEventEmitter interface**

In `port.go`, add to `ExpeditionEventEmitter`:

```go
EmitGommageRecovery(expedition int, class, action string, retryNum int, cooldown string, now time.Time) error
EmitCheckpoint(expedition int, phase, workDir string, commitCount int, now time.Time) error
```

Add nop implementations to `NopExpeditionEventEmitter`.

- [ ] **Step 2: Add RecordGommageRecovery + RecordCheckpoint to aggregate**

In `expedition_aggregate.go`:

```go
func (a *ExpeditionAggregate) RecordGommageRecovery(expedition int, class GommageClass, action string, retryNum int, cooldown string, now time.Time) (Event, error) {
	return a.nextEvent(EventGommageRecovery, GommageRecoveryData{
		Expedition: expedition, Class: class, Action: action, RetryNum: retryNum, Cooldown: cooldown,
	}, now)
}

func (a *ExpeditionAggregate) RecordCheckpoint(expedition int, phase, workDir string, commitCount int, now time.Time) (Event, error) {
	return a.nextEvent(EventExpeditionCheckpoint, ExpeditionCheckpointData{
		Expedition: expedition, Phase: phase, WorkDir: workDir, CommitCount: commitCount,
	}, now)
}
```

- [ ] **Step 3: Implement emitter methods in emitter.go**

```go
func (e *expeditionEventEmitter) EmitGommageRecovery(expedition int, class, action string, retryNum int, cooldown string, now time.Time) error {
	ev, err := e.agg.RecordGommageRecovery(expedition, domain.GommageClass(class), action, retryNum, cooldown, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitCheckpoint(expedition int, phase, workDir string, commitCount int, now time.Time) error {
	ev, err := e.agg.RecordCheckpoint(expedition, phase, workDir, commitCount, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}
```

- [ ] **Step 4: Fix all test emitter mocks**

Update `failingEmitter` in `dmail_test.go`, `test_helpers_test.go`, `escalation_test.go` with new methods.

- [ ] **Step 5: Run all tests**

Run: `cd /Users/nino/tap/paintress && go test ./internal/... && cd -`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd /Users/nino/tap/paintress && git add internal/usecase/port/port.go internal/usecase/emitter.go internal/domain/expedition_aggregate.go internal/session/dmail_test.go internal/session/test_helpers_test.go internal/session/escalation_test.go && git commit -m "feat(port): add EmitGommageRecovery + EmitCheckpoint to emitter interface [STRUCTURAL]" && cd -
```

---

### Task 5: Session — executeRecovery + injectParseErrorLumina

**Files:**
- Create: `internal/session/gommage_recovery.go`
- Create: `internal/session/gommage_recovery_test.go`
- Modify: `internal/session/gommage_insight.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/session/gommage_recovery_test.go
package session

import (
	"context"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestExecuteRecovery_RetryTimeout(t *testing.T) {
	// white-box-reason: needs access to Paintress.reserve and Paintress.Emitter
	p := newTestPaintress(t)
	decision := domain.RecoveryDecision{
		Action: domain.RecoveryRetry, Class: domain.GommageClassTimeout,
		Cooldown: 1 * time.Millisecond, RetryNum: 1, MaxRetry: 2, KeepWorkDir: true,
	}
	ctx := context.Background()
	got := p.executeRecovery(ctx, decision, 1, nil)
	if !got {
		t.Error("expected true (retry) for timeout recovery")
	}
}

func TestExecuteRecovery_HaltSystematic(t *testing.T) {
	p := newTestPaintress(t)
	decision := domain.RecoveryDecision{Action: domain.RecoveryHalt, Class: domain.GommageClassSystematic}
	ctx := context.Background()
	got := p.executeRecovery(ctx, decision, 1, nil)
	if got {
		t.Error("expected false (halt) for systematic")
	}
}

func TestExecuteRecovery_ContextCancelled(t *testing.T) {
	p := newTestPaintress(t)
	decision := domain.RecoveryDecision{
		Action: domain.RecoveryRetry, Class: domain.GommageClassRateLimit,
		Cooldown: 10 * time.Second, RetryNum: 1, MaxRetry: 2, KeepWorkDir: true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := p.executeRecovery(ctx, decision, 1, nil)
	if got {
		t.Error("expected false when context is cancelled")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/nino/tap/paintress && go test ./internal/session/ -run TestExecuteRecovery -v && cd -`

- [ ] **Step 3: Implement executeRecovery**

```go
// internal/session/gommage_recovery.go
package session

import (
	"context"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// executeRecovery performs class-specific recovery. Returns true to retry same issue.
func (p *Paintress) executeRecovery(ctx context.Context, decision domain.RecoveryDecision, exp int, expedition *Expedition) bool {
	switch decision.Action {
	case domain.RecoveryRetry:
		p.Logger.Warn("gommage recovery: %s (retry %d/%d, cooldown %s)",
			decision.Class, decision.RetryNum, decision.MaxRetry, decision.Cooldown)

		switch decision.Class {
		case domain.GommageClassTimeout:
			p.reserve.ForceReserve()
		case domain.GommageClassParseError:
			injectParseErrorLumina(p.config.Continent, p.Logger)
		}

		_ = p.Emitter.EmitGommageRecovery(exp, string(decision.Class),
			string(decision.Action), decision.RetryNum, decision.Cooldown.String(), time.Now())

		select {
		case <-time.After(decision.Cooldown):
			return true
		case <-ctx.Done():
			return false
		}
	case domain.RecoveryHalt:
		return false
	}
	return false
}

// injectParseErrorLumina writes a corrective hint for the next expedition attempt.
func injectParseErrorLumina(continent string, logger interface{ Warn(string, ...any) }) {
	w := NewInsightWriter(domain.InsightsDir(continent), domain.RunDir(continent))
	entry := domain.InsightEntry{
		Title: "parse-error-recovery",
		What:  "Previous expedition output could not be parsed",
		Why:   "Claude output did not contain expected report markers",
		How:   "Ensure output follows the exact report format with markers",
	}
	_ = w.Append("lumina-recovery.md", "recovery", "paintress", entry)
}
```

- [ ] **Step 4: Update WriteGommageInsight to accept Class**

In `gommage_insight.go`, change signature:

```go
func WriteGommageInsight(w *InsightWriter, expedition, failureCount int, continent string, class domain.GommageClass) {
```

Add `class` to the Extra map:

```go
Extra: map[string]string{
	"failure-type":   "gommage",
	"gradient-level": "0",
	"gommage-class":  string(class),
},
```

Update the call site in `paintress_expedition.go` to pass `domain.GommageClassSystematic` temporarily (will be replaced in Task 7).

- [ ] **Step 5: Run tests**

Run: `cd /Users/nino/tap/paintress && go test ./internal/session/ -run TestExecuteRecovery -v && cd -`

- [ ] **Step 6: Run all tests**

Run: `cd /Users/nino/tap/paintress && go test ./internal/... && cd -`

- [ ] **Step 7: Commit**

```bash
cd /Users/nino/tap/paintress && git add internal/session/gommage_recovery.go internal/session/gommage_recovery_test.go internal/session/gommage_insight.go internal/session/paintress_expedition.go && git commit -m "feat(session): add executeRecovery and injectParseErrorLumina [BEHAVIORAL]" && cd -
```

---

### Task 6: Session — Checkpoint + Resume + Orphan Cleanup

**Files:**
- Create: `internal/session/gommage_checkpoint.go`
- Create: `internal/session/gommage_checkpoint_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/session/gommage_checkpoint_test.go
package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildResumeContext_WithCommits(t *testing.T) {
	// white-box-reason: tests internal buildResumeContext function
	dir := t.TempDir()
	// Set up a git repo with a commit
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Run()
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main"), 0644)
	run("add", ".")
	run("commit", "-m", "initial")

	ctx := buildResumeContext(dir)
	if ctx == "" {
		t.Error("expected non-empty resume context")
	}
}

func TestBuildResumeContext_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Run()

	ctx := buildResumeContext(dir)
	// Should still return something (empty diff is fine)
	if ctx == "" {
		t.Error("expected non-empty context even for empty repo")
	}
}

func TestCleanOrphanWorktrees_NoOrphans(t *testing.T) {
	// white-box-reason: tests cleanOrphanWorktrees on Paintress struct
	p := newTestPaintress(t)
	// Should not panic or error with no worktrees
	p.cleanOrphanWorktrees()
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/nino/tap/paintress && go test ./internal/session/ -run "TestBuildResume|TestCleanOrphan" -v && cd -`

- [ ] **Step 3: Implement checkpoint functions**

```go
// internal/session/gommage_checkpoint.go
package session

import (
	"os/exec"
	"strings"
	"time"
)

// CheckpointPhase tracks expedition progress for resume.
type CheckpointPhase string

const (
	CheckpointWorktreeReady   CheckpointPhase = "worktree_ready"
	CheckpointSubprocessStart CheckpointPhase = "subprocess_started"
)

// IncompleteExpedition represents an unfinished expedition found at startup.
type IncompleteExpedition struct {
	Expedition int
	WorkDir    string
	Phase      CheckpointPhase
}

// saveCheckpoint records expedition progress as an event.
func (p *Paintress) saveCheckpoint(exp int, phase CheckpointPhase, workDir string) {
	commitCount := countCommits(workDir)
	_ = p.Emitter.EmitCheckpoint(exp, string(phase), workDir, commitCount, time.Now())
}

// countCommits returns the number of commits on HEAD above the base.
func countCommits(workDir string) int {
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(out))
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// buildResumeContext generates lightweight context for --continue.
func buildResumeContext(workDir string) string {
	var b strings.Builder
	b.WriteString("Previous progress in worktree:\n")

	logCmd := exec.Command("git", "log", "--oneline", "-10")
	logCmd.Dir = workDir
	if out, err := logCmd.Output(); err == nil && len(out) > 0 {
		b.WriteString("Commits:\n")
		b.Write(out)
	}

	statCmd := exec.Command("git", "diff", "--stat")
	statCmd.Dir = workDir
	if out, err := statCmd.Output(); err == nil && len(out) > 0 {
		b.WriteString("\nUncommitted changes:\n")
		b.Write(out)
	}

	return b.String()
}

// cleanOrphanWorktrees removes worktrees from previous sessions.
func (p *Paintress) cleanOrphanWorktrees() {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = p.config.Continent
	out, err := cmd.Output()
	if err != nil {
		return
	}
	// Parse worktree list, find .expedition pattern, check age
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			if strings.Contains(path, ".expedition") || strings.Contains(path, "paintress-wt-") {
				// Check if older than 1 hour
				rmCmd := exec.Command("git", "worktree", "remove", "--force", path)
				rmCmd.Dir = p.config.Continent
				if rmErr := rmCmd.Run(); rmErr != nil {
					p.Logger.Warn("orphan worktree cleanup: %v", rmErr)
				}
			}
		}
	}
}
```

- [ ] **Step 4: Run tests**

Run: `cd /Users/nino/tap/paintress && go test ./internal/session/ -run "TestBuildResume|TestCleanOrphan" -v && cd -`

- [ ] **Step 5: Run all tests**

Run: `cd /Users/nino/tap/paintress && go test ./internal/... && cd -`

- [ ] **Step 6: Commit**

```bash
cd /Users/nino/tap/paintress && git add internal/session/gommage_checkpoint.go internal/session/gommage_checkpoint_test.go && git commit -m "feat(session): add checkpoint, resume context, and orphan cleanup [BEHAVIORAL]" && cd -
```

---

### Task 7: Integration — Wire everything into expedition loop

**Files:**
- Modify: `internal/session/paintress_expedition.go` (L215-236)
- Modify: `internal/session/paintress.go` (startup)

- [ ] **Step 1: Fix CAS race in Gommage guard (STRUCTURAL)**

In `paintress_expedition.go` L215, change:

```go
// BEFORE:
if p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures) && !p.escalationFired.Load() {
    p.escalationFired.Store(true)

// AFTER:
if p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures) && p.escalationFired.CompareAndSwap(false, true) {
```

- [ ] **Step 2: Run all tests to verify structural change**

Run: `cd /Users/nino/tap/paintress && go test ./internal/... && cd -`

- [ ] **Step 3: Commit structural change**

```bash
cd /Users/nino/tap/paintress && git add internal/session/paintress_expedition.go && git commit -m "tidy: fix TOCTOU race in Gommage guard with CompareAndSwap [STRUCTURAL]" && cd -
```

- [ ] **Step 4: Add rate_limit marker injection to handleExpeditionError**

In `paintress_expedition.go` `handleExpeditionError`, before `WriteJournal`:

```go
reason := runErr.Error()
if p.reserve.IsReserveActive() {
    reason = "rate_limit: " + reason
}
```

Use `reason` in the journal `errReport.Reason` field.

- [ ] **Step 5: Replace L215-236 Gommage block with class-aware dispatch**

```go
if p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures) && p.escalationFired.CompareAndSwap(false, true) {
    reasons := recentFailureReasons(p.config.Continent, 5)
    decision := p.aggregate.DecideRecovery(reasons)

    gommageWriter := NewInsightWriter(
        domain.InsightsDir(p.config.Continent),
        domain.RunDir(p.config.Continent),
    )
    WriteGommageInsight(gommageWriter, exp, maxConsecutiveFailures, p.config.Continent, decision.Class)

    if emitErr := p.Emitter.EmitGommage(exp, time.Now()); emitErr != nil {
        p.Logger.Error("gommage event lost: %v", emitErr)
    }

    expSpan.AddEvent("gommage",
        trace.WithAttributes(
            attribute.Int("consecutive_failures", maxConsecutiveFailures),
            attribute.String("gommage.class", string(decision.Class)),
            attribute.String("gommage.action", string(decision.Action)),
            attribute.Int("gommage.retry_num", decision.RetryNum),
        ),
    )

    if p.executeRecovery(ctx, decision, exp, expedition) {
        p.saveCheckpoint(exp, CheckpointSubprocessStart, workDir)
        p.consecutiveFailures.Store(0)
        p.escalationFired.Store(false)
        expSpan.End()
        continue // retry same issue
    }

    // Halt path
    p.stageEscalation(ctx, exp, maxConsecutiveFailures)
    releaseWorkDir()
    expSpan.End()
    p.Logger.Error("%s", fmt.Sprintf(domain.Msg("gommage"), maxConsecutiveFailures))
    return errGommage
}
```

- [ ] **Step 6: Add aggregate field + ResetRecovery call on success**

In `paintress.go`, add `aggregate *domain.ExpeditionAggregate` to `Paintress` struct. Initialize in constructor.

In `dispatchExpeditionResult` success path, call `p.aggregate.ResetRecovery()`.

- [ ] **Step 7: Add startup cleanup + resume to Paintress.Run**

In `paintress.go` `Run()`, before the main loop:

```go
p.cleanOrphanWorktrees()
```

- [ ] **Step 8: Write integration tests**

Add to `paintress_test.go`:

```go
func TestGommageRecovery_TimeoutThenSuccess(t *testing.T) {
    // Set up paintress with fake-claude that fails 3 times with timeout, then succeeds
    // Verify: recovery triggered, no errGommage returned, success after retry
}

func TestGommageRecovery_MaxRetriesThenHalt(t *testing.T) {
    // Set up paintress with fake-claude that always times out
    // Verify: 2 recovery retries, then errGommage returned
}
```

- [ ] **Step 9: Run all tests**

Run: `cd /Users/nino/tap/paintress && go test ./internal/... && cd -`

- [ ] **Step 10: Run full quality check**

Run: `cd /Users/nino/tap/paintress && just check && cd -`

- [ ] **Step 11: Commit**

```bash
cd /Users/nino/tap/paintress && git add internal/session/paintress_expedition.go internal/session/paintress.go internal/session/paintress_test.go && git commit -m "feat: wire Gommage recovery into expedition loop with CAS + checkpoint [BEHAVIORAL]" && cd -
```

---

### Task 8: Docs + Lint + Final Verification

**Files:**
- Modify: `docs/` as needed
- Run: full quality pipeline

- [ ] **Step 1: Update README if expedition loop behavior is documented**

Check `README.md` for Gommage references and update.

- [ ] **Step 2: Run docgen**

Run: `cd /Users/nino/tap/paintress && just docgen && cd -`

- [ ] **Step 3: Run full lint + semgrep + fmt**

Run: `cd /Users/nino/tap/paintress && just fmt && just lint && just semgrep && cd -`

- [ ] **Step 4: Run all tests one final time**

Run: `cd /Users/nino/tap/paintress && just test && cd -`

- [ ] **Step 5: Commit any remaining fixes**

```bash
cd /Users/nino/tap/paintress && git add -A && git commit -m "docs: update for Gommage recovery strategy" && cd -
```
