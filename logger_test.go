package paintress

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestInitLogFile_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	err := InitLogFile(path)
	if err != nil {
		t.Fatalf("InitLogFile error: %v", err)
	}
	defer CloseLogFile()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("log file should be created")
	}
}

func TestInitLogFile_InvalidPath(t *testing.T) {
	err := InitLogFile("/nonexistent/dir/test.log")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestCloseLogFile_WhenNotOpen(t *testing.T) {
	// Ensure no previous log file is open
	CloseLogFile()
	// Should not panic when called without InitLogFile
	CloseLogFile()
}

func TestLogFunctions_NoPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	InitLogFile(path)
	defer CloseLogFile()

	// These should not panic
	LogInfo("info %s", "test")
	LogOK("ok %d", 42)
	LogWarn("warn %v", true)
	LogError("error %s", "oops")
	LogQA("qa %s", "check")
	LogExp("exp %d", 1)
}

func TestLogFunctions_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	InitLogFile(path)

	LogInfo("hello from test")
	CloseLogFile()

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
	// Ensure no log file is open
	CloseLogFile()

	// Should not panic even without log file
	LogInfo("no file")
	LogWarn("no file")
	LogError("no file")
}

func TestLogFunctions_WritesToStderr(t *testing.T) {
	CloseLogFile()

	// Capture stderr
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	LogInfo("stderr test message")

	_ = w.Close()
	os.Stderr = origStderr

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !containsStr(string(out), "stderr test message") {
		t.Errorf("expected log output on stderr, got %q", string(out))
	}
}

func TestLogFunctions_DoesNotWriteToStdout(t *testing.T) {
	CloseLogFile()

	// Capture stdout — should be empty
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	LogInfo("should not appear on stdout")

	_ = w.Close()
	os.Stdout = origStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected no stdout output from LogInfo, got %q", string(out))
	}
}

func TestLogFunctions_NoColorWhenNotTTY(t *testing.T) {
	CloseLogFile()

	// Pipe is not a TTY, so output should have no ANSI escape codes
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	LogInfo("no color test")

	_ = w.Close()
	os.Stderr = origStderr

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if containsStr(string(out), "\033[") {
		t.Errorf("expected no ANSI color codes when stderr is a pipe, got %q", string(out))
	}
	if !containsStr(string(out), "no color test") {
		t.Errorf("expected message content, got %q", string(out))
	}
}

func TestLogFunctions_QuietMode_SuppressesStderr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "quiet.log")
	InitLogFile(path)
	defer CloseLogFile()

	t.Setenv("PAINTRESS_QUIET", "1")

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	LogInfo("quiet mode")

	_ = w.Close()
	os.Stderr = origStderr

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected no stderr output in quiet mode, got %q", string(out))
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(content), "quiet mode") {
		t.Error("log file should still contain message in quiet mode")
	}
}
