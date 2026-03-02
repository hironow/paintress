package usecase

import (
	"context"
	"io"
	"testing"

	"github.com/hironow/paintress"
)

func TestRunExpeditions_InvalidCommand(t *testing.T) {
	// given: empty RepoPath
	cmd := paintress.RunExpeditionCommand{}

	// when
	exitCode, err := RunExpeditions(context.Background(), cmd, paintress.Config{}, paintress.NewLogger(io.Discard, false), io.Discard, nil, nil)

	// then
	if err == nil {
		t.Fatal("expected validation error for empty RepoPath")
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestArchivePrune_InvalidCommand(t *testing.T) {
	// given: empty RepoPath
	cmd := paintress.ArchivePruneCommand{}

	// when
	_, err := ArchivePrune(cmd)

	// then
	if err == nil {
		t.Fatal("expected validation error for empty RepoPath")
	}
}

func TestArchivePrune_InvalidDays(t *testing.T) {
	// given: Days <= 0
	cmd := paintress.ArchivePruneCommand{RepoPath: "/tmp", Days: 0}

	// when
	_, err := ArchivePrune(cmd)

	// then
	if err == nil {
		t.Fatal("expected validation error for non-positive Days")
	}
}
