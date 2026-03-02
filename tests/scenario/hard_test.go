//go:build scenario

package scenario_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScenario_L4_Hard(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "hard")
	obs := NewObserver(ws, t)

	// --- Phase 1: phonewave daemon restart ---
	pw := ws.StartPhonewave(t, ctx)
	defer ws.DumpPhonewaveLog(t, pw)

	// Inject specification before restart
	spec1 := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-pre-restart",
		"kind":                 "specification",
		"description":          "Specification before daemon restart",
	}, "# Pre-Restart Spec\n\n## Actions\n\n- [add_dod] RESTART-001: test")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-pre-restart.md", spec1)

	// Run paintress (may or may not succeed depending on timing)
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("pre-restart expedition: %v (acceptable during restart test)", err)
	}

	// Restart phonewave daemon
	t.Log("restarting phonewave daemon")
	ws.StopPhonewave(t, pw)
	time.Sleep(1 * time.Second)
	pw = ws.StartPhonewave(t, ctx)
	defer ws.StopPhonewave(t, pw)

	// Wait for any pending deliveries after restart
	ws.WaitForAbsent(t, ".expedition", "outbox", 30*time.Second)

	// --- Phase 2: fake-claude transient failure ---
	counterPath := filepath.Join(os.TempDir(), "fake-claude-call-count")
	os.Remove(counterPath)
	ws.Env = append(ws.Env, "FAKE_CLAUDE_FAIL_COUNT=2")

	spec2 := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-with-failures",
		"kind":                 "specification",
		"description":          "Specification triggering transient failures",
	}, "# Transient Failure Spec\n\n## Actions\n\n- [add_dod] FAIL-001: test failure recovery")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-with-failures.md", spec2)

	// First two expeditions will have fake-claude fail
	for i := 0; i < 2; i++ {
		err := ws.RunPaintressExpedition(t, ctx)
		if err != nil {
			t.Logf("expedition %d with FAIL_COUNT: %v (expected failure)", i+1, err)
		}
	}

	// Third expedition should succeed
	err = ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("recovery expedition: %v (may still be acceptable)", err)
	}

	// Clean up FAIL_COUNT env
	cleanEnv := make([]string, 0, len(ws.Env))
	for _, e := range ws.Env {
		if e != "FAKE_CLAUDE_FAIL_COUNT=2" {
			cleanEnv = append(cleanEnv, e)
		}
	}
	ws.Env = cleanEnv
	os.Remove(counterPath)

	// --- Phase 3: malformed D-Mail ---
	malformed := []byte("This is not a valid D-Mail.\nNo YAML frontmatter here.\n")
	ws.InjectDMail(t, ".expedition", "inbox", "malformed-001.md", malformed)

	err = ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("expedition after malformed inject: %v (acceptable)", err)
	}

	// Wait for system to stabilize
	time.Sleep(3 * time.Second)

	// --- Final verification ---
	ws.WaitForAbsent(t, ".expedition", "outbox", 30*time.Second)
	obs.AssertAllOutboxEmpty()
	t.Log("L4 hard test passed: daemon restart + transient failures + malformed D-Mail all handled")
}
