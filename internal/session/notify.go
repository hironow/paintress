package session

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/hironow/paintress"
)

// cmdRunner abstracts exec.Cmd.Run for testing.
type cmdRunner interface {
	Run() error
}

// cmdFactory creates a cmdRunner from command name and args.
type cmdFactory func(ctx context.Context, name string, args ...string) cmdRunner

func defaultCmdFactory(ctx context.Context, name string, args ...string) cmdRunner {
	return exec.CommandContext(ctx, name, args...)
}

// LocalNotifier sends desktop notifications using OS-native commands.
// darwin: osascript, linux: notify-send, others: returns ErrUnsupportedOS.
type LocalNotifier struct {
	makeCmd cmdFactory
	forceOS string // for testing; empty = use runtime.GOOS
}

func (n *LocalNotifier) os() string {
	if n.forceOS != "" {
		return n.forceOS
	}
	return runtime.GOOS
}

func (n *LocalNotifier) factory() cmdFactory {
	if n.makeCmd != nil {
		return n.makeCmd
	}
	return defaultCmdFactory
}

func (n *LocalNotifier) Notify(ctx context.Context, title, message string) error {
	mk := n.factory()

	switch n.os() {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q sound name "Funk"`, message, title)
		return mk(ctx, "osascript", "-e", script).Run()
	case "linux":
		return mk(ctx, "notify-send", title, message).Run()
	default:
		return paintress.ErrUnsupportedOS
	}
}

// CmdNotifier executes a user-provided shell command for notifications.
// The template may contain {title} and {message} placeholders.
type CmdNotifier struct {
	cmdTemplate string
	makeCmd     cmdFactory
}

func NewCmdNotifier(cmdTemplate string) *CmdNotifier {
	return &CmdNotifier{cmdTemplate: cmdTemplate}
}

func (n *CmdNotifier) factory() cmdFactory {
	if n.makeCmd != nil {
		return n.makeCmd
	}
	return defaultCmdFactory
}

const notifyTimeout = 30 * time.Second

func (n *CmdNotifier) Notify(ctx context.Context, title, message string) error {
	if n.cmdTemplate == "" {
		return fmt.Errorf("notify: empty command template")
	}
	ctx, cancel := context.WithTimeout(ctx, notifyTimeout)
	defer cancel()
	expanded := strings.ReplaceAll(n.cmdTemplate, "{title}", shellQuote(title))
	expanded = strings.ReplaceAll(expanded, "{message}", shellQuote(message))
	return n.factory()(ctx, shellName(), shellFlag(), expanded).Run()
}

