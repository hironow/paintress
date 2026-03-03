package platform_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hironow/paintress/internal/platform"
)

func containsStr(s, sub string) bool {
	return strings.Contains(s, sub)
}

func TestSetExtraWriter_DualWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("OpenFile error: %v", err)
	}
	defer f.Close()

	logger := platform.NewLogger(io.Discard, false)
	logger.SetExtraWriter(f)

	logger.Info("dual write test")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(content), "dual write test") {
		t.Error("log file should contain the message")
	}
}

func TestSetExtraWriter_NilSafe(t *testing.T) {
	logger := platform.NewLogger(io.Discard, false)
	// Should not panic when called with nil multiple times
	logger.SetExtraWriter(nil)
	logger.SetExtraWriter(nil)
}

func TestLogFunctions_NoPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	logger := platform.NewLogger(io.Discard, false)
	f, _ := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	logger.SetExtraWriter(f)
	defer f.Close()

	// These should not panic
	logger.Info("info %s", "test")
	logger.OK("ok %d", 42)
	logger.Warn("warn %v", true)
	logger.Error("error %s", "oops")
	logger.Info("qa %s", "check")
	logger.Info("exp %d", 1)
}

func TestLogFunctions_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	logger := platform.NewLogger(io.Discard, false)
	f, _ := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	logger.SetExtraWriter(f)

	logger.Info("hello from test")
	f.Close()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !containsStr(s, "INFO") {
		t.Errorf("log file should contain INFO prefix: %q", s)
	}
	if !containsStr(s, "hello from test") {
		t.Errorf("log file should contain message: %q", s)
	}
}

func TestLogFunctions_WithoutLogFile(t *testing.T) {
	logger := platform.NewLogger(io.Discard, false)

	// Should not panic even without log file
	logger.Info("no file")
	logger.Warn("no file")
	logger.Error("no file")
}

func TestLogFunctions_WritesToWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)

	logger.Info("writer test message")

	out := buf.String()
	if !containsStr(out, "writer test message") {
		t.Errorf("expected log output on writer, got %q", out)
	}
}

func TestLogFunctions_DoesNotWriteToStdout(t *testing.T) {
	// Capture stdout — should be empty
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	logger := platform.NewLogger(io.Discard, false)
	logger.Info("should not appear on stdout")

	_ = w.Close()
	os.Stdout = origStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected no stdout output from Info, got %q", string(out))
	}
}

func TestLogFunctions_NoColorCodes(t *testing.T) {
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)

	logger.Info("no color test")

	out := buf.String()
	if containsStr(out, "\033[") {
		t.Errorf("expected no ANSI color codes, got %q", out)
	}
	if !containsStr(out, "no color test") {
		t.Errorf("expected message content, got %q", out)
	}
}

func TestLogFunctions_QuietMode_SuppressesWriter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "quiet.log")

	// Quiet mode: pass io.Discard as out to suppress console output.
	logger := platform.NewLogger(io.Discard, false)
	f, _ := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	logger.SetExtraWriter(f)
	defer f.Close()

	logger.Info("quiet mode")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(content), "quiet mode") {
		t.Error("log file should still contain message in quiet mode")
	}
}

func TestLogger_Debug_Verbose(t *testing.T) {
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, true)

	logger.Debug("debug message")

	out := buf.String()
	if !containsStr(out, "debug message") {
		t.Errorf("expected debug message when verbose, got %q", out)
	}
}

func TestLogger_Debug_NotVerbose(t *testing.T) {
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)

	logger.Debug("debug message")

	out := buf.String()
	if len(out) != 0 {
		t.Errorf("expected no output when not verbose, got %q", out)
	}
}

func TestLogger_Writer(t *testing.T) {
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)

	if logger.Writer() != &buf {
		t.Error("Writer() should return the configured writer")
	}
}

// --- from edge_cases_test.go ---

func TestLogFunctions_ConcurrentLogging(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.log")
	logger := platform.NewLogger(io.Discard, false)
	f, _ := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	logger.SetExtraWriter(f)
	defer f.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(4)
		go func(n int) {
			defer wg.Done()
			logger.Info("concurrent info %d", n)
		}(i)
		go func(n int) {
			defer wg.Done()
			logger.Warn("concurrent warn %d", n)
		}(i)
		go func(n int) {
			defer wg.Done()
			logger.OK("concurrent ok %d", n)
		}(i)
		go func(n int) {
			defer wg.Done()
			logger.Error("concurrent error %d", n)
		}(i)
	}
	wg.Wait()

	content, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 200 {
		t.Errorf("expected 200 log lines (50*4), got %d", len(lines))
	}
}

func TestLogFunctions_ReinitLogFile(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "log1.log")
	path2 := filepath.Join(dir, "log2.log")

	logger := platform.NewLogger(io.Discard, false)
	f1, _ := os.OpenFile(path1, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	logger.SetExtraWriter(f1)
	logger.Info("to first file")
	f1.Close()
	f2, _ := os.OpenFile(path2, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	logger.SetExtraWriter(f2)
	logger.Info("to second file")
	f2.Close()

	content2, _ := os.ReadFile(path2)
	if !containsStr(string(content2), "to second file") {
		t.Error("second log file should contain second message")
	}
}

// --- from race_test.go ---

func TestLogger_ConcurrentSetExtraWriterAndWrite(t *testing.T) {
	logger := platform.NewLogger(io.Discard, false)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			logger.SetExtraWriter(&buf)
		}()
		go func(n int) {
			defer wg.Done()
			logger.Info("race test info %d", n)
			logger.Warn("race test warn %d", n)
		}(i)
		go func() {
			defer wg.Done()
			logger.SetExtraWriter(nil)
		}()
	}
	wg.Wait()

	// Clean up
	logger.SetExtraWriter(nil)
}
