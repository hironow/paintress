package session

// white-box-reason: session internals: tests unexported DevServer struct state inspection

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/platform"
)

func TestNewDevServer(t *testing.T) {
	ds := NewDevServer("npm run dev", "http://localhost:3000", "/some/dir", "/tmp/dev.log", platform.NewLogger(io.Discard, false))

	if ds.cmd != "npm run dev" {
		t.Errorf("cmd = %q", ds.cmd)
	}
	if ds.url != "http://localhost:3000" {
		t.Errorf("url = %q", ds.url)
	}
	if ds.dir != "/some/dir" {
		t.Errorf("dir = %q", ds.dir)
	}
	if ds.logPath != "/tmp/dev.log" {
		t.Errorf("logPath = %q", ds.logPath)
	}
	if ds.running {
		t.Error("should not be running initially")
	}
}

func TestDevServer_Start_EmptyCmd(t *testing.T) {
	ds := NewDevServer("", "http://localhost:3000", t.TempDir(), "/dev/null", platform.NewLogger(io.Discard, false))

	err := ds.Start(context.Background())
	if err == nil {
		t.Fatal("expected error for empty cmd, got nil")
	}
	if !strings.Contains(err.Error(), "dev_cmd is empty") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "dev_cmd is empty")
	}
}

func TestDevServer_StopWhenNotRunning(t *testing.T) {
	ds := NewDevServer("echo", "http://localhost:3000", t.TempDir(), "/dev/null", platform.NewLogger(io.Discard, false))
	// Stop on a non-running server should not panic
	ds.Stop()
}
