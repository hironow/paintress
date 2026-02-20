package paintress

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWatchInbox_DetectsNewDMail(t *testing.T) {
	dir := t.TempDir()
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	var mu sync.Mutex
	var got DMail
	done := make(chan struct{}, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ready := make(chan struct{}, 1)
	go watchInbox(ctx, dir, func(dm DMail) {
		mu.Lock()
		got = dm
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
	}, ready)

	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("timeout waiting for watcher ready")
	}

	// Write a valid D-Mail file
	content := "---\nname: spec-my-42\nkind: specification\ndescription: test spec\n---\n\nBody text\n"
	os.WriteFile(filepath.Join(inboxDir, "spec-my-42.md"), []byte(content), 0644)

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout waiting for callback")
	}

	mu.Lock()
	defer mu.Unlock()
	if got.Name != "spec-my-42" {
		t.Errorf("name = %q, want spec-my-42", got.Name)
	}
	if got.Kind != "specification" {
		t.Errorf("kind = %q, want specification", got.Kind)
	}
}

func TestWatchInbox_StopsOnContextCancel(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "inbox"), 0755)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		watchInbox(ctx, dir, func(dm DMail) {}, nil)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watchInbox did not return after context cancel")
	}
}

func TestWatchInbox_IgnoresNonMdFiles(t *testing.T) {
	dir := t.TempDir()
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	var callCount int
	var mu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	ready := make(chan struct{}, 1)
	go watchInbox(ctx, dir, func(dm DMail) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}, ready)

	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("timeout waiting for watcher ready")
	}

	// Write non-.md files — should be ignored
	os.WriteFile(filepath.Join(inboxDir, "notes.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(inboxDir, ".hidden"), []byte("hello"), 0644)

	// Wait for context timeout (watcher should not fire)
	<-ctx.Done()

	mu.Lock()
	defer mu.Unlock()
	if callCount != 0 {
		t.Errorf("callback fired %d times for non-.md files, want 0", callCount)
	}
}

func TestWatchInbox_NoDirNoPanic(t *testing.T) {
	dir := t.TempDir()
	// No .expedition/inbox/ directory

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Should not panic — returns silently
	watchInbox(ctx, dir, func(dm DMail) {
		t.Error("callback should not fire when inbox dir does not exist")
	}, nil)
}

func TestWatchInbox_ParsesCorrectly(t *testing.T) {
	dir := t.TempDir()
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	var mu sync.Mutex
	var got DMail
	done := make(chan struct{}, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ready := make(chan struct{}, 1)
	go watchInbox(ctx, dir, func(dm DMail) {
		mu.Lock()
		got = dm
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
	}, ready)

	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("timeout waiting for watcher ready")
	}

	content := "---\nname: feedback-d-072\nkind: feedback\ndescription: arch drift detected\nseverity: high\nissues:\n  - MY-42\n  - MY-43\n---\n\n## Findings\n\nDrift in module X.\n"
	os.WriteFile(filepath.Join(inboxDir, "feedback-d-072.md"), []byte(content), 0644)

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout waiting for callback")
	}

	mu.Lock()
	defer mu.Unlock()
	if got.Name != "feedback-d-072" {
		t.Errorf("name = %q, want feedback-d-072", got.Name)
	}
	if got.Kind != "feedback" {
		t.Errorf("kind = %q, want feedback", got.Kind)
	}
	if got.Severity != "high" {
		t.Errorf("severity = %q, want high", got.Severity)
	}
	if len(got.Issues) != 2 {
		t.Errorf("issues = %v, want [MY-42 MY-43]", got.Issues)
	}
	if got.Body == "" {
		t.Error("body should not be empty")
	}
}
