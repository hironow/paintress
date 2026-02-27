package paintress_test

import (
	"testing"

	"github.com/hironow/paintress"
)

func TestRunExpeditionCommand_Validate_Valid(t *testing.T) {
	// given
	cmd := paintress.RunExpeditionCommand{
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
	cmd := paintress.RunExpeditionCommand{}

	// when
	errs := cmd.Validate()

	// then
	if len(errs) == 0 {
		t.Fatal("expected validation error for missing RepoPath")
	}
}
