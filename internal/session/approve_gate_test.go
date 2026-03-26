package session

// white-box-reason: session internals: tests unexported Paintress.approver field injection for gate behavior

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
)

func TestHighSeverityGate_NoHighSeverity(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: spec-1\nkind: specification\ndescription: normal spec\nseverity: low\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "spec-1.md"), []byte(content), 0644)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	p.approver = &failApprover{t: t}
	p.notifier = &port.NopNotifier{}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 (no gate should trigger)", code)
	}
	if p.totalSuccess.Load() != 1 {
		t.Errorf("totalSuccess = %d, want 1 (expedition should run normally)", p.totalSuccess.Load())
	}
}

func TestHighSeverityGate_Approved(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-1\nkind: alert\ndescription: critical issue\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-1.md"), []byte(content), 0644)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	p.approver = &port.AutoApprover{}
	p.notifier = &port.NopNotifier{}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 (approved gate should continue)", code)
	}
	if p.totalSuccess.Load() != 1 {
		t.Errorf("totalSuccess = %d, want 1", p.totalSuccess.Load())
	}
}

func TestHighSeverityGate_Denied(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-deny\nkind: alert\ndescription: critical\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-deny.md"), []byte(content), 0644)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	p.approver = &denyApprover{}
	p.notifier = &port.NopNotifier{}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}
	if p.totalAttempted.Load() != 0 {
		t.Errorf("totalAttempted = %d, want 0 (gate denial should abort before workers)", p.totalAttempted.Load())
	}
	if p.totalSuccess.Load() != 0 {
		t.Errorf("totalSuccess = %d, want 0", p.totalSuccess.Load())
	}
}

func TestHighSeverityGate_AutoApprove(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-auto\nkind: alert\ndescription: critical\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-auto.md"), []byte(content), 0644)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
		AutoApprove:    true,
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 (auto-approve should skip gate)", code)
	}
	if p.totalSuccess.Load() != 1 {
		t.Errorf("totalSuccess = %d, want 1", p.totalSuccess.Load())
	}
}

func TestHighSeverityGate_ApproverCalledOnce(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-once\nkind: alert\ndescription: critical\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-once.md"), []byte(content), 0644)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 3,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	var callCount atomic.Int32
	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	p.approver = &countingApprover{count: &callCount, approve: true}
	p.notifier = &port.NopNotifier{}

	p.Run(context.Background())

	if callCount.Load() != 1 {
		t.Errorf("approver called %d times, want exactly 1 (gate should run once before workers)", callCount.Load())
	}
}

func TestHighSeverityGate_DeniedAbortsAllExpeditions(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-abort\nkind: alert\ndescription: critical\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-abort.md"), []byte(content), 0644)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 5,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	p.approver = &denyApprover{}
	p.notifier = &port.NopNotifier{}

	code := p.Run(context.Background())
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 (denied is a clean exit)", code)
	}
	if p.totalAttempted.Load() != 0 {
		t.Errorf("totalAttempted = %d, want 0 (gate denial should abort before workers)", p.totalAttempted.Load())
	}
}

func TestHighSeverityGate_ScanError_FailsClosed(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-perm\nkind: alert\ndescription: test\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-perm.md"), []byte(content), 0644)
	os.Chmod(inboxDir, 0000)
	t.Cleanup(func() { os.Chmod(inboxDir, 0755) })

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	p.approver = &failApprover{t: t}
	p.notifier = &port.NopNotifier{}

	code := p.Run(context.Background())
	if code != 1 {
		t.Fatalf("Run() = %d, want 1 (fail-closed on scan error)", code)
	}
	if p.totalAttempted.Load() != 0 {
		t.Errorf("totalAttempted = %d, want 0 (no expeditions should run)", p.totalAttempted.Load())
	}
}

func TestHighSeverityGate_ApprovalError_FailsClosed(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	content := "---\nname: alert-err\nkind: alert\ndescription: critical\nseverity: high\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "alert-err.md"), []byte(content), 0644)

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	p.approver = &errorApprover{err: fmt.Errorf("exec: command not found")}
	p.notifier = &port.NopNotifier{}

	code := p.Run(context.Background())
	if code != 1 {
		t.Fatalf("Run() = %d, want 1 (fail-closed on approval error)", code)
	}
	if p.totalAttempted.Load() != 0 {
		t.Errorf("totalAttempted = %d, want 0 (no expeditions should run on approval error)", p.totalAttempted.Load())
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

// errorApprover simulates a technical failure during approval.
type errorApprover struct {
	err error
}

func (a *errorApprover) RequestApproval(_ context.Context, _ string) (bool, error) {
	return false, a.err
}
