package session

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/eventsource"
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
	p.cleanOrphanWorktrees(nil)
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

func newPaintressWithScanner(t *testing.T, continent string) *Paintress {
	t.Helper()
	os.MkdirAll(filepath.Join(continent, domain.StateDir, ".run", "logs"), 0755)
	os.MkdirAll(domain.EventsDir(continent), 0755)
	cfg := domain.Config{Continent: continent, Model: "opus"}
	logger := platform.NewLogger(io.Discard, false)
	p := NewPaintress(cfg, logger, io.Discard, io.Discard, nil, nil, nil, nil)
	p.checkpointScanner = eventsource.NewCheckpointScanner(continent)
	return p
}

func TestResumeIncompleteExpeditions_NoEvents(t *testing.T) {
	// white-box-reason: tests resumeIncompleteExpeditions on Paintress struct
	continent := t.TempDir()
	p := newPaintressWithScanner(t, continent)

	result := p.resumeIncompleteExpeditions()
	if len(result) != 0 {
		t.Errorf("expected 0 incomplete expeditions, got %d", len(result))
	}
}

func TestResumeIncompleteExpeditions_WithCheckpointNoCompletion(t *testing.T) {
	// white-box-reason: tests resumeIncompleteExpeditions with simulated events
	continent := t.TempDir()
	p := newPaintressWithScanner(t, continent)

	// Create a worktree dir that the checkpoint references
	wtDir := filepath.Join(t.TempDir(), "paintress-wt-test")
	os.MkdirAll(wtDir, 0755)

	// Write a checkpoint event (no completion event)
	eventsDir := domain.EventsDir(continent)
	checkpointEvent := `{"id":"evt1","type":"expedition.checkpoint","timestamp":"2026-03-27T00:00:00Z","data":{"expedition":5,"phase":"subprocess_started","work_dir":"` + wtDir + `","commit_count":1}}`
	os.WriteFile(filepath.Join(eventsDir, "2026-03-27.jsonl"), []byte(checkpointEvent+"\n"), 0644)

	result := p.resumeIncompleteExpeditions()
	if len(result) != 1 {
		t.Fatalf("expected 1 incomplete expedition, got %d", len(result))
	}
	if result[0].Expedition != 5 {
		t.Errorf("expected expedition 5, got %d", result[0].Expedition)
	}
	if result[0].WorkDir != wtDir {
		t.Errorf("expected workdir %s, got %s", wtDir, result[0].WorkDir)
	}
}

func TestResumeIncompleteExpeditions_CompletedExpeditionIsExcluded(t *testing.T) {
	// white-box-reason: completed expeditions should not appear as incomplete
	continent := t.TempDir()
	p := newPaintressWithScanner(t, continent)

	wtDir := filepath.Join(t.TempDir(), "paintress-wt-done")
	os.MkdirAll(wtDir, 0755)

	eventsDir := domain.EventsDir(continent)
	events := `{"id":"evt1","type":"expedition.checkpoint","timestamp":"2026-03-27T00:00:00Z","data":{"expedition":3,"phase":"subprocess_started","work_dir":"` + wtDir + `","commit_count":1}}
{"id":"evt2","type":"expedition.completed","timestamp":"2026-03-27T00:01:00Z","data":{"expedition":3,"status":"success"}}`
	os.WriteFile(filepath.Join(eventsDir, "2026-03-27.jsonl"), []byte(events+"\n"), 0644)

	result := p.resumeIncompleteExpeditions()
	if len(result) != 0 {
		t.Errorf("expected 0 (completed expedition excluded), got %d", len(result))
	}
}

func TestResumeIncompleteExpeditions_MissingWorktreeIsSkipped(t *testing.T) {
	// white-box-reason: checkpoint with missing worktree should be skipped
	continent := t.TempDir()
	p := newPaintressWithScanner(t, continent)

	eventsDir := domain.EventsDir(continent)
	events := `{"id":"evt1","type":"expedition.checkpoint","timestamp":"2026-03-27T00:00:00Z","data":{"expedition":7,"phase":"subprocess_started","work_dir":"/nonexistent/path","commit_count":0}}`
	os.WriteFile(filepath.Join(eventsDir, "2026-03-27.jsonl"), []byte(events+"\n"), 0644)

	result := p.resumeIncompleteExpeditions()
	if len(result) != 0 {
		t.Errorf("expected 0 (missing worktree skipped), got %d", len(result))
	}
}

// containsStr and indexOfStr are defined in test_helpers_test.go
