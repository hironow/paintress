package session_test

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase/port"
)

func TestStdinApprover_Yes(t *testing.T) {
	in := strings.NewReader("y\n")
	out := new(bytes.Buffer)
	a := session.NewStdinApprover(in, out)

	approved, err := a.RequestApproval(context.Background(), "Continue expedition?")
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
	in := strings.NewReader("yes\n")
	a := session.NewStdinApprover(in, new(bytes.Buffer))

	approved, err := a.RequestApproval(context.Background(), "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true for 'yes' input")
	}
}

func TestStdinApprover_No(t *testing.T) {
	in := strings.NewReader("n\n")
	a := session.NewStdinApprover(in, new(bytes.Buffer))

	approved, err := a.RequestApproval(context.Background(), "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for 'n' input")
	}
}

func TestStdinApprover_EmptyDefault(t *testing.T) {
	in := strings.NewReader("\n")
	a := session.NewStdinApprover(in, new(bytes.Buffer))

	approved, err := a.RequestApproval(context.Background(), "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for empty input (safe default)")
	}
}

func TestStdinApprover_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := session.NewStdinApprover(new(blockingReader), new(bytes.Buffer))

	approved, err := a.RequestApproval(ctx, "msg")
	if approved {
		t.Error("expected approved=false when context is cancelled")
	}
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestStdinApprover_NilInput(t *testing.T) {
	a := session.NewStdinApprover(nil, new(bytes.Buffer))

	approved, err := a.RequestApproval(context.Background(), "msg")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected denial for nil input")
	}
}

func TestStdinApprover_EOFTerminatedYes(t *testing.T) {
	a := session.NewStdinApprover(strings.NewReader("y"), new(bytes.Buffer))

	approved, err := a.RequestApproval(context.Background(), "msg")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approval for EOF-terminated 'y' input")
	}
}

func TestStdinApprover_EOFTerminatedNo(t *testing.T) {
	a := session.NewStdinApprover(strings.NewReader("n"), new(bytes.Buffer))

	approved, err := a.RequestApproval(context.Background(), "msg")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected denial for EOF-terminated 'n' input")
	}
}

func TestStdinApprover_SharedReader(t *testing.T) {
	in := strings.NewReader("y\nnext-line\n")
	a := session.NewStdinApprover(in, new(bytes.Buffer))

	approved, err := a.RequestApproval(context.Background(), "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Fatal("expected approval")
	}
	remaining := make([]byte, 64)
	n, _ := in.Read(remaining)
	if string(remaining[:n]) != "next-line\n" {
		t.Errorf("shared reader lost data: got %q, want %q", string(remaining[:n]), "next-line\n")
	}
}

func TestStdinApprover_ShowsMessage(t *testing.T) {
	out := new(bytes.Buffer)
	a := session.NewStdinApprover(strings.NewReader("n\n"), out)

	a.RequestApproval(context.Background(), "HIGH severity D-Mail detected")
	if !strings.Contains(out.String(), "HIGH severity D-Mail detected") {
		t.Errorf("output should contain message, got: %q", out.String())
	}
}

func TestStdinApprover_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	a := session.NewStdinApprover(new(blockingReader), new(bytes.Buffer))

	approved, err := a.RequestApproval(ctx, "msg")
	if approved {
		t.Error("expected approved=false on timeout")
	}
	if err == nil {
		t.Error("expected error on timeout")
	}
}

func TestCmdApprover_ExitZero(t *testing.T) {
	a := session.NewCmdApproverForTest("true",
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("true")
		},
	)

	approved, err := a.RequestApproval(context.Background(), "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true for exit 0")
	}
}

func TestCmdApprover_ExitNonZero(t *testing.T) {
	a := session.NewCmdApproverForTest("false",
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("false")
		},
	)

	approved, err := a.RequestApproval(context.Background(), "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for exit 1")
	}
}

func TestCmdApprover_ExecutionError_SurfacesError(t *testing.T) {
	a := session.NewCmdApproverForTest("nonexistent-binary",
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("nonexistent-binary-that-does-not-exist-anywhere")
		},
	)

	approved, err := a.RequestApproval(context.Background(), "msg")
	if approved {
		t.Error("expected approved=false for execution error")
	}
	if err == nil {
		t.Fatal("expected error to be surfaced for execution failure, got nil")
	}
}

func TestCmdApprover_PlaceholderReplacement(t *testing.T) {
	var capturedShellCmd string
	a := session.NewCmdApproverForTest(`echo {message}`,
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			capturedShellCmd = args[len(args)-1]
			return exec.Command("true")
		},
	)

	a.RequestApproval(context.Background(), "HIGH severity alert")
	want := `echo 'HIGH severity alert'`
	if capturedShellCmd != want {
		t.Errorf("shell cmd = %q, want %q", capturedShellCmd, want)
	}
}

func TestCmdApprover_EscapesShellMetacharacters(t *testing.T) {
	var capturedShellCmd string
	a := session.NewCmdApproverForTest(`echo {message}`,
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			capturedShellCmd = args[len(args)-1]
			return exec.Command("true")
		},
	)

	a.RequestApproval(context.Background(), `"; exit 0; #`)
	want := `echo '"; exit 0; #'`
	if capturedShellCmd != want {
		t.Errorf("shell cmd = %q, want %q", capturedShellCmd, want)
	}
}

func TestAutoApprover(t *testing.T) {
	a := &port.AutoApprover{}

	approved, err := a.RequestApproval(context.Background(), "anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("AutoApprover should always return true")
	}
}

func TestBuildApprover_AutoApprove(t *testing.T) {
	approver := session.BuildApprover(domain.Config{AutoApprove: true}, nil, nil)
	if _, ok := approver.(*port.AutoApprover); !ok {
		t.Errorf("expected AutoApprover, got %T", approver)
	}
}

func TestBuildApprover_CmdApprover(t *testing.T) {
	approver := session.BuildApprover(domain.Config{ApproveCmd: "echo approve"}, nil, nil)
	if approver == nil {
		t.Fatal("expected non-nil approver")
	}
	if _, ok := approver.(*port.AutoApprover); ok {
		t.Error("expected CmdApprover, got AutoApprover")
	}
}

func TestBuildApprover_StdinApprover(t *testing.T) {
	approver := session.BuildApprover(domain.Config{}, strings.NewReader(""), io.Discard)
	if approver == nil {
		t.Fatal("expected non-nil approver")
	}
	if _, ok := approver.(*port.AutoApprover); ok {
		t.Error("expected StdinApprover, got AutoApprover")
	}
}

// blockingReader never returns data, simulating a blocking stdin.
type blockingReader struct{}

func (r *blockingReader) Read(p []byte) (int, error) {
	select {} //nolint:staticcheck // intentional blocking for test
}
