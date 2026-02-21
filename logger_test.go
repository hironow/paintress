package paintress

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestSetLogFile_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	logger := NewLogger(io.Discard, false)
	err := logger.SetLogFile(path)
	if err != nil {
		t.Fatalf("SetLogFile error: %v", err)
	}
	defer logger.CloseLogFile()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("log file should be created")
	}
}

func TestSetLogFile_InvalidPath(t *testing.T) {
	logger := NewLogger(io.Discard, false)
	err := logger.SetLogFile("/nonexistent/dir/test.log")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestCloseLogFile_WhenNotOpen(t *testing.T) {
	logger := NewLogger(io.Discard, false)
	// Should not panic when called without SetLogFile
	logger.CloseLogFile()
	logger.CloseLogFile()
}

func TestLogFunctions_NoPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	logger := NewLogger(io.Discard, false)
	logger.SetLogFile(path)
	defer logger.CloseLogFile()

	// These should not panic
	logger.Info("info %s", "test")
	logger.OK("ok %d", 42)
	logger.Warn("warn %v", true)
	logger.Error("error %s", "oops")
	logger.QA("qa %s", "check")
	logger.Exp("exp %d", 1)
}

func TestLogFunctions_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	logger := NewLogger(io.Discard, false)
	logger.SetLogFile(path)

	logger.Info("hello from test")
	logger.CloseLogFile()

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
	logger := NewLogger(io.Discard, false)

	// Should not panic even without log file
	logger.Info("no file")
	logger.Warn("no file")
	logger.Error("no file")
}

func TestLogFunctions_WritesToWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, false)

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

	logger := NewLogger(io.Discard, false)
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
	logger := NewLogger(&buf, false)

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

	var buf bytes.Buffer
	logger := NewQuietLogger(&buf)
	logger.SetLogFile(path)
	defer logger.CloseLogFile()

	logger.Info("quiet mode")

	out := buf.String()
	if len(out) != 0 {
		t.Errorf("expected no writer output in quiet mode, got %q", out)
	}

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
	logger := NewLogger(&buf, true)

	logger.Debug("debug message")

	out := buf.String()
	if !containsStr(out, "debug message") {
		t.Errorf("expected debug message when verbose, got %q", out)
	}
}

func TestLogger_Debug_NotVerbose(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, false)

	logger.Debug("debug message")

	out := buf.String()
	if len(out) != 0 {
		t.Errorf("expected no output when not verbose, got %q", out)
	}
}

func TestLogger_Writer(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, false)

	if logger.Writer() != &buf {
		t.Error("Writer() should return the configured writer")
	}
}
