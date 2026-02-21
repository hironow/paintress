package paintress

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
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
	// given
	a := &CmdApprover{
		cmdTemplate: "false", // exit 1
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			return &fakeCmd{err: &fakeExitError{code: 1}}
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

func TestCmdApprover_PlaceholderReplacement(t *testing.T) {
	// given
	var capturedShellCmd string
	a := &CmdApprover{
		cmdTemplate: `echo "{message}"`,
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			capturedShellCmd = args[len(args)-1]
			return &fakeCmd{}
		},
	}

	// when
	_, _ = a.RequestApproval(context.Background(), "HIGH severity alert")

	// then
	if !strings.Contains(capturedShellCmd, "HIGH severity alert") {
		t.Errorf("shell cmd should contain message, got: %s", capturedShellCmd)
	}
	if strings.Contains(capturedShellCmd, "{message}") {
		t.Error("shell cmd still contains {message} placeholder")
	}
}

func TestAutoApprover(t *testing.T) {
	// given
	a := &AutoApprover{}

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

// fakeExitError satisfies the error interface for testing non-zero exits.
type fakeExitError struct {
	code int
}

func (e *fakeExitError) Error() string {
	return "exit status " + string(rune('0'+e.code))
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
