package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPaintressRun_DryRun_FirstRun_StartsAtExpedition1(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	cfg := Config{
		Continent:      dir,
		MaxExpeditions: 5,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "echo ok",
		DevURL:         "http://localhost:3000",
		DryRun:         true,
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}

	// Dry-run should create expedition-001-prompt.md (starts at 1)
	promptFile := filepath.Join(p.logDir, "expedition-001-prompt.md")
	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("prompt file not created: %v", err)
	}

	if !containsStr(string(content), "Expedition #1") {
		t.Error("prompt should contain 'Expedition #1'")
	}
}

func TestPaintressRun_DryRun_ResumeFromFlag(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	// Plant a flag indicating expedition 7 was the last
	WriteFlag(dir, 7, "AWE-50", "success", "3")

	cfg := Config{
		Continent:      dir,
		MaxExpeditions: 5,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "echo ok",
		DevURL:         "http://localhost:3000",
		DryRun:         true,
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("Run() = %d, want 0", code)
	}

	// Should resume at expedition 8, not 1
	promptFile := filepath.Join(p.logDir, "expedition-008-prompt.md")
	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("prompt file expedition-008-prompt.md not created: %v", err)
	}

	if !containsStr(string(content), "Expedition #8") {
		t.Error("prompt should contain 'Expedition #8' (resumed from flag)")
	}

	// expedition-001-prompt.md should NOT exist
	oldPrompt := filepath.Join(p.logDir, "expedition-001-prompt.md")
	if _, err := os.Stat(oldPrompt); !os.IsNotExist(err) {
		t.Error("expedition-001-prompt.md should not exist on resume")
	}
}

func TestPaintressRun_DryRun_PreservesExistingJournals(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// Simulate 3 previous expeditions with journals
	for i := 1; i <= 3; i++ {
		WriteJournal(dir, &ExpeditionReport{
			Expedition: i, IssueID: "AWE-" + string(rune('0'+i)),
			IssueTitle: "past", MissionType: "implement",
			Status: "success", Reason: "done", PRUrl: "none", BugIssues: "none",
		})
	}
	WriteFlag(dir, 3, "AWE-3", "success", "5")

	// Capture original content of journal 001
	original001, err := os.ReadFile(filepath.Join(jDir, "001.md"))
	if err != nil {
		t.Fatalf("pre-existing journal 001.md missing: %v", err)
	}

	cfg := Config{
		Continent:      dir,
		MaxExpeditions: 5,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "echo ok",
		DevURL:         "http://localhost:3000",
		DryRun:         true,
	}

	p := NewPaintress(cfg)
	p.Run(context.Background())

	// Verify original journals are untouched
	after001, err := os.ReadFile(filepath.Join(jDir, "001.md"))
	if err != nil {
		t.Fatal("journal 001.md was deleted")
	}
	if string(original001) != string(after001) {
		t.Error("journal 001.md was overwritten")
	}
}

func TestReadFlag_ResumeExpeditionNumber(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(dir string)
		wantLastExp   int
		wantStartExp  int
		wantRemaining string
	}{
		{
			name:          "no flag file — fresh start",
			setup:         func(dir string) {},
			wantLastExp:   0,
			wantStartExp:  1,
			wantRemaining: "?",
		},
		{
			name: "flag at expedition 5",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)
				WriteFlag(dir, 5, "AWE-10", "success", "8")
			},
			wantLastExp:   5,
			wantStartExp:  6,
			wantRemaining: "8",
		},
		{
			name: "flag at expedition 20",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)
				WriteFlag(dir, 20, "AWE-99", "failed", "2")
			},
			wantLastExp:   20,
			wantStartExp:  21,
			wantRemaining: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			flag := ReadFlag(dir)
			startExp := flag.LastExpedition + 1

			if flag.LastExpedition != tt.wantLastExp {
				t.Errorf("LastExpedition = %d, want %d", flag.LastExpedition, tt.wantLastExp)
			}
			if startExp != tt.wantStartExp {
				t.Errorf("startExp = %d, want %d", startExp, tt.wantStartExp)
			}
			if flag.Remaining != tt.wantRemaining {
				t.Errorf("Remaining = %q, want %q", flag.Remaining, tt.wantRemaining)
			}
		})
	}
}

func TestWriteJournal_ResumedNumbering(t *testing.T) {
	dir := t.TempDir()

	// Write journal at expedition 8 (simulating a resumed run)
	report := &ExpeditionReport{
		Expedition:  8,
		IssueID:     "AWE-50",
		IssueTitle:  "Fix login",
		MissionType: "fix",
		Status:      "success",
		Reason:      "done",
		PRUrl:       "https://github.com/org/repo/pull/8",
		BugIssues:   "none",
	}
	if err := WriteJournal(dir, report); err != nil {
		t.Fatal(err)
	}

	// Should create 008.md, not 001.md
	path008 := filepath.Join(dir, ".expedition", "journal", "008.md")
	if _, err := os.Stat(path008); os.IsNotExist(err) {
		t.Fatal("expected 008.md to be created")
	}

	path001 := filepath.Join(dir, ".expedition", "journal", "001.md")
	if _, err := os.Stat(path001); !os.IsNotExist(err) {
		t.Error("001.md should not exist — journal should use resumed number")
	}

	content, err := os.ReadFile(path008)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(content), "Expedition #8") {
		t.Error("journal should reference expedition #8")
	}
}

// === Sentinel Errors ===

func TestSentinelErrors_AreDistinct(t *testing.T) {
	if errors.Is(errGommage, errComplete) {
		t.Error("errGommage and errComplete must be distinct errors")
	}
	if errGommage.Error() == "" {
		t.Error("errGommage must have a non-empty message")
	}
	if errComplete.Error() == "" {
		t.Error("errComplete must have a non-empty message")
	}
}

// === Swarm Mode DryRun Integration Tests ===

// setupTestRepo creates a minimal git repo for Paintress tests.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "test"},
		{"git", "config", "commit.gpgsign", "false"},
		{"git", "checkout", "-b", "main"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup (%v) failed: %s", args, string(out))
		}
	}
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	return dir
}

func TestSwarmMode_DryRun_CreatesUniquePrompts(t *testing.T) {
	dir := setupTestRepo(t)

	cfg := Config{
		Continent:      dir,
		Workers:        3,
		MaxExpeditions: 3,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	logDir := filepath.Join(dir, ".expedition", ".logs")
	prompts, err := filepath.Glob(filepath.Join(logDir, "expedition-*-prompt.md"))
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(prompts) != 3 {
		t.Errorf("expected 3 prompt files, got %d: %v", len(prompts), prompts)
	}

	// Verify all expedition numbers are unique
	seen := make(map[string]bool)
	for _, p := range prompts {
		base := filepath.Base(p)
		if seen[base] {
			t.Errorf("duplicate prompt file: %s", base)
		}
		seen[base] = true
	}
}

func TestSwarmMode_DryRun_SingleWorker(t *testing.T) {
	dir := setupTestRepo(t)

	cfg := Config{
		Continent:      dir,
		Workers:        0,
		MaxExpeditions: 1,
		DryRun:         true,
		BaseBranch:     "main",
		TimeoutSec:     30,
		Model:          "opus",
	}

	p := NewPaintress(cfg)
	code := p.Run(context.Background())

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	logDir := filepath.Join(dir, ".expedition", ".logs")
	prompts, _ := filepath.Glob(filepath.Join(logDir, "expedition-*-prompt.md"))
	if len(prompts) != 1 {
		t.Errorf("expected 1 prompt file, got %d", len(prompts))
	}
}
