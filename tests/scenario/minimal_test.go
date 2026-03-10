//go:build scenario

package scenario_test

import (
	"context"
	"testing"
	"time"
)

// TestScenario_L1_Minimal verifies the full closed loop: spec -> report -> feedback.
//
//  1. specification D-Mail is injected into .expedition/inbox (upstream from sightjack via phonewave)
//  2. paintress expedition processes the specification, produces report in .expedition/outbox
//  3. phonewave routes the report to .gate/inbox, cleans .expedition/outbox
//  4. amadeus run consumes report from .gate/inbox, produces feedback in .gate/outbox
//  5. phonewave routes feedback to .expedition/inbox + .siren/inbox
//  6. all outboxes empty at the end
//
// Route exercised:
//
//	.expedition/inbox -> paintress -> .expedition/outbox -> phonewave -> .gate/inbox
//	.gate/inbox -> amadeus -> .gate/outbox -> phonewave -> .expedition/inbox + .siren/inbox
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

	// 3. Start amadeus run as daemon → feedback in .gate/outbox → phonewave → .expedition/inbox
	am := ws.StartAmadeusRun(t, ctx)
	defer ws.StopAmadeusRun(t, am)

	// 4. Verify feedback arrived in .expedition/inbox
	feedbackPath := ws.WaitForDMail(t, ".expedition", "inbox", 30*time.Second)
	obs.AssertDMailKind(feedbackPath, "implementation-feedback")

	// Final state: all outboxes empty
	obs.AssertAllOutboxEmpty()
}
