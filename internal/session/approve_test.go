package session

// white-box-reason: session internals: tests unexported StdinApprover reader/writer injection

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
)

func TestStdinApprover_Yes(t *testing.T) {
	// given
	in := strings.NewReader("y\n")
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	approved, err := a.RequestApproval(context.Background(), "Continue expedition?")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true for 'y' input")
	}
	if !strings.Contains(out.String(), "Continue? [y/N]") {
		t.Errorf("prompt not shown, got: %q", out.String())
	}
}

func TestStdinApprover_YesFull(t *testing.T) {
	// given
	in := strings.NewReader("yes\n")
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true for 'yes' input")
	}
}

func TestStdinApprover_No(t *testing.T) {
	// given
	in := strings.NewReader("n\n")
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for 'n' input")
	}
}

func TestStdinApprover_EmptyDefault(t *testing.T) {
	// given: empty enter = default = deny (safe side)
	in := strings.NewReader("\n")
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for empty input (safe default)")
	}
}

func TestStdinApprover_ContextCancel(t *testing.T) {
	// given: context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Use a reader that blocks forever (empty pipe simulation)
	in := new(blockingReader)
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	approved, err := a.RequestApproval(ctx, "msg")

	// then
	if approved {
		t.Error("expected approved=false when context is cancelled")
	}
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestCmdApprover_ExitZero(t *testing.T) {
	// given
	a := &CmdApprover{
		cmdTemplate: "true", // exit 0
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			return &fakeCmd{err: nil}
		},
	}

	// when
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true for exit 0")
	}
}

func TestCmdApprover_ExitNonZero(t *testing.T) {
	// given: non-zero exit is intentional deny — error should be nil
	a := &CmdApprover{
		cmdTemplate: "false", // exit 1
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			return &fakeCmd{err: &exec.ExitError{}}
		},
	}

	// when
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for exit 1")
	}
}

func TestCmdApprover_ExecutionError_SurfacesError(t *testing.T) {
	// given: execution error (e.g. binary not found) — must surface as error
	execErr := fmt.Errorf("exec: \"nonexistent\": executable file not found in $PATH")
	a := &CmdApprover{
		cmdTemplate: "nonexistent-binary",
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			return &fakeCmd{err: execErr}
		},
	}

	// when
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then
	if approved {
		t.Error("expected approved=false for execution error")
	}
	if err == nil {
		t.Fatal("expected error to be surfaced for execution failure, got nil")
	}
	if !strings.Contains(err.Error(), "executable file not found") {
		t.Errorf("error should contain original message, got: %v", err)
	}
}

func TestCmdApprover_PlaceholderReplacement(t *testing.T) {
	// given
	var capturedShellCmd string
	a := &CmdApprover{
		cmdTemplate: `echo {message}`,
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			capturedShellCmd = args[len(args)-1]
			return &fakeCmd{}
		},
	}

	// when
	_, _ = a.RequestApproval(context.Background(), "HIGH severity alert")

	// then: message is shell-quoted
	want := `echo 'HIGH severity alert'`
	if capturedShellCmd != want {
		t.Errorf("shell cmd = %q, want %q", capturedShellCmd, want)
	}
	if strings.Contains(capturedShellCmd, "{message}") {
		t.Error("shell cmd still contains {message} placeholder")
	}
}

func TestCmdApprover_EscapesShellMetacharacters(t *testing.T) {
	// given: message with shell metacharacters that could force exit 0
	var capturedShellCmd string
	a := &CmdApprover{
		cmdTemplate: `echo {message}`,
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			capturedShellCmd = args[len(args)-1]
			return &fakeCmd{}
		},
	}

	// when: attacker message tries to inject shell commands
	_, _ = a.RequestApproval(context.Background(), `"; exit 0; #`)

	// then: message should be single-quoted, not interpolated raw
	want := `echo '"; exit 0; #'`
	if capturedShellCmd != want {
		t.Errorf("shell cmd = %q, want %q (message must be shell-quoted)", capturedShellCmd, want)
	}
}

func TestAutoApprover(t *testing.T) {
	// given
	a := &port.AutoApprover{}

	// when
	approved, err := a.RequestApproval(context.Background(), "anything")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("AutoApprover should always return true")
	}
}

// blockingReader never returns data, simulating a blocking stdin.
type blockingReader struct{}

func (r *blockingReader) Read(p []byte) (int, error) {
	select {} //nolint:staticcheck // intentional blocking for test
}

// TestStdinApprover_ShowsMessage verifies the approval message is displayed.
func TestStdinApprover_ShowsMessage(t *testing.T) {
	// given
	in := strings.NewReader("n\n")
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	_, _ = a.RequestApproval(context.Background(), "HIGH severity D-Mail detected")

	// then
	if !strings.Contains(out.String(), "HIGH severity D-Mail detected") {
		t.Errorf("output should contain message, got: %q", out.String())
	}
}

