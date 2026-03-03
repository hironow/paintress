package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestRunExpeditionCommand_Validate_Valid(t *testing.T) {
	// given
	cmd := domain.RunExpeditionCommand{
		RepoPath: "/tmp/repo",
	}

	// when
	errs := cmd.Validate()

	// then
	if len(errs) > 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestRunExpeditionCommand_Validate_MissingRepoPath(t *testing.T) {
	// given
	cmd := domain.RunExpeditionCommand{}

	// when
	errs := cmd.Validate()

	// then
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing RepoPath")
	}
}

func TestInitCommand_Validate_Valid(t *testing.T) {
	// given
	cmd := domain.InitCommand{
		RepoPath: "/tmp/repo",
	}

	// when
	errs := cmd.Validate()

	// then
	if len(errs) > 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestInitCommand_Validate_MissingRepoPath(t *testing.T) {
	// given
	cmd := domain.InitCommand{}

	// when
	errs := cmd.Validate()

	// then
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing RepoPath")
	}
}

func TestArchivePruneCommand_Validate_Valid(t *testing.T) {
	// given
	cmd := domain.ArchivePruneCommand{
		RepoPath: "/tmp/repo",
		Days:     30,
	}

	// when
	errs := cmd.Validate()

	// then
	if len(errs) > 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestArchivePruneCommand_Validate_InvalidDays(t *testing.T) {
	// given
	cmd := domain.ArchivePruneCommand{
		RepoPath: "/tmp/repo",
		Days:     0,
	}

	// when
	errs := cmd.Validate()

	// then
	if len(errs) == 0 {
		t.Fatal("expected validation error for non-positive Days")
	}
}
