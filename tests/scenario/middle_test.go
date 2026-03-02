//go:build scenario

package scenario_test

import (
	"context"
	"testing"
	"time"
)

func TestScenario_L3_Middle(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "middle")
	obs := NewObserver(ws, t)

	pw := ws.StartPhonewave(t, ctx)
	defer ws.StopPhonewave(t, pw)
	defer ws.DumpPhonewaveLog(t, pw)

	// Inject 3 specification D-Mails
	for i, spec := range []struct {
		name     string
		priority string
		desc     string
		body     string
	}{
		{"spec-critical-001", "1", "Critical specification", "# Critical\n\n## Actions\n\n- [add_dod] AUTH-001: critical fix"},
		{"spec-medium-002", "2", "Medium specification", "# Medium\n\n## Actions\n\n- [add_dod] DATA-002: data layer fix"},
		{"spec-low-003", "3", "Low specification", "# Low\n\n## Actions\n\n- [add_dod] UI-003: minor styling"},
	} {
		dmail := FormatDMail(map[string]string{
			"dmail-schema-version": "1",
			"name":                 spec.name,
			"kind":                 "specification",
			"description":          spec.desc,
			"priority":             spec.priority,
		}, spec.body)
		ws.InjectDMail(t, ".expedition", "inbox", spec.name+".md", dmail)
		t.Logf("injected spec %d: %s (priority %s)", i+1, spec.name, spec.priority)
	}

	// First paintress expedition
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("first expedition: %v", err)
	}

	// Wait for report delivery
	ws.WaitForDMailCount(t, ".gate", "inbox", 1, 30*time.Second)
	ws.WaitForAbsent(t, ".expedition", "outbox", 15*time.Second)

	// Inject convergence D-Mail
	convergence := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "convergence-001",
		"kind":                 "convergence",
		"description":          "System convergence checkpoint",
	}, "# Convergence\n\nAll tools have stabilized.")
	ws.InjectDMail(t, ".expedition", "inbox", "convergence-001.md", convergence)

	// Second expedition
	err = ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("second expedition: %v", err)
	}

	// Verify no deadlock, all outboxes eventually empty
	ws.WaitForAbsent(t, ".expedition", "outbox", 15*time.Second)
	obs.AssertAllOutboxEmpty()
}
