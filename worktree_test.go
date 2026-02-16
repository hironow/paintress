package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

// === localGitExecutor tests ===

func TestLocalGitExecutor_Git_StatusOnNewRepo(t *testing.T) {
	// given
	dir := t.TempDir()
	executor := &localGitExecutor{}
	ctx := context.Background()

	// init a git repo in the temp dir
	_, err := executor.Git(ctx, dir, "init")
	if err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// when
	out, err := executor.Git(ctx, dir, "status")

	// then
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	if !strings.Contains(string(out), "On branch") {
		t.Errorf("expected 'On branch' in output, got: %s", string(out))
	}
}

func TestLocalGitExecutor_Shell_EchoCommand(t *testing.T) {
	// given
	dir := t.TempDir()
	executor := &localGitExecutor{}
	ctx := context.Background()

	// when
	out, err := executor.Shell(ctx, dir, "echo hello")

	// then
	if err != nil {
		t.Fatalf("shell command failed: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "hello" {
		t.Errorf("expected 'hello', got: %q", got)
	}
}

// === containerGitExecutor (test-only) ===

// containerGitExecutor runs git commands inside a Docker container via testcontainers-go.
type containerGitExecutor struct {
	ctr testcontainers.Container
}

func (e *containerGitExecutor) Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	// Ensure the directory exists before running git commands.
	mkdirCmd := []string{"mkdir", "-p", dir}
	if exitCode, _, err := e.ctr.Exec(ctx, mkdirCmd, tcexec.Multiplexed()); err != nil || exitCode != 0 {
		return nil, fmt.Errorf("mkdir -p %s failed (exit %d): %w", dir, exitCode, err)
	}

	cmd := append([]string{"git", "-C", dir}, args...)
	exitCode, reader, err := e.ctr.Exec(ctx, cmd, tcexec.Multiplexed())
	if err != nil {
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read output failed: %w", err)
	}
	if exitCode != 0 {
		return out, fmt.Errorf("git exited with code %d: %s", exitCode, string(out))
	}
	return out, nil
}

func (e *containerGitExecutor) Shell(ctx context.Context, dir string, command string) ([]byte, error) {
	cmd := []string{"sh", "-c", fmt.Sprintf("cd %s && %s", dir, command)}
	exitCode, reader, err := e.ctr.Exec(ctx, cmd, tcexec.Multiplexed())
	if err != nil {
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read output failed: %w", err)
	}
	if exitCode != 0 {
		return out, fmt.Errorf("shell exited with code %d: %s", exitCode, string(out))
	}
	return out, nil
}

// setupGitContainer creates a running alpine/git container configured for git operations.
func setupGitContainer(t *testing.T, ctx context.Context) testcontainers.Container {
	t.Helper()

	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:      "alpine/git:latest",
			Entrypoint: []string{"/bin/sh", "-c"},
			Cmd:        []string{"sleep infinity"},
			WaitingFor: wait.ForExec([]string{"git", "--version"}).
				WithExitCodeMatcher(func(exitCode int) bool {
					return exitCode == 0
				}),
		},
		Started: true,
	}

	ctr, err := testcontainers.GenericContainer(ctx, req)
	testcontainers.CleanupContainer(t, ctr)
	if err != nil {
		t.Fatalf("failed to start git container: %v", err)
	}

	// Configure git user and default branch inside the container.
	for _, cmd := range [][]string{
		{"git", "config", "--global", "user.email", "test@example.com"},
		{"git", "config", "--global", "user.name", "Test User"},
		{"git", "config", "--global", "init.defaultBranch", "main"},
	} {
		exitCode, _, err := ctr.Exec(ctx, cmd, tcexec.Multiplexed())
		if err != nil {
			t.Fatalf("git config failed: %v", err)
		}
		if exitCode != 0 {
			t.Fatalf("git config exited with code %d", exitCode)
		}
	}

	return ctr
}

func TestContainerGitExecutor_Git_StatusOnNewRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in short mode")
	}

	// given
	ctx := context.Background()
	ctr := setupGitContainer(t, ctx)
	executor := &containerGitExecutor{ctr: ctr}
	workDir := "/tmp/test-repo"

	// init a repo inside the container
	_, err := executor.Git(ctx, workDir, "init")
	if err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// when
	out, err := executor.Git(ctx, workDir, "status")

	// then
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	if !strings.Contains(string(out), "On branch main") {
		t.Errorf("expected 'On branch main' in output, got: %s", string(out))
	}
}

func TestContainerGitExecutor_Shell_EchoCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in short mode")
	}

	// given
	ctx := context.Background()
	ctr := setupGitContainer(t, ctx)
	executor := &containerGitExecutor{ctr: ctr}

	// when
	out, err := executor.Shell(ctx, "/tmp", "echo hello")

	// then
	if err != nil {
		t.Fatalf("shell command failed: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "hello" {
		t.Errorf("expected 'hello', got: %q", got)
	}
}

// Verify that containerGitExecutor satisfies the GitExecutor interface at compile time.
var _ GitExecutor = (*containerGitExecutor)(nil)
