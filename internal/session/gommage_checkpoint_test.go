package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// white-box-reason: tests internal buildResumeContext and countCommitsInDir functions

func TestBuildResumeContext_WithCommits(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Helper()
			t.Fatalf("git %v: %v", args, err)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "initial")

	ctx := buildResumeContext(dir)
	if ctx == "" {
		t.Error("expected non-empty resume context")
	}
	if !contains(ctx, "initial") {
		t.Errorf("expected commit message in context, got: %s", ctx)
	}
}

func TestBuildResumeContext_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	ctx := buildResumeContext(dir)
	// Should still return the header, even with no commits
	if ctx == "" {
		t.Error("expected non-empty context even for empty repo")
	}
}

func TestCountCommitsInDir_WithCommits(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Run()
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a"), 0644)
	run("add", ".")
	run("commit", "-m", "first")
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b"), 0644)
	run("add", ".")
	run("commit", "-m", "second")

	got := countCommitsInDir(dir)
	if got != 2 {
		t.Errorf("expected 2 commits, got %d", got)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
