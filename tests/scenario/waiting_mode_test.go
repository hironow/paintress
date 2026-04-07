//go:build scenario

package scenario_test

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestScenario_WaitingModeExpeditionRescan verifies that paintress's waiting
// mode correctly handles D-Mail arrival after the initial expedition:
//
//  1. Initial expedition completes and session enters waiting phase
//  2. Specification D-Mail injected into .expedition/inbox triggers re-run
//  3. The default waiting path (--idle-timeout > 0) is exercised
//
// This test uses --idle-timeout 45s (positive) instead of the usual -1s,
// which exercises the production default path that other scenario tests skip.
func TestScenario_WaitingModeExpeditionRescan(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	ws := NewWorkspace(t, "minimal")

	// Inject initial specification D-Mail before starting paintress.
	spec1 := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-wait-init",
		"kind":                 "specification",
		"description":          "Initial spec for waiting mode test",
	}, "# Initial Spec\n\nFirst expedition target.")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-wait-init.md", spec1)

	// Start paintress in background with waiting mode enabled.
	_, output := ws.StartPaintressAsync(t, ctx, "45s")

	// Phase 1: Wait for initial expedition to complete and session to enter waiting.
	waitForLog(t, output, "Waiting for", 60*time.Second)
	t.Log("Phase 1: waiting mode entered after initial expedition")

	// Phase 2: Inject second specification D-Mail to trigger re-run.
	spec2 := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-wait-rerun",
		"kind":                 "specification",
		"description":          "Second spec to trigger re-run in waiting mode",
	}, "# Re-run Spec\n\nSecond expedition target via waiting mode.")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-wait-rerun.md", spec2)
	t.Log("Phase 2: specification D-Mail injected")

	// Phase 3: Verify re-run was triggered.
	waitForLog(t, output, "D-Mail received", 30*time.Second)
	t.Log("Phase 3: D-Mail received, re-run triggered")

	// Phase 4: Verify no duplicate re-run from same D-Mail.
	time.Sleep(3 * time.Second)
	rerunCount := strings.Count(output.String(), "re-running expedition cycle")
	if rerunCount != 1 {
		t.Errorf("expected exactly 1 re-run, got %d", rerunCount)
	}
	t.Log("Phase 4: no duplicate re-run")
}

// waitForLog polls the output buffer for a substring, with timeout.
func waitForLog(t *testing.T, buf interface{ String() string }, substr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), substr) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for log %q in output:\n%s", substr, buf.String())
}
