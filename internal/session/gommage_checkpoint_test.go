package session

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
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
	if !containsStr(ctx, "initial") {
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

func TestCleanOrphanWorktrees_NoOrphans(t *testing.T) {
	// white-box-reason: tests cleanOrphanWorktrees on Paintress struct
	continent := t.TempDir()
	os.MkdirAll(filepath.Join(continent, domain.StateDir, ".run", "logs"), 0755)
	// Initialize git repo so git worktree list works
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = continent
		cmd.Run()
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	os.WriteFile(filepath.Join(continent, "dummy.go"), []byte("package main"), 0644)
	run("add", ".")
	run("commit", "-m", "init")

	cfg := domain.Config{Continent: continent, Model: "opus"}
	logger := platform.NewLogger(io.Discard, false)
	p := NewPaintress(cfg, logger, io.Discard, io.Discard, nil, nil, nil, nil)
	// Should not panic with no worktrees
	p.cleanOrphanWorktrees()
}

func TestSaveCheckpoint_EmitsEvent(t *testing.T) {
	// white-box-reason: tests saveCheckpoint method on Paintress struct
	continent := t.TempDir()
	os.MkdirAll(filepath.Join(continent, domain.StateDir, ".run", "logs"), 0755)
	// Create git repo for countCommitsInDir
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = continent
		cmd.Run()
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	os.WriteFile(filepath.Join(continent, "file.go"), []byte("package main"), 0644)
	run("add", ".")
	run("commit", "-m", "init")

	cfg := domain.Config{Continent: continent, Model: "opus"}
	logger := platform.NewLogger(io.Discard, false)
	p := NewPaintress(cfg, logger, io.Discard, io.Discard, nil, nil, nil, nil)
	// Should not panic — emitter is NopExpeditionEventEmitter
	p.saveCheckpoint(1, CheckpointSubprocessStart, continent)
}

// containsStr and indexOfStr are defined in test_helpers_test.go
