//go:build contract

package contract_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/policy"
	"github.com/hironow/paintress/internal/harness/verifier"
)

const goldenDir = "testdata/golden"

func goldenFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("read golden dir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		t.Fatal("no golden files found")
	}
	return files
}

func readGolden(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(goldenDir, name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return data
}

// TestContract_ParseDMail verifies that paintress's ParseDMail can
// parse all cross-tool golden files.
func TestContract_ParseDMail(t *testing.T) {
	for _, name := range goldenFiles(t) {
		t.Run(name, func(t *testing.T) {
			data := readGolden(t, name)
			dm, err := domain.ParseDMail(data)
			if err != nil {
				t.Fatalf("ParseDMail error: %v", err)
			}
			if dm.Name == "" {
				t.Error("parsed name is empty")
			}
			if dm.Kind == "" {
				t.Error("parsed kind is empty")
			}
			if dm.Description == "" {
				t.Error("parsed description is empty")
			}
			if dm.SchemaVersion == "" {
				t.Error("parsed schema version is empty")
			}
		})
	}
}

// TestContract_ValidateDMailRejectsEdgeCases verifies send-side strict
// validation rejects unknown kinds and future schemas.
func TestContract_ValidateDMailRejectsEdgeCases(t *testing.T) {
	data := readGolden(t, "future-schema.md")
	dm, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	if err := verifier.ValidateDMail(dm); err == nil {
		t.Error("expected ValidateDMail to fail for schema version '2', but it passed")
	}

	data = readGolden(t, "unknown-kind.md")
	dm, err = domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	if err := verifier.ValidateDMail(dm); err == nil {
		t.Error("expected ValidateDMail to reject unknown kind, but it passed")
	}
}

// --- Send-side contract tests ---

func TestContract_NewReportDMail_ValidatesSuccessfully(t *testing.T) {
	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "AUTH-42",
		IssueTitle:  "Implement 2FA",
		MissionType: "implement",
		Status:      "success",
		PRUrl:       "https://github.com/test/repo/pull/123",
		Reason:      "All tests pass, 2FA implemented",
	}
	dmail := policy.NewReportDMail(report, 0)
	if err := verifier.ValidateDMail(dmail); err != nil {
		t.Fatalf("NewReportDMail produced invalid D-Mail: %v", err)
	}
	if dmail.Kind != "report" {
		t.Errorf("kind: got %q, want %q", dmail.Kind, "report")
	}
	data, err := dmail.Marshal()
	if err != nil {
		t.Fatalf("MarshalDMail: %v", err)
	}
	if !strings.Contains(string(data), "https://github.com/test/repo/pull/123") {
		t.Error("marshaled D-Mail should contain PR URL")
	}
}

func TestContract_NewReportDMail_OmitsPRWhenNone(t *testing.T) {
	cases := []struct {
		name  string
		prUrl string
	}{
		{"empty", ""},
		{"none", "none"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := &domain.ExpeditionReport{
				Expedition:  1,
				IssueID:     "BUG-1",
				IssueTitle:  "Fix crash",
				MissionType: "fix",
				Status:      "failed",
				PRUrl:       tc.prUrl,
			}
			dmail := policy.NewReportDMail(report, 0)
			if err := verifier.ValidateDMail(dmail); err != nil {
				t.Fatalf("validation failed: %v", err)
			}
			data, _ := dmail.Marshal()
			if strings.Contains(string(data), "**PR:**") {
				t.Error("PR line should be omitted when URL is empty/none")
			}
		})
	}
}

func TestContract_NewReportDMail_FailedStatus(t *testing.T) {
	report := &domain.ExpeditionReport{
		Expedition:  3,
		IssueID:     "PERF-7",
		IssueTitle:  "Optimize query",
		MissionType: "optimize",
		Status:      "failed",
		Reason:      "OOM during benchmark",
	}
	dmail := policy.NewReportDMail(report, 0)
	if err := verifier.ValidateDMail(dmail); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	data, _ := dmail.Marshal()
	if !strings.Contains(string(data), "OOM during benchmark") {
		t.Error("body should contain failure reason")
	}
}

// TestContract_CorrectiveMetadataRoundTrip verifies corrective-feedback.md
// golden file parses correctly and CorrectionMetadataFromMap extracts all fields.
func TestContract_CorrectiveMetadataRoundTrip(t *testing.T) {
	data := readGolden(t, "corrective-feedback.md")
	dm, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	meta := domain.CorrectionMetadataFromMap(dm.Metadata)
	if !meta.IsImprovement() {
		t.Fatal("expected IsImprovement() = true for corrective-feedback.md")
	}
	checks := map[string]string{
		"routing_mode":   string(meta.RoutingMode),
		"target_agent":   meta.TargetAgent,
		"provider_state": string(meta.ProviderState),
		"correlation_id": meta.CorrelationID,
		"trace_id":       meta.TraceID,
		"failure_type":   string(meta.FailureType),
	}
	expected := map[string]string{
		"routing_mode":   "escalate",
		"target_agent":   "sightjack",
		"provider_state": "active",
		"correlation_id": "corr-abc-123",
		"trace_id":       "trace-xyz-789",
		"failure_type":   "scope_violation",
	}
	for key, want := range expected {
		got := checks[key]
		if got != want {
			t.Errorf("metadata[%q] = %q, want %q", key, got, want)
		}
	}
}
