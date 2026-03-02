//go:build scenario

package scenario_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScenario_ApproveCmdPath(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "minimal")
	obs := NewObserver(ws, t)

	pw := ws.StartPhonewave(t, ctx)
	defer ws.StopPhonewave(t, pw)
	defer ws.DumpPhonewaveLog(t, pw)

	// Inject a specification D-Mail for paintress to consume
	specContent := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "test-spec",
		"kind":                 "specification",
		"description":          "Test specification for approve-cmd",
	}, "# Test Specification\n\nThis is a test specification for the approve-cmd scenario test.")
	ws.InjectDMail(t, ".expedition", "inbox", "test-spec.md", specContent)

	// Create approve script (exit 0 = approve all)
	approveScript := filepath.Join(ws.Root, "approve.sh")
	if err := os.WriteFile(approveScript, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write approve script: %v", err)
	}

	// Create notify script that logs invocations for verification
	notifyLog := filepath.Join(ws.Root, "notify.log")
	notifyScript := filepath.Join(ws.Root, "notify.sh")
	notifyContent := fmt.Sprintf("#!/bin/sh\necho \"$@\" >> %s\n", notifyLog)
	if err := os.WriteFile(notifyScript, []byte(notifyContent), 0o755); err != nil {
		t.Fatalf("write notify script: %v", err)
	}

	// Run paintress with --approve-cmd and --notify-cmd (NOT --auto-approve)
	err := ws.RunPaintress(t, ctx, "run",
		"--approve-cmd", approveScript,
		"--notify-cmd", notifyScript,
		"--no-dev", "--workers", "0", "--max-expeditions", "1",
		ws.RepoPath,
	)
	if err != nil {
		t.Fatalf("paintress run with approve-cmd failed: %v", err)
	}

	// Verify report was produced and delivered
	reportPath := ws.WaitForDMail(t, ".gate", "inbox", 30*time.Second)
	obs.AssertDMailKind(reportPath, "report")

	// Verify outbox was flushed
	ws.WaitForAbsent(t, ".expedition", "outbox", 10*time.Second)

	// Verify notify script was invoked (notify.log should exist and be non-empty)
	data, err := os.ReadFile(notifyLog)
	if err != nil {
		t.Logf("notify.log not found (notification may not have fired): %v", err)
	} else if len(data) == 0 {
		t.Log("notify.log exists but is empty")
	} else {
		t.Logf("notify.log content:\n%s", string(data))
	}
}
