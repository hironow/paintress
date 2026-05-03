package policy_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/harness/policy"
)

func readRivalFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", "rival", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}

func TestParseRivalContractBody_ValidV1(t *testing.T) {
	// given
	body := readRivalFixture(t, "valid-v1.md")

	// when
	contract, ok, err := policy.ParseRivalContractBody(body)

	// then
	if err != nil {
		t.Fatalf("ParseRivalContractBody: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("ParseRivalContractBody: expected ok=true for valid v1 body")
	}
	if contract.Title != "Add session expiry enforcement" {
		t.Errorf("Title: got %q", contract.Title)
	}
	if !strings.Contains(contract.Intent, "Prevent expired sessions") {
		t.Errorf("Intent missing expected text: %q", contract.Intent)
	}
	if !strings.Contains(contract.Domain, "validate session for request") {
		t.Errorf("Domain missing expected text: %q", contract.Domain)
	}
	if !strings.Contains(contract.Decisions, "Enforce expiry in middleware") {
		t.Errorf("Decisions missing expected text: %q", contract.Decisions)
	}
	if !strings.Contains(contract.Steps, "Add expiry check to auth middleware") {
		t.Errorf("Steps missing expected text: %q", contract.Steps)
	}
	if !strings.Contains(contract.Boundaries, "Do not add OAuth") {
		t.Errorf("Boundaries missing expected text: %q", contract.Boundaries)
	}
	if !strings.Contains(contract.Evidence, "test: just test") {
		t.Errorf("Evidence missing expected text: %q", contract.Evidence)
	}
}

func TestParseRivalContractBody_LegacyReturnsFalse(t *testing.T) {
	// given
	body := readRivalFixture(t, "legacy-spec.md")

	// when
	_, ok, err := policy.ParseRivalContractBody(body)

	// then
	if err != nil {
		t.Fatalf("ParseRivalContractBody on legacy body must not error: %v", err)
	}
	if ok {
		t.Fatal("ParseRivalContractBody: expected ok=false for legacy body without # Contract: heading")
	}
}

func TestParseRivalContractBody_PartialReturnsError(t *testing.T) {
	// given
	body := readRivalFixture(t, "partial-v1.md")

	// when
	_, ok, err := policy.ParseRivalContractBody(body)

	// then
	if err == nil {
		t.Fatal("ParseRivalContractBody: expected error for partial v1 body")
	}
	if ok {
		t.Errorf("ParseRivalContractBody: expected ok=false on error, got ok=true")
	}
}

func TestParseEvidenceItems_ParsesSupportedKeys(t *testing.T) {
	// given
	evidence := strings.Join([]string{
		"- check: just check",
		"- test: just test",
		"- lint: just lint",
		"- semgrep: just semgrep",
		"- nfr.p95_latency_ms: <= 200",
		"- nfr.error_rate_percent: <= 1",
		"- nfr.success_rate_percent: >= 99",
		"- nfr.target_rps: >= 50",
	}, "\n")

	// when
	items := policy.ParseEvidenceItems(evidence)

	// then
	want := map[string]struct {
		Operator string
		Value    string
	}{
		"check":                    {"", "just check"},
		"test":                     {"", "just test"},
		"lint":                     {"", "just lint"},
		"semgrep":                  {"", "just semgrep"},
		"nfr.p95_latency_ms":       {"<=", "200"},
		"nfr.error_rate_percent":   {"<=", "1"},
		"nfr.success_rate_percent": {">=", "99"},
		"nfr.target_rps":           {">=", "50"},
	}
	if len(items) != len(want) {
		t.Fatalf("ParseEvidenceItems: got %d items, want %d (items=%+v)", len(items), len(want), items)
	}
	for _, item := range items {
		expected, found := want[item.Key]
		if !found {
			t.Errorf("unexpected key %q", item.Key)
			continue
		}
		if item.Operator != expected.Operator {
			t.Errorf("key %q: operator got %q want %q", item.Key, item.Operator, expected.Operator)
		}
		if item.Value != expected.Value {
			t.Errorf("key %q: value got %q want %q", item.Key, item.Value, expected.Value)
		}
	}
}

func TestParseEvidenceItems_IgnoresUnknownAndProse(t *testing.T) {
	// given
	evidence := strings.Join([]string{
		"- Add a regression test for expired sessions.",
		"- test: just test",
		"- unknown.key: 1",
		"- nfr.unknown_metric: <= 99",
		"Plain prose without bullet.",
		"- still prose without colon",
	}, "\n")

	// when
	items := policy.ParseEvidenceItems(evidence)

	// then
	if len(items) != 1 {
		t.Fatalf("ParseEvidenceItems: expected 1 item (only test), got %d (items=%+v)", len(items), items)
	}
	if items[0].Key != "test" {
		t.Errorf("expected only key 'test', got %q", items[0].Key)
	}
	if items[0].Value != "just test" {
		t.Errorf("expected value 'just test', got %q", items[0].Value)
	}
}

func TestDeriveContractID_PrefersWaveID(t *testing.T) {
	// when
	id, err := policy.DeriveContractID("auth-session-expiry", []string{"ISS-2", "ISS-1"}, "auth-cluster")

	// then
	if err != nil {
		t.Fatalf("DeriveContractID: unexpected error: %v", err)
	}
	if id != "auth-session-expiry" {
		t.Errorf("DeriveContractID: expected wave ID, got %q", id)
	}
}

func TestDeriveContractID_RejectsDMailNameFallback(t *testing.T) {
	// when no wave / issues / cluster is available
	id, err := policy.DeriveContractID("", nil, "")

	// then
	if err == nil {
		t.Fatalf("DeriveContractID: expected error when no stable input, got id=%q", id)
	}
	if !errors.Is(err, policy.ErrContractIDUnavailable) {
		t.Errorf("DeriveContractID: expected ErrContractIDUnavailable, got %v", err)
	}
}

func TestFormatRivalContractForPrompt_IncludesIntentBoundariesEvidence(t *testing.T) {
	// given a fully populated Rival Contract
	rc := policy.RivalContract{
		Title:      "Add session expiry enforcement",
		Intent:     "- Prevent expired sessions from authorizing API calls.",
		Domain:     "- Command: validate session for request.",
		Decisions:  "- Enforce expiry in middleware before handler execution.",
		Steps:      "1. Add expiry check to auth middleware.",
		Boundaries: "- Do not add OAuth, refresh tokens, or background cleanup.",
		Evidence:   "- test: just test",
	}

	// when
	prompt := policy.FormatRivalContractForPrompt(rc)

	// then
	if !strings.Contains(prompt, "Add session expiry enforcement") {
		t.Errorf("prompt missing title: %q", prompt)
	}
	if !strings.Contains(prompt, "Prevent expired sessions from authorizing API calls.") {
		t.Errorf("prompt missing Intent body: %q", prompt)
	}
	if !strings.Contains(prompt, "Add expiry check to auth middleware.") {
		t.Errorf("prompt missing Steps body: %q", prompt)
	}
	if !strings.Contains(prompt, "Do not add OAuth, refresh tokens, or background cleanup.") {
		t.Errorf("prompt missing Boundaries body: %q", prompt)
	}
	if !strings.Contains(prompt, "test: just test") {
		t.Errorf("prompt missing Evidence body: %q", prompt)
	}
	// Domain and Decisions are not part of the focused expedition prompt section.
	if strings.Contains(prompt, "Command: validate session for request.") {
		t.Errorf("prompt should not include Domain content (not part of Phase 2 prompt section): %q", prompt)
	}
	if strings.Contains(prompt, "Enforce expiry in middleware before handler execution.") {
		t.Errorf("prompt should not include Decisions content (not part of Phase 2 prompt section): %q", prompt)
	}
}

func TestFormatRivalContractForPrompt_DoesNotDuplicateBoundary(t *testing.T) {
	// given a contract whose Boundaries section repeats the same line
	rc := policy.RivalContract{
		Title:      "Some contract",
		Intent:     "- intent",
		Domain:     "- domain",
		Decisions:  "- decisions",
		Steps:      "1. step",
		Boundaries: "- Do not add OAuth.\n- Do not add OAuth.\n- Preserve error format.",
		Evidence:   "- test: just test",
	}

	// when
	prompt := policy.FormatRivalContractForPrompt(rc)

	// then
	first := strings.Index(prompt, "Do not add OAuth.")
	if first < 0 {
		t.Fatalf("prompt missing Boundary line: %q", prompt)
	}
	rest := prompt[first+len("Do not add OAuth."):]
	if strings.Contains(rest, "Do not add OAuth.") {
		t.Errorf("Boundary line was duplicated in prompt: %q", prompt)
	}
	if !strings.Contains(prompt, "Preserve error format.") {
		t.Errorf("non-duplicate Boundary line missing: %q", prompt)
	}
}
