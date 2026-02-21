package paintress

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestFilterHighSeverity_NoHighSeverity(t *testing.T) {
	// given
	dmails := []DMail{
		{Name: "report-1", Kind: "report", Severity: ""},
		{Name: "spec-2", Kind: "specification", Severity: "low"},
	}

	// when
	high := FilterHighSeverity(dmails)

	// then
	if len(high) != 0 {
		t.Errorf("expected 0 HIGH severity d-mails, got %d", len(high))
	}
}

func TestFilterHighSeverity_MixedSeverity(t *testing.T) {
	// given
	dmails := []DMail{
		{Name: "report-1", Kind: "report", Severity: ""},
		{Name: "alert-1", Kind: "alert", Severity: "high"},
		{Name: "spec-1", Kind: "specification", Severity: "low"},
		{Name: "alert-2", Kind: "alert", Severity: "high"},
	}

	// when
	high := FilterHighSeverity(dmails)

	// then
	if len(high) != 2 {
		t.Fatalf("expected 2 HIGH severity d-mails, got %d", len(high))
	}
	if high[0].Name != "alert-1" || high[1].Name != "alert-2" {
		t.Errorf("unexpected high d-mails: %v", high)
	}
}

func TestFilterHighSeverity_EmptySlice(t *testing.T) {
	// given
	var dmails []DMail

	// when
	high := FilterHighSeverity(dmails)

	// then
	if len(high) != 0 {
		t.Errorf("expected 0 for nil/empty, got %d", len(high))
	}
}

// TestHighSeverityGate_NoHighSeverity verifies no gate is triggered
// when inbox has no HIGH severity d-mails.
func TestHighSeverityGate_NoHighSeverity(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	// Place a LOW severity d-mail in inbox
	content := "---\nname: spec-1\nkind: specification\ndescription: normal spec\nseverity: low\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "spec-1.md"), []byte(content), 0644)

	cfg := Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	// Approver that would fail if called
	p := NewPaintress(cfg, NewLogger(io.Discard, false))
	p.approver = &failApprover{t: t}
	p.notifier = &NopNotifier{}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 (no gate should trigger)", code)
	}
	if p.totalSuccess.Load() != 1 {
		t.Errorf("totalSuccess = %d, want 1 (expedition should run normally)", p.totalSuccess.Load())
	}
}

// TestHighSeverityGate_Approved verifies expedition runs when HIGH
// severity d-mail exists but human approves.
func TestHighSeverityGate_Approved(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	// Place a HIGH severity d-mail
	content := "---\nname: alert-1\nkind: alert\ndescription: critical issue\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-1.md"), []byte(content), 0644)

	cfg := Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, NewLogger(io.Discard, false))
	p.approver = &AutoApprover{}
	p.notifier = &NopNotifier{}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 (approved gate should continue)", code)
	}
	if p.totalSuccess.Load() != 1 {
		t.Errorf("totalSuccess = %d, want 1", p.totalSuccess.Load())
	}
}

// TestHighSeverityGate_Denied verifies no expeditions run when
// HIGH severity d-mail exists and human denies.
func TestHighSeverityGate_Denied(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-deny\nkind: alert\ndescription: critical\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-deny.md"), []byte(content), 0644)

	cfg := Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, NewLogger(io.Discard, false))
	p.approver = &denyApprover{}
	p.notifier = &NopNotifier{}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}
	// Pre-flight gate denial should abort before any expedition is attempted
	if p.totalAttempted.Load() != 0 {
		t.Errorf("totalAttempted = %d, want 0 (gate denial should abort before workers)", p.totalAttempted.Load())
	}
	if p.totalSuccess.Load() != 0 {
		t.Errorf("totalSuccess = %d, want 0", p.totalSuccess.Load())
	}
}

