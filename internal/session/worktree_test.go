package session

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockGitExecutor is a test double for port.GitExecutor.
type mockGitExecutor struct {
	gitFn   func(ctx context.Context, dir string, args ...string) ([]byte, error)
	shellFn func(ctx context.Context, dir string, command string) ([]byte, error)
}

func (m *mockGitExecutor) Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	if m.gitFn != nil {
		return m.gitFn(ctx, dir, args...)
	}
	return nil, nil
}

func (m *mockGitExecutor) Shell(ctx context.Context, dir string, command string) ([]byte, error) {
	if m.shellFn != nil {
		return m.shellFn(ctx, dir, command)
	}
	return nil, nil
}

func TestValidateBaseBranch_NonexistentBranch(t *testing.T) {
	// given
	git := &mockGitExecutor{
		gitFn: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("fatal: Needed a single revision")
		},
	}

	// when
	err := ValidateBaseBranch(context.Background(), git, "/tmp/repo", "mian")

	// then
	if err == nil {
		t.Fatal("expected error for nonexistent branch")
	}
	if !strings.Contains(err.Error(), "mian") {
		t.Errorf("error should mention branch name, got: %v", err)
	}
}

func TestValidateBaseBranch_ExistingBranch(t *testing.T) {
	// given
	git := &mockGitExecutor{
		gitFn: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			return []byte("abc123\n"), nil
		},
	}

	// when
	err := ValidateBaseBranch(context.Background(), git, "/tmp/repo", "main")

	// then
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
