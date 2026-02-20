package paintress

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWatchFlag_DetectsCurrentIssue(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	var mu sync.Mutex
	var gotIssue, gotTitle string
	done := make(chan struct{}, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ready := make(chan struct{}, 1)
	go watchFlag(ctx, dir, func(issue, title string) {
		mu.Lock()
		gotIssue = issue
		gotTitle = title
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
	}, ready)

	// Wait for watcher to be fully set up before writing
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("timeout waiting for watcher ready")
	}

	content := "current_issue: MY-239\ncurrent_title: flag watcher\n"
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout waiting for callback")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotIssue != "MY-239" {
		t.Errorf("issue = %q, want MY-239", gotIssue)
	}
	if gotTitle != "flag watcher" {
		t.Errorf("title = %q, want 'flag watcher'", gotTitle)
	}
}

func TestWatchFlag_StopsOnContextCancel(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		watchFlag(ctx, dir, func(issue, title string) {}, nil)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// ok — returned promptly
	case <-time.After(2 * time.Second):
		t.Fatal("watchFlag did not return after context cancel")
	}
}

func TestWatchFlag_DoesNotFireOnSameIssue(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	// Pre-write flag before watcher starts
	content := "current_issue: MY-239\ncurrent_title: same issue\n"
	os.WriteFile(filepath.Join(runDir, "flag.md"), []byte(content), 0644)

	var callCount int
	var mu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	watchFlag(ctx, dir, func(issue, title string) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}, nil)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 1 {
		t.Errorf("callback fired %d times, want exactly 1 (same issue should not re-fire)", callCount)
	}
}

func TestWatchFlag_DetectsIssueChange(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".expedition", ".run")
	os.MkdirAll(runDir, 0755)

	var mu sync.Mutex
	var issues []string
	firstFired := make(chan struct{}, 1)
	secondFired := make(chan struct{}, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ready := make(chan struct{}, 1)
	go watchFlag(ctx, dir, func(issue, title string) {
		mu.Lock()
		issues = append(issues, issue)
		count := len(issues)
		mu.Unlock()
		if count == 1 {
			select {
			case firstFired <- struct{}{}:
			default:
			}
		}
		if count >= 2 {
			select {
			case secondFired <- struct{}{}:
			default:
			}
		}
	}, ready)

	// Wait for watcher to be fully set up
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("timeout waiting for watcher ready")
	}

	// First issue
	os.WriteFile(filepath.Join(runDir, "flag.md"),
		[]byte("current_issue: MY-239\ncurrent_title: first\n"), 0644)

	// Wait for first callback before writing second
	select {
	case <-firstFired:
	case <-ctx.Done():
		t.Fatal("timeout waiting for first callback")
	}

	// Second issue
	os.WriteFile(filepath.Join(runDir, "flag.md"),
		[]byte("current_issue: MY-240\ncurrent_title: second\n"), 0644)

	select {
	case <-secondFired:
	case <-ctx.Done():
		t.Fatal("timeout waiting for second callback")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(issues) != 2 {
		t.Fatalf("expected 2 callbacks, got %d: %v", len(issues), issues)
	}
	if issues[0] != "MY-239" || issues[1] != "MY-240" {
		t.Errorf("issues = %v, want [MY-239, MY-240]", issues)
	}
}

func TestWatchFlag_NoFlagFile_NoPanic(t *testing.T) {
	dir := t.TempDir()
	// No .expedition directory at all

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Should not panic — returns immediately because runDir doesn't exist
	watchFlag(ctx, dir, func(issue, title string) {
		t.Error("callback should not fire when no flag file exists")
	}, nil)
}
