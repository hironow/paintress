//go:build scenario

package scenario_test

import (
	"context"
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

	// Inject 2 specification D-Mails with different priorities
	spec1 := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-high-001",
		"kind":                 "specification",
		"description":          "High priority specification",
		"priority":             "1",
	}, "# High Priority Spec\n\n## Actions\n\n- [add_dod] AUTH-001: critical auth fix")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-high-001.md", spec1)

	spec2 := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-low-002",
		"kind":                 "specification",
		"description":          "Low priority specification",
		"priority":             "3",
	}, "# Low Priority Spec\n\n## Actions\n\n- [add_dod] UI-002: minor UI tweak")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-low-002.md", spec2)

	// Run paintress expedition -- processes specs
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("first paintress expedition: %v", err)
	}

	// Wait for report delivery to .gate/inbox
	ws.WaitForDMailCount(t, ".gate", "inbox", 1, 30*time.Second)

	// Verify outbox cleanup
	ws.WaitForAbsent(t, ".expedition", "outbox", 15*time.Second)

	// Inject feedback D-Mail (simulates amadeus retry response)
	feedback := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "feedback-retry-001",
		"kind":                 "feedback",
		"description":          "Retry feedback",
	}, "# Feedback\n\n## Action: retry\n\nPlease retry the specification.")
	ws.InjectDMail(t, ".expedition", "inbox", "feedback-retry-001.md", feedback)

	// Second expedition (follow-up from feedback)
	err = ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("second paintress expedition: %v", err)
	}

	// Verify final state
	obs.AssertAllOutboxEmpty()
}
