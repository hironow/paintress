package usecase

import (
	"context"
	"io"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

func TestRunExpeditions_InvalidCommand(t *testing.T) {
	// given: empty RepoPath
	cmd := domain.RunExpeditionCommand{}

	// when
	exitCode, err := RunExpeditions(context.Background(), cmd, domain.Config{}, platform.NewLogger(io.Discard, false), io.Discard, nil, nil)

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
	cmd := domain.ArchivePruneCommand{}

	// when
	_, err := ArchivePrune(cmd)

	// then
	if err == nil {
		t.Fatal("expected validation error for empty RepoPath")
	}
}

func TestArchivePrune_InvalidDays(t *testing.T) {
	// given: Days <= 0
	cmd := domain.ArchivePruneCommand{RepoPath: "/tmp", Days: 0}

	// when
	_, err := ArchivePrune(cmd)

	// then
	if err == nil {
		t.Fatal("expected validation error for non-positive Days")
	}
}
