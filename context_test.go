package paintress

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadContextFiles_ReadsMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	os.WriteFile(filepath.Join(ctxDir, "architecture.md"), []byte("Use hexagonal architecture.\n"), 0644)
	os.WriteFile(filepath.Join(ctxDir, "naming.md"), []byte("Use snake_case for API fields.\n"), 0644)

	result, err := ReadContextFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "architecture") {
		t.Error("expected context to contain 'architecture' header")
	}
	if !strings.Contains(result, "Use hexagonal architecture.") {
		t.Error("expected context to contain architecture.md content")
	}
	if !strings.Contains(result, "naming") {
		t.Error("expected context to contain 'naming' header")
	}
	if !strings.Contains(result, "Use snake_case for API fields.") {
		t.Error("expected context to contain naming.md content")
	}
}

func TestReadContextFiles_EmptyWhenNoDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := ReadContextFiles(dir)

	if err != nil {
		t.Errorf("missing directory should not be an error, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string when no context dir, got %q", result)
	}
}

func TestReadContextFiles_ErrorOnPermissionDenied(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	// Write a valid file, then remove read permission on the directory
	os.WriteFile(filepath.Join(ctxDir, "rules.md"), []byte("important rules\n"), 0644)
	os.Chmod(ctxDir, 0000)
	t.Cleanup(func() { os.Chmod(ctxDir, 0755) })

	_, err := ReadContextFiles(dir)

	if err == nil {
		t.Error("expected error for permission-denied directory, got nil")
	}
}

func TestReadContextFiles_IgnoresNonMarkdown(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	os.WriteFile(filepath.Join(ctxDir, "notes.md"), []byte("important\n"), 0644)
	os.WriteFile(filepath.Join(ctxDir, "data.json"), []byte(`{"key":"val"}`), 0644)
	os.MkdirAll(filepath.Join(ctxDir, "subdir"), 0755)

	result, err := ReadContextFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "important") {
		t.Error("expected .md file to be included")
	}
	if strings.Contains(result, "key") {
		t.Error(".json files should be excluded")
	}
}

func TestReadContextFiles_OrdersByFilename(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	os.WriteFile(filepath.Join(ctxDir, "b.md"), []byte("second\n"), 0644)
	os.WriteFile(filepath.Join(ctxDir, "a.md"), []byte("first\n"), 0644)

	result, err := ReadContextFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	firstIdx := strings.Index(result, "### a")
	secondIdx := strings.Index(result, "### b")
	if firstIdx == -1 || secondIdx == -1 {
		t.Fatalf("expected headers for a.md and b.md, got: %q", result)
	}
	if firstIdx >= secondIdx {
		t.Errorf("expected a.md before b.md, got indices %d >= %d", firstIdx, secondIdx)
	}
}

func TestReadContextFiles_EmptyFileStillCreatesHeader(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	os.WriteFile(filepath.Join(ctxDir, "empty.md"), []byte(""), 0644)

	result, err := ReadContextFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "### empty") {
		t.Error("expected header for empty.md even when file is empty")
	}
}