// TestHighSeverityGate_AutoApprove verifies --auto-approve skips the gate entirely.
func TestHighSeverityGate_AutoApprove(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-auto\nkind: alert\ndescription: critical\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-auto.md"), []byte(content), 0644)

	cfg := Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
		AutoApprove:    true,
	}

	p := NewPaintress(cfg, NewLogger(io.Discard, false))
	// AutoApprove wiring happens in NewPaintress — approver should be AutoApprover

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 (auto-approve should skip gate)", code)
	}
	if p.totalSuccess.Load() != 1 {
		t.Errorf("totalSuccess = %d, want 1", p.totalSuccess.Load())
	}
}

// TestHighSeverityGate_ApproverCalledOnce verifies that the gate invokes
// the approver exactly once, even with multiple workers. This prevents
// concurrent StdinApprover reads from deadlocking the run.
func TestHighSeverityGate_ApproverCalledOnce(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-once\nkind: alert\ndescription: critical\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-once.md"), []byte(content), 0644)

	cfg := Config{
		Continent:      dir,
		Workers:        0, // single-worker mode (avoids needing worktree pool)
		MaxExpeditions: 3,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	var callCount atomic.Int32
	p := NewPaintress(cfg, NewLogger(io.Discard, false))
	p.approver = &countingApprover{count: &callCount, approve: true}
	p.notifier = &NopNotifier{}

	p.Run(context.Background())

	if callCount.Load() != 1 {
		t.Errorf("approver called %d times, want exactly 1 (gate should run once before workers)", callCount.Load())
	}
}

// TestHighSeverityGate_DeniedAbortsAllExpeditions verifies that denial
// in the pre-flight gate prevents ALL expeditions from running.
func TestHighSeverityGate_DeniedAbortsAllExpeditions(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-abort\nkind: alert\ndescription: critical\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-abort.md"), []byte(content), 0644)

	cfg := Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 5,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, NewLogger(io.Discard, false))
	p.approver = &denyApprover{}
	p.notifier = &NopNotifier{}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 (denied is a clean exit)", code)
	}
	// No expeditions should have been attempted
	if p.totalAttempted.Load() != 0 {
		t.Errorf("totalAttempted = %d, want 0 (gate denial should abort before workers)", p.totalAttempted.Load())
	}
}

// TestHighSeverityGate_ScanError_FailsClosed verifies that when the inbox
// cannot be read (e.g. permission denied), the run aborts with exit code 1
// rather than silently skipping the gate.
func TestHighSeverityGate_ScanError_FailsClosed(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	// Place a valid d-mail, then make the inbox unreadable
	content := "---\nname: alert-perm\nkind: alert\ndescription: test\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-perm.md"), []byte(content), 0644)
	os.Chmod(inboxDir, 0000) // make inbox unreadable
	t.Cleanup(func() { os.Chmod(inboxDir, 0755) })

	cfg := Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, NewLogger(io.Discard, false))
	p.approver = &failApprover{t: t}
	p.notifier = &NopNotifier{}

	code := p.Run(context.Background())
	if code != 1 {
		t.Fatalf("Run() = %d, want 1 (fail-closed on scan error)", code)
	}
	if p.totalAttempted.Load() != 0 {
		t.Errorf("totalAttempted = %d, want 0 (no expeditions should run)", p.totalAttempted.Load())
	}
}

// countingApprover counts how many times RequestApproval is called.
type countingApprover struct {
	count   *atomic.Int32
	approve bool
}

func (a *countingApprover) RequestApproval(_ context.Context, _ string) (bool, error) {
	a.count.Add(1)
	return a.approve, nil
}

// failApprover fails the test if RequestApproval is called.
type failApprover struct {
	t *testing.T
}

func (a *failApprover) RequestApproval(_ context.Context, _ string) (bool, error) {
	a.t.Error("Approver.RequestApproval should not be called when no HIGH severity d-mails exist")
	return false, nil
}

// denyApprover always denies.
type denyApprover struct{}

func (a *denyApprover) RequestApproval(_ context.Context, _ string) (bool, error) {
	return false, nil
}