// TestStdinApprover_Timeout verifies timeout behavior.
func TestStdinApprover_Timeout(t *testing.T) {
	// given
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	in := new(blockingReader)
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	approved, err := a.RequestApproval(ctx, "msg")

	// then
	if approved {
		t.Error("expected approved=false on timeout")
	}
	if err == nil {
		t.Error("expected error on timeout")
	}
}

// --- High Severity Gate integration tests (moved from gate_test.go) ---

func TestStdinApprover_NilInput(t *testing.T) {
	// given: StdinApprover with nil input (library/non-interactive usage)
	a := &StdinApprover{reader: nil, writer: new(bytes.Buffer)}

	// when: should not panic
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then: safe default = deny, no error
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected denial for nil input")
	}
}

func TestStdinApprover_EOFTerminatedYes(t *testing.T) {
	// given: piped input "y" without trailing newline (echo -n "y" | paintress run)
	in := strings.NewReader("y")
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then: should approve even without trailing newline
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approval for EOF-terminated 'y' input")
	}
}

func TestStdinApprover_EOFTerminatedNo(t *testing.T) {
	// given: piped "n" without trailing newline — should deny (not error)
	in := strings.NewReader("n")
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected denial for EOF-terminated 'n' input")
	}
}

func TestStdinApprover_SharedReader(t *testing.T) {
	// given: a shared reader with approval line + subsequent data.
	// After RequestApproval consumes "y\n", the remaining "next-line\n"
	// must still be readable from the same reader.
	in := strings.NewReader("y\nnext-line\n")
	out := new(bytes.Buffer)
	a := &StdinApprover{reader: in, writer: out}

	// when
	approved, err := a.RequestApproval(context.Background(), "msg")

	// then: approved
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Fatal("expected approval")
	}

	// then: remaining data is still available from the shared reader
	remaining := make([]byte, 64)
	n, _ := in.Read(remaining)
	got := string(remaining[:n])
	if got != "next-line\n" {
		t.Errorf("shared reader lost data: got %q, want %q", got, "next-line\n")
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

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	// Approver that would fail if called
	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil)
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

// TestHighSeverityGate_Approved verifies expedition runs when HIGH
// severity d-mail exists but human approves.
func TestHighSeverityGate_Approved(t *testing.T) {
	dir := setupTestRepo(t)
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	// Place a HIGH severity d-mail
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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil)
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

// TestHighSeverityGate_Denied verifies no expeditions run when
// HIGH severity d-mail exists and human denies.
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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil)
	p.approver = &denyApprover{}
	p.notifier = &port.NopNotifier{}

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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil)
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

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0, // single-worker mode (avoids needing worktree pool)
		MaxExpeditions: 3,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	var callCount atomic.Int32
	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil)
	p.approver = &countingApprover{count: &callCount, approve: true}
	p.notifier = &port.NopNotifier{}

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

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 5,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil)
	p.approver = &denyApprover{}
	p.notifier = &port.NopNotifier{}

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

	cfg := domain.Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil)
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

// TestHighSeverityGate_ApprovalError_FailsClosed verifies that when the
// approver returns a technical error (e.g. command not found, context timeout),
// the run aborts with exit code 1 rather than treating it as a denial (exit 0).
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

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil)
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

func TestBuildApprover_AutoApprove(t *testing.T) {
	// given
	cfg := domain.Config{AutoApprove: true}

	// when
	approver := BuildApprover(cfg, nil, nil)

	// then
	if _, ok := approver.(*port.AutoApprover); !ok {
		t.Errorf("expected AutoApprover, got %T", approver)
	}
}

func TestBuildApprover_CmdApprover(t *testing.T) {
	// given
	cfg := domain.Config{ApproveCmd: "echo approve"}

	// when
	approver := BuildApprover(cfg, nil, nil)

	// then
	if approver == nil {
		t.Fatal("expected non-nil approver")
	}
	if _, ok := approver.(*port.AutoApprover); ok {
		t.Error("expected CmdApprover, got AutoApprover")
	}
}

func TestBuildApprover_StdinApprover(t *testing.T) {
	// given
	cfg := domain.Config{}
	input := strings.NewReader("")

	// when
	approver := BuildApprover(cfg, input, io.Discard)

	// then
	if approver == nil {
		t.Fatal("expected non-nil approver")
	}
	if _, ok := approver.(*port.AutoApprover); ok {
		t.Error("expected StdinApprover, got AutoApprover")
	}
}
