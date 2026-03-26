package session_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/session"
)

func TestArchiveReader_EmptyDir(t *testing.T) {
	// given: empty archive directory
	dir := t.TempDir()
	reader := session.NewArchiveReader(dir)

	// when
	dmails, err := reader.ReadArchiveDMails(context.Background())

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dmails) != 0 {
		t.Errorf("expected 0 dmails, got %d", len(dmails))
	}
}

func TestArchiveReader_NonExistentDir(t *testing.T) {
	// given: directory that doesn't exist
	reader := session.NewArchiveReader(filepath.Join(t.TempDir(), "nonexistent"))

	// when
	dmails, err := reader.ReadArchiveDMails(context.Background())

	// then: no error (graceful)
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got: %v", err)
	}
	if dmails != nil {
		t.Errorf("expected nil dmails, got %v", dmails)
	}
}

func TestArchiveReader_ParsesValidDMails(t *testing.T) {
	// given: archive with a spec D-Mail
	dir := t.TempDir()
	content := "---\ndmail-schema-version: \"1\"\nname: spec-test\nkind: specification\ndescription: Test wave\nwave:\n  id: \"test-w1\"\n  steps:\n    - id: \"s1\"\n      title: \"Step one\"\n---\n# Body\n"
	if err := os.WriteFile(filepath.Join(dir, "spec-test.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	reader := session.NewArchiveReader(dir)

	// when
	dmails, err := reader.ReadArchiveDMails(context.Background())

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dmails) != 1 {
		t.Fatalf("expected 1 dmail, got %d", len(dmails))
	}
	if dmails[0].Name != "spec-test" {
		t.Errorf("name = %q, want spec-test", dmails[0].Name)
	}
	if dmails[0].Wave == nil {
		t.Fatal("expected wave reference")
	}
	if dmails[0].Wave.ID != "test-w1" {
		t.Errorf("wave.id = %q, want test-w1", dmails[0].Wave.ID)
	}
}

func TestArchiveReader_SkipsInvalidFiles(t *testing.T) {
	// given: archive with one valid and one invalid file
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "valid.md"), []byte("---\nname: valid\nkind: report\ndescription: ok\n---\n"), 0644)
	os.WriteFile(filepath.Join(dir, "invalid.md"), []byte("not yaml frontmatter"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignored"), 0644)

	reader := session.NewArchiveReader(dir)

	// when
	dmails, err := reader.ReadArchiveDMails(context.Background())

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dmails) != 1 {
		t.Errorf("expected 1 valid dmail (skip invalid + non-.md), got %d", len(dmails))
	}
}
