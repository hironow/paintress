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

func TestWatchInbox_DetectsWriteToExistingFile(t *testing.T) {
	dir := t.TempDir()
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	// Pre-create an empty file BEFORE watcher starts.
	// This means no Create event will fire — only Write events when content is added.
	filePath := filepath.Join(inboxDir, "spec-overwrite-42.md")
	os.WriteFile(filePath, []byte{}, 0644)

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

	// Overwrite with valid content — fires Write event only (no Create)
	content := "---\nname: spec-overwrite-42\nkind: specification\ndescription: overwrite test\n---\n\nBody\n"
	os.WriteFile(filePath, []byte(content), 0644)

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout — Write event on existing file was not handled")
	}

	mu.Lock()
	defer mu.Unlock()
	if got.Name != "spec-overwrite-42" {
		t.Errorf("name = %q, want spec-overwrite-42", got.Name)
	}
}

func TestWatchInbox_ScansExistingFilesOnStartup(t *testing.T) {
	dir := t.TempDir()
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	// Create a valid d-mail BEFORE watcher starts
	content := "---\nname: spec-pre-existing\nkind: specification\ndescription: pre-existing file\n---\n\nBody\n"
	os.WriteFile(filepath.Join(inboxDir, "spec-pre-existing.md"), []byte(content), 0644)

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

	// No file writes after watcher starts — must detect pre-existing file
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout — pre-existing d-mail was not detected on startup")
	}

	mu.Lock()
	defer mu.Unlock()
	if got.Name != "spec-pre-existing" {
		t.Errorf("name = %q, want spec-pre-existing", got.Name)
	}
}

func TestWatchInbox_IgnoresInvalidDMailFile(t *testing.T) {
	dir := t.TempDir()
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	var callCount int
	var mu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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

	// Write .md files with invalid frontmatter — should be silently skipped
	os.WriteFile(filepath.Join(inboxDir, "bad1.md"), []byte("no frontmatter"), 0644)
	os.WriteFile(filepath.Join(inboxDir, "bad2.md"), []byte("---\ninvalid: [yaml\n---\n"), 0644)

	// Give watcher time to process events
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 0 {
		t.Errorf("callback fired %d times for invalid d-mail files, want 0", callCount)
	}
}

func TestWatchInbox_MultipleFilesInSequence(t *testing.T) {
	dir := t.TempDir()
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	var mu sync.Mutex
	var names []string

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ready := make(chan struct{}, 1)
	go watchInbox(ctx, dir, func(dm DMail) {
		mu.Lock()
		names = append(names, dm.Name)
		mu.Unlock()
	}, ready)

	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("timeout waiting for watcher ready")
	}

	// Write 3 files in quick succession
	for _, name := range []string{"first", "second", "third"} {
		content := "---\nname: " + name + "\nkind: report\ndescription: " + name + "\n---\n"
		os.WriteFile(filepath.Join(inboxDir, name+".md"), []byte(content), 0644)
	}

	// Wait for all callbacks to fire
	deadline := time.After(4 * time.Second)
	for {
		mu.Lock()
		count := len(names)
		mu.Unlock()
		if count >= 3 {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("timeout: got %d callbacks (%v), want 3", len(names), names)
			mu.Unlock()
			return
		case <-time.After(100 * time.Millisecond):
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(names) != 3 {
		t.Errorf("got %d callbacks, want 3: %v", len(names), names)
	}
}

func TestWatchInbox_InitialScanSkipsInvalidFiles(t *testing.T) {
	dir := t.TempDir()
	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	os.MkdirAll(inboxDir, 0755)

	// Pre-create one valid and one invalid file before watcher starts
	validContent := "---\nname: valid-pre\nkind: report\ndescription: valid\n---\n"
	os.WriteFile(filepath.Join(inboxDir, "valid-pre.md"), []byte(validContent), 0644)
	os.WriteFile(filepath.Join(inboxDir, "invalid-pre.md"), []byte("not a dmail"), 0644)

	var mu sync.Mutex
	var names []string
	done := make(chan struct{}, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ready := make(chan struct{}, 1)
	go watchInbox(ctx, dir, func(dm DMail) {
		mu.Lock()
		names = append(names, dm.Name)
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

	// Should get callback for valid file only
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout — valid pre-existing file was not detected")
	}

	// Wait a bit to ensure invalid file didn't trigger callback
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(names) != 1 {
		t.Fatalf("got %d callbacks, want 1 (only valid file): %v", len(names), names)
	}
	if names[0] != "valid-pre" {
		t.Errorf("name = %q, want valid-pre", names[0])
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
