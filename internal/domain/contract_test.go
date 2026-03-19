//go:build contract

package domain_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

const contractGoldenDir = "testdata/contract"

func contractGoldenFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(contractGoldenDir)
	if err != nil {
		t.Fatalf("read contract golden dir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		t.Fatal("no contract golden files found")
	}
	return files
}

func readContractGolden(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(contractGoldenDir, name))
	if err != nil {
		t.Fatalf("read contract golden %s: %v", name, err)
	}
	return data
}

// TestContract_ParseDMail verifies that paintress's ParseDMail can
// parse all cross-tool golden files. Paintress is Postel-liberal at
// the parse level — unknown kinds and future schemas parse without error.
func TestContract_ParseDMail(t *testing.T) {
	for _, name := range contractGoldenFiles(t) {
		t.Run(name, func(t *testing.T) {
			data := readContractGolden(t, name)
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

// TestContract_ValidateDMailRejectsEdgeCases verifies that paintress's
// ValidateDMail rejects D-Mails with unsupported schema version.
// NOTE: paintress ValidateDMail does NOT validate kind enum (unlike
// phonewave/sightjack). This is a known divergence — kind validation
// is deferred to the consuming tool's business logic.
func TestContract_ValidateDMailRejectsEdgeCases(t *testing.T) {
	// future-schema.md has dmail-schema-version "2" — should be rejected
	data := readContractGolden(t, "future-schema.md")
	dm, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	if err := domain.ValidateDMail(dm); err == nil {
		t.Error("expected ValidateDMail to fail for schema version '2', but it passed")
	}

	// unknown-kind.md parses and validates (paintress doesn't enforce kind enum)
	data = readContractGolden(t, "unknown-kind.md")
	dm, err = domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	if err := domain.ValidateDMail(dm); err != nil {
		t.Errorf("unexpected ValidateDMail error for unknown kind: %v", err)
	}
}

// --- Send-side contract tests (proposal 035) ---

// TestContract_NewReportDMail_ValidatesSuccessfully verifies that a report
// D-Mail created by NewReportDMail passes ValidateDMail and marshals correctly.
func TestContract_NewReportDMail_ValidatesSuccessfully(t *testing.T) {
	// given
	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "AUTH-42",
		IssueTitle:  "Implement 2FA",
		MissionType: "implement",
		Status:      "success",
		PRUrl:       "https://github.com/test/repo/pull/123",
		Reason:      "All tests pass, 2FA implemented",
	}

	// when
	dmail := domain.NewReportDMail(report)

	// then: must pass validation
	if err := domain.ValidateDMail(dmail); err != nil {
		t.Fatalf("NewReportDMail produced invalid D-Mail: %v", err)
	}

	// then: kind must be "report"
	if dmail.Kind != "report" {
		t.Errorf("kind: got %q, want %q", dmail.Kind, "report")
	}

	// then: name must start with "report-"
	if !strings.HasPrefix(dmail.Name, "report-") {
		t.Errorf("name: got %q, want prefix %q", dmail.Name, "report-")
	}

	// then: marshal must succeed
	data, err := dmail.Marshal()
	if err != nil {
		t.Fatalf("MarshalDMail: %v", err)
	}

	// then: marshaled output must contain PR URL
	if !strings.Contains(string(data), "https://github.com/test/repo/pull/123") {
		t.Error("marshaled D-Mail should contain PR URL")
	}
}

// TestContract_NewReportDMail_OmitsPRWhenNone verifies that PR URL is
// omitted when set to empty string or "none".
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

			dmail := domain.NewReportDMail(report)

			if err := domain.ValidateDMail(dmail); err != nil {
				t.Fatalf("validation failed: %v", err)
			}

			data, _ := dmail.Marshal()
			if strings.Contains(string(data), "**PR:**") {
				t.Error("PR line should be omitted when URL is empty/none")
			}
		})
	}
}

// TestContract_NewReportDMail_FailedStatus verifies report D-Mail with
// failed status passes validation and includes reason in body.
func TestContract_NewReportDMail_FailedStatus(t *testing.T) {
	report := &domain.ExpeditionReport{
		Expedition:  3,
		IssueID:     "PERF-7",
		IssueTitle:  "Optimize query",
		MissionType: "optimize",
		Status:      "failed",
		Reason:      "OOM during benchmark",
	}

	dmail := domain.NewReportDMail(report)

	if err := domain.ValidateDMail(dmail); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	data, _ := dmail.Marshal()
	if !strings.Contains(string(data), "OOM during benchmark") {
		t.Error("body should contain failure reason")
	}
	if !strings.Contains(string(data), "failed") {
		t.Error("body should contain status")
	}
}
