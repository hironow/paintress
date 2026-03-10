//go:build e2e

package e2e

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestE2E_PaintressInit verifies that `paintress init` creates the expected
// directory structure in a clean working directory.
func TestE2E_PaintressInit(t *testing.T) {
	dir := t.TempDir()

	// given: a git repository
	git := exec.Command("git", "init", dir)
	if err := git.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	// when: run paintress init (requires repo path as positional arg)
	cmd := exec.Command("paintress", "init", "--lang", "en", dir)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("paintress init: %v\n%s", err, out)
	}

	// then: expedition directory structure exists
	for _, sub := range []string{
		".expedition",
		".expedition/inbox",
		".expedition/outbox",
		".expedition/archive",
		".expedition/.run",
	} {
		path := filepath.Join(dir, sub)
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected directory %s to exist", sub)
		}
	}
}
