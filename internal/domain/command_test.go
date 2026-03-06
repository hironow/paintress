package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestNewRunExpeditionCommand(t *testing.T) {
	// given
	rp, err := domain.NewRepoPath("/tmp/repo")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// when
	cmd := domain.NewRunExpeditionCommand(rp)

	// then
	if cmd.RepoPath().String() != "/tmp/repo" {
		t.Errorf("expected /tmp/repo, got %s", cmd.RepoPath().String())
	}
}

func TestNewInitCommand(t *testing.T) {
	// given
	rp, err := domain.NewRepoPath("/tmp/repo")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// when
	cmd := domain.NewInitCommand(rp, domain.NewTeam("MY"), domain.NewProject("Hades"))

	// then
	if cmd.RepoPath().String() != "/tmp/repo" {
		t.Errorf("expected /tmp/repo, got %s", cmd.RepoPath().String())
	}
	if cmd.Team().String() != "MY" {
		t.Errorf("expected MY, got %s", cmd.Team().String())
	}
	if cmd.Project().String() != "Hades" {
		t.Errorf("expected Hades, got %s", cmd.Project().String())
	}
}

func TestNewArchivePruneCommand(t *testing.T) {
	// given
	rp, err := domain.NewRepoPath("/tmp/repo")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	days, err := domain.NewDays(30)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// when
	cmd := domain.NewArchivePruneCommand(rp, days, true)

	// then
	if cmd.RepoPath().String() != "/tmp/repo" {
		t.Errorf("expected /tmp/repo, got %s", cmd.RepoPath().String())
	}
	if cmd.Days().Int() != 30 {
		t.Errorf("expected 30, got %d", cmd.Days().Int())
	}
	if !cmd.Execute() {
		t.Error("expected Execute to be true")
	}
}

func TestNewRebuildCommand(t *testing.T) {
	// given
	rp, err := domain.NewRepoPath("/tmp/repo")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// when
	cmd := domain.NewRebuildCommand(rp)

	// then
	if cmd.RepoPath().String() != "/tmp/repo" {
		t.Errorf("expected /tmp/repo, got %s", cmd.RepoPath().String())
	}
}
