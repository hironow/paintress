package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
)

func TestUpdatePRReviewGate_Integration(t *testing.T) {
	// given: build fake-gh and put it on PATH
	fakeGHDir := t.TempDir()
	fakeGH := filepath.Join(fakeGHDir, "gh")

	// fake-gh is a separate Go module; build from its directory
	fakeGHSrc := filepath.Join("..", "scenario", "testdata", "fake-gh")
	buildCmd := exec.Command("go", "build", "-o", fakeGH, ".")
	buildCmd.Dir = fakeGHSrc
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake-gh: %v\n%s", err, out)
	}

	editLog := filepath.Join(t.TempDir(), "edit.log")
	t.Setenv("FAKE_GH_EDIT_LOG", editLog)
	t.Setenv("PATH", fakeGHDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	logger := platform.NewLogger(os.Stderr, false)
	prURL := "https://github.com/test/repo/pull/42"
	status := domain.ReviewGateStatus{
		Passed: true,
		Cycle:  1,
		MaxCycles: 3,
	}

	// when
	err := session.UpdatePRReviewGate(context.Background(), prURL, status, logger)

	// then
	if err != nil {
		t.Fatalf("UpdatePRReviewGate: %v", err)
	}

	// Verify fake-gh recorded the edit
	data, readErr := os.ReadFile(editLog)
	if readErr != nil {
		t.Fatalf("read edit log: %v", readErr)
	}
	log := string(data)
	if !strings.Contains(log, "pr edit") {
		t.Errorf("expected 'pr edit' in log, got: %s", log)
	}
	if !strings.Contains(log, "--body") {
		t.Errorf("expected '--body' in log, got: %s", log)
	}
}
