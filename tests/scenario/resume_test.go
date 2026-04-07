//go:build scenario

package scenario_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestScenario_ResumeAfterCrash_Workers0 verifies that workers=0 resumes
// an interrupted expedition after a crash (simulated by kill):
//
//  1. Start paintress with --idle-timeout=-1s (one-shot)
//  2. Wait for subprocess_started checkpoint to be emitted
//  3. Kill the process (simulates crash)
//  4. Restart paintress — it should detect the incomplete expedition and resume
//
// The resume is verified by the log message "resuming incomplete expedition".
func TestScenario_ResumeAfterCrash_Workers0(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	ws := NewWorkspace(t, "minimal")

	// Inject specification D-Mail
	spec := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-resume-001",
		"kind":                 "specification",
		"description":          "Resume test specification",
	}, "# Resume Test\n\nExpedition to be interrupted and resumed.")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-resume-001.md", spec)

	// Phase 1: Start paintress and wait for checkpoint event to appear in JSONL.
	// Using async start so we can kill it after checkpoint is written.
	cmd, output := ws.StartPaintressAsync(t, ctx, "-1s")

	// Wait for the subprocess to start (checkpoint event should be in event store)
	eventsDir := filepath.Join(ws.RepoPath, ".expedition", "events")
	waitForCheckpointEvent(t, eventsDir, "subprocess_started", 60*time.Second)
	t.Log("Phase 1: subprocess_started checkpoint detected")

	// Phase 2: Kill the process (simulate crash)
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	t.Log("Phase 2: process killed (simulating crash)")

	// Phase 3: Restart paintress — should detect and resume
	// Use sync one-shot since we just want to see the resume log
	restartCtx, restartCancel := context.WithTimeout(ctx, 60*time.Second)
	defer restartCancel()
	_, restartOutput := ws.StartPaintressAsync(t, restartCtx, "-1s")

	// Wait for resume log
	waitForLog(t, restartOutput, "resuming incomplete expedition", 30*time.Second)
	t.Log("Phase 3: resume detected on restart")
}

// TestScenario_FreshStart_WorkersN verifies that workers>0 does NOT resume
// interrupted expeditions (swarm mode = fresh start).
func TestScenario_FreshStart_WorkersN(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	ws := NewWorkspace(t, "minimal")

	// Inject specification D-Mail
	spec := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-swarm-001",
		"kind":                 "specification",
		"description":          "Swarm fresh start test",
	}, "# Swarm Test\n\nExpedition for swarm mode fresh start verification.")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-swarm-001.md", spec)

	// Phase 1: Start paintress with workers=0 (to generate checkpoint),
	// then kill to create incomplete state.
	cmd, _ := ws.StartPaintressAsync(t, ctx, "-1s")

	eventsDir := filepath.Join(ws.RepoPath, ".expedition", "events")
	waitForCheckpointEvent(t, eventsDir, "subprocess_started", 60*time.Second)
	t.Log("Phase 1: checkpoint created")

	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	t.Log("Phase 2: process killed")

	// Phase 3: Restart with --workers=2 (swarm mode)
	restartCtx, restartCancel := context.WithTimeout(ctx, 60*time.Second)
	defer restartCancel()
	_, restartOutput := ws.StartPaintressAsync(t, restartCtx, "-1s", "--workers", "2")

	// Wait for the swarm-mode skip log
	waitForLog(t, restartOutput, "skipping resume in swarm mode", 30*time.Second)
	t.Log("Phase 3: swarm mode correctly skips resume")

	// Verify no "resuming incomplete" log appears
	if strings.Contains(restartOutput.String(), "resuming incomplete expedition") {
		t.Error("swarm mode should NOT resume incomplete expeditions")
	}
}

// waitForCheckpointEvent polls the events directory for a JSONL file containing
// the given checkpoint phase.
func waitForCheckpointEvent(t *testing.T, eventsDir, phase string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(eventsDir)
		if err == nil {
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".jsonl") {
					data, _ := os.ReadFile(filepath.Join(eventsDir, e.Name()))
					if strings.Contains(string(data), phase) {
						return
					}
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for checkpoint event %q in %s", phase, eventsDir)
}
