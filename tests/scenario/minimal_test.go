//go:build scenario

package scenario_test

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// TestScenario_L1_Minimal verifies the full closed loop: spec -> report -> feedback.
//
//  1. specification D-Mail is injected into .expedition/inbox (upstream from sightjack via phonewave)
//  2. paintress expedition processes the specification, produces report in .expedition/outbox
//  3. phonewave routes the report to .gate/inbox, cleans .expedition/outbox
//  4. amadeus check consumes report from .gate/inbox, produces feedback in .gate/outbox
//  5. phonewave routes feedback to .expedition/inbox + .siren/inbox
//  6. all outboxes empty at the end
//
// Route exercised:
//   .expedition/inbox -> paintress -> .expedition/outbox -> phonewave -> .gate/inbox
//   .gate/inbox -> amadeus -> .gate/outbox -> phonewave -> .expedition/inbox + .siren/inbox
func TestScenario_L1_Minimal(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "minimal")
	obs := NewObserver(ws, t)

	// Start phonewave daemon (routes .expedition/outbox -> .gate/inbox)
	pw := ws.StartPhonewave(t, ctx)
	defer ws.StopPhonewave(t, pw)
	defer ws.DumpPhonewaveLog(t, pw)

	// Inject specification D-Mail into .expedition/inbox (upstream input from sightjack)
	spec := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-test-001",
		"kind":                 "specification",
		"description":          "Test specification",
	}, "# Test Spec\n\n## Actions\n\n- [add_dod] TEST-001: Test action")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-test-001.md", spec)

	// Run paintress expedition (consumes specification, produces report)
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Fatalf("paintress expedition failed: %v", err)
	}

	// Wait for report D-Mail delivery: .expedition/outbox -> phonewave -> .gate/inbox
	reportPath := ws.WaitForDMail(t, ".gate", "inbox", 30*time.Second)

	// Verify outbox is cleaned up (phonewave consumed the report from .expedition/outbox)
	ws.WaitForAbsent(t, ".expedition", "outbox", 10*time.Second)

	// Verify report kind in frontmatter
	obs.AssertDMailKind(reportPath, "report")

	// 3. Run amadeus → feedback in .gate/outbox → phonewave → .expedition/inbox + .siren/inbox
	err = ws.RunAmadeusCheck(t, ctx)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			t.Logf("amadeus exit code 2 (drift) — expected")
		} else {
			t.Fatalf("amadeus check failed: %v", err)
		}
	}

	// 4. Verify feedback arrived in .expedition/inbox and .siren/inbox
	// .expedition/inbox: feedback only (spec was consumed/archived by paintress)
	// .siren/inbox: feedback delivered by phonewave
	// .gate/inbox: empty (amadeus consumed the report)
	feedbackPath := ws.WaitForDMail(t, ".expedition", "inbox", 30*time.Second)
	obs.AssertDMailKind(feedbackPath, "feedback")
	ws.WaitForDMail(t, ".siren", "inbox", 30*time.Second)

	// Final state: all outboxes empty
	obs.AssertAllOutboxEmpty()
}
