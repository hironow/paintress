package main

import (
	"context"
	"strings"
	"testing"
)

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
