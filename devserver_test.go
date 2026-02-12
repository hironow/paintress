package main

import (
	"testing"
)

func TestNewDevServer(t *testing.T) {
	ds := NewDevServer("npm run dev", "http://localhost:3000", "/some/dir", "/tmp/dev.log")

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

func TestDevServer_StopWhenNotRunning(t *testing.T) {
	ds := NewDevServer("echo", "http://localhost:3000", t.TempDir(), "/dev/null")
	// Stop on a non-running server should not panic
	ds.Stop()
}
