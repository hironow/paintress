package session

// white-box-reason: session internals: tests ReadContextFiles size guard constants and behavior

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadContextFiles_MaxContextFileBytes_ConstantDefined(t *testing.T) {
	// then: constant must be positive
	if MaxContextFileBytes <= 0 {
		t.Errorf("MaxContextFileBytes must be positive, got %d", MaxContextFileBytes)
	}
}

func TestReadContextFiles_MaxContextTotalBytes_ConstantDefined(t *testing.T) {
	// then: MaxContextTotalBytes must be positive and >= MaxContextFileBytes
	if MaxContextTotalBytes <= 0 {
		t.Errorf("MaxContextTotalBytes must be positive, got %d", MaxContextTotalBytes)
	}
	if MaxContextTotalBytes < MaxContextFileBytes {
		t.Errorf("MaxContextTotalBytes (%d) must be >= MaxContextFileBytes (%d)", MaxContextTotalBytes, MaxContextFileBytes)
	}
}

func TestReadContextFiles_SkipsOversizedFile(t *testing.T) {
	// given: a context directory with one oversized file and one normal file
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	// Write a file that exceeds MaxContextFileBytes
	bigContent := strings.Repeat("x", int(MaxContextFileBytes)+1)
	os.WriteFile(filepath.Join(ctxDir, "aa-big.md"), []byte(bigContent), 0644)

	// Write a normal file sorted after the big one
	os.WriteFile(filepath.Join(ctxDir, "zz-small.md"), []byte("small content\n"), 0644)

	// given: a logger that captures warnings
	wl := &testWarnLogger{}

	// when
	ctx, err := readContextFilesWithLogger(dir, wl)

	// then: no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// oversized content must not appear
	if strings.Contains(ctx, bigContent[:100]) {
		t.Error("oversized file content should not appear in context output")
	}
	// small file must still be included
	if !strings.Contains(ctx, "small content") {
		t.Error("small file content should appear in context output")
	}
	// a warning must be logged mentioning the oversized file
	if !wl.containsAny("aa-big", "skip", "large", "size", "exceed") {
		t.Errorf("expected a warning about oversized file, got: %v", wl.messages)
	}
}

func TestReadContextFiles_StopsAtTotalSizeLimit(t *testing.T) {
	// given: multiple files whose combined size exceeds MaxContextTotalBytes
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	// Each file is half of MaxContextFileBytes — small enough to pass per-file guard
	chunkSize := int(MaxContextFileBytes) / 2
	// Write enough files to exceed MaxContextTotalBytes
	numFiles := int(MaxContextTotalBytes)/chunkSize + 2
	for i := range numFiles {
		fname := filepath.Join(ctxDir, fmt.Sprintf("%02d.md", i))
		os.WriteFile(fname, []byte(strings.Repeat("y", chunkSize)), 0644)
	}

	// when
	ctx, err := readContextFilesWithLogger(dir, &testWarnLogger{})

	// then: no error, output is bounded
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Allow some overhead for headers ("### filename\n\n") but output must stay near the limit
	maxAllowed := int(MaxContextTotalBytes) + 4096 // generous margin for section headers
	if len(ctx) > maxAllowed {
		t.Errorf("output size %d exceeds allowed maximum %d", len(ctx), maxAllowed)
	}
}

// testWarnLogger captures Warn calls for assertion.
type testWarnLogger struct {
	messages []string
}

func (l *testWarnLogger) Info(format string, args ...any) {}
func (l *testWarnLogger) OK(format string, args ...any)   {}
func (l *testWarnLogger) Warn(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf(format, args...))
}
func (l *testWarnLogger) Error(format string, args ...any) {}
func (l *testWarnLogger) Debug(format string, args ...any) {}

// containsAny returns true if any captured message contains one of the substrings.
func (l *testWarnLogger) containsAny(substrs ...string) bool {
	for _, msg := range l.messages {
		lower := strings.ToLower(msg)
		for _, s := range substrs {
			if strings.Contains(lower, strings.ToLower(s)) {
				return true
			}
		}
	}
	return false
}
