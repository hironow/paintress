package paintress

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestLocalNotifier_Darwin_CommandShape(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only test")
	}

	// given
	var capturedName string
	var capturedArgs []string
	n := &LocalNotifier{
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			capturedName = name
			capturedArgs = args
			return &fakeCmd{} // succeeds immediately
		},
	}

	// when
	err := n.Notify(context.Background(), "Test Title", "Test Message")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedName != "osascript" {
		t.Errorf("command = %q, want osascript", capturedName)
	}
	joined := strings.Join(capturedArgs, " ")
	if !strings.Contains(joined, "Test Title") {
		t.Errorf("args should contain title, got: %s", joined)
	}
	if !strings.Contains(joined, "Test Message") {
		t.Errorf("args should contain message, got: %s", joined)
	}
}

func TestLocalNotifier_Linux_CommandShape(t *testing.T) {
	// given: force linux path regardless of runtime
	var capturedName string
	var capturedArgs []string
	n := &LocalNotifier{
		forceOS: "linux",
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			capturedName = name
			capturedArgs = args
			return &fakeCmd{}
		},
	}

	// when
	err := n.Notify(context.Background(), "Test Title", "Test Message")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedName != "notify-send" {
		t.Errorf("command = %q, want notify-send", capturedName)
	}
	if len(capturedArgs) < 2 || capturedArgs[0] != "Test Title" || capturedArgs[1] != "Test Message" {
		t.Errorf("args = %v, want [Test Title, Test Message]", capturedArgs)
	}
}

func TestLocalNotifier_UnsupportedOS_FallsBack(t *testing.T) {
	// given: force unsupported OS
	n := &LocalNotifier{
		forceOS: "windows",
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			t.Error("should not invoke command for unsupported OS")
			return &fakeCmd{}
		},
	}

	// when
	err := n.Notify(context.Background(), "Title", "Message")

	// then: returns ErrUnsupportedOS, does not panic or invoke command
	if err != ErrUnsupportedOS {
		t.Errorf("err = %v, want ErrUnsupportedOS", err)
	}
}

func TestCmdNotifier_PlaceholderReplacement(t *testing.T) {
	// given
	var capturedShellCmd string
	n := &CmdNotifier{
		cmdTemplate: `curl -d "{title}: {message}" https://example.com`,
		makeCmd: func(ctx context.Context, name string, args ...string) cmdRunner {
			capturedShellCmd = args[len(args)-1] // last arg to "sh -c ..."
			return &fakeCmd{}
		},
	}

	// when
	err := n.Notify(context.Background(), "Paintress", "HIGH severity D-Mail")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedShellCmd, "Paintress") {
		t.Errorf("shell cmd should contain title, got: %s", capturedShellCmd)
	}
	if !strings.Contains(capturedShellCmd, "HIGH severity D-Mail") {
		t.Errorf("shell cmd should contain message, got: %s", capturedShellCmd)
	}
	if strings.Contains(capturedShellCmd, "{title}") {
		t.Error("shell cmd still contains {title} placeholder")
	}
	if strings.Contains(capturedShellCmd, "{message}") {
		t.Error("shell cmd still contains {message} placeholder")
	}
}

func TestNopNotifier_NoError(t *testing.T) {
	// given
	n := &NopNotifier{}

	// when
	err := n.Notify(context.Background(), "Any", "Thing")

	// then
	if err != nil {
		t.Errorf("NopNotifier should always return nil, got: %v", err)
	}
}

// fakeCmd implements cmdRunner for testing without executing real commands.
type fakeCmd struct {
	err error
}

func (f *fakeCmd) Run() error { return f.err }
