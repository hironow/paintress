//go:build scenario

package scenario_test

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestScenario_L2_Small(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "small")
	obs := NewObserver(ws, t)

	pw := ws.StartPhonewave(t, ctx)
	defer ws.StopPhonewave(t, pw)
	defer ws.DumpPhonewaveLog(t, pw)

	// 1. Sightjack scan → specification delivered to .expedition/inbox via phonewave
	err := ws.RunSightjackScan(t, ctx)
	if err != nil {
		t.Fatalf("sightjack scan failed: %v", err)
	}

	// Wait for specification(s) routed from .siren/outbox → phonewave → .expedition/inbox
	ws.WaitForDMailCount(t, ".expedition", "inbox", 1, 30*time.Second)

	// Verify siren outbox is cleaned up (phonewave consumed the specs)
	ws.WaitForAbsent(t, ".siren", "outbox", 15*time.Second)

	// 2. Paintress processes specification(s) → report in .expedition/outbox → phonewave → .gate/inbox
	err = ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("first paintress expedition: %v", err)
	}

	// Wait for report delivery to .gate/inbox
	ws.WaitForDMailCount(t, ".gate", "inbox", 1, 30*time.Second)

	// Verify outbox cleanup
	ws.WaitForAbsent(t, ".expedition", "outbox", 15*time.Second)

	// 3. Amadeus processes report → feedback in .gate/outbox → phonewave → .expedition/inbox + .siren/inbox
	err = ws.RunAmadeusCheck(t, ctx)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			t.Logf("amadeus check returned exit code 2 (drift detected) — expected")
		} else {
			t.Fatalf("amadeus check failed: %v", err)
		}
	}

	// Wait for feedback routed to .expedition/inbox
	ws.WaitForDMailCount(t, ".expedition", "inbox", 1, 30*time.Second)

	// Wait for feedback routed to .siren/inbox
	ws.WaitForDMail(t, ".siren", "inbox", 30*time.Second)

	// Verify .gate outbox cleaned up
	ws.WaitForAbsent(t, ".gate", "outbox", 15*time.Second)

	// 4. Second expedition (processes feedback from amadeus)
	err = ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("second paintress expedition: %v", err)
	}

	// Verify final state: all outboxes empty
	obs.AssertAllOutboxEmpty()
}
