package filter_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/filter"
)

// rivalContractBody is a Rival Contract v1 specification body containing all
// six canonical sections. The fixture mirrors the one in
// internal/harness/policy/testdata/rival/valid-v1.md.
const rivalContractBody = `# Contract: Add session expiry enforcement

## Intent
- Prevent expired sessions from authorizing API calls.
- Success means expired sessions return 401 and active sessions continue to work.

## Domain
- Command: validate session for request.
- Event: session validation failed.
- Read model: auth middleware sees session status and expiry timestamp.

## Decisions
- Enforce expiry in middleware before handler execution.
- Reuse the existing session repository instead of adding a cache.

## Steps
1. Add expiry check to auth middleware.
   - Target: ` + "`internal/http/auth_middleware.go`" + `
   - Acceptance: expired sessions return 401.
2. Add unit tests for active, expired, and missing sessions.
   - Target: ` + "`tests/unit/auth_middleware_test.go`" + `
   - Acceptance: all three cases are covered.

## Boundaries
- Do not add OAuth, refresh tokens, or background cleanup.
- Do not change session table shape.
- Preserve existing error response format.

## Evidence
- test: just test
- lint: just lint
- nfr.p95_latency_ms: <= 200
- Add a regression test for expired sessions.
`

// legacySpecBody is a non-Rival-Contract specification body in the legacy
// action-list format. It MUST still render through the existing path.
const legacySpecBody = `# Add session expiry enforcement

## Actions
- Add expiry check to auth middleware in ` + "`internal/http/auth_middleware.go`" + `.
- Add unit tests for active, expired, and missing sessions.
- Update existing handler integration test if behavior changes.

## Acceptance
- Expired sessions return 401.
- Active sessions still authorize.
`

// TestFormatDMailForPrompt_RivalContractSpecIncludesContractSections verifies
// that a Rival Contract v1 specification body renders via the focused
// FormatRivalContractForPrompt path (Intent, Steps, Boundaries, Evidence).
func TestFormatDMailForPrompt_RivalContractSpecIncludesContractSections(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{
			Name:        "spec-add-expiry_abcdef01",
			Kind:        domain.KindSpecification,
			Description: "Add session expiry enforcement",
			Body:        rivalContractBody,
		},
	}

	// when
	result := filter.FormatDMailForPrompt(dmails)

	// then — focused Rival Contract section is present
	if !strings.Contains(result, "# Rival Contract: Add session expiry enforcement") {
		t.Errorf("expected focused Rival Contract heading, got %q", result)
	}
	if !strings.Contains(result, "## Intent") {
		t.Errorf("expected Intent section, got %q", result)
	}
	if !strings.Contains(result, "## Steps") {
		t.Errorf("expected Steps section, got %q", result)
	}
	if !strings.Contains(result, "## Boundaries") {
		t.Errorf("expected Boundaries section, got %q", result)
	}
	if !strings.Contains(result, "## Evidence") {
		t.Errorf("expected Evidence section, got %q", result)
	}
	// Domain and Decisions are intentionally omitted from the focused render
	// (per Phase 2 plan §"Production behavior").
	if strings.Contains(result, "## Domain") {
		t.Errorf("focused render should omit Domain, got %q", result)
	}
	if strings.Contains(result, "## Decisions") {
		t.Errorf("focused render should omit Decisions, got %q", result)
	}
	// Evidence content is preserved.
	if !strings.Contains(result, "nfr.p95_latency_ms: <= 200") {
		t.Errorf("expected NFR bullet preserved, got %q", result)
	}
}

// TestFormatDMailForPrompt_LegacySpecUsesExistingFormatting verifies that a
// non-Rival-Contract specification body still renders through the existing
// per-D-Mail header + body path (graceful fallback).
func TestFormatDMailForPrompt_LegacySpecUsesExistingFormatting(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{
			Name:        "spec-legacy-expiry_12345678",
			Kind:        domain.KindSpecification,
			Description: "Add session expiry enforcement (legacy)",
			Body:        legacySpecBody,
		},
	}

	// when
	result := filter.FormatDMailForPrompt(dmails)

	// then — legacy path still emits the standard header
	if !strings.Contains(result, "spec-legacy-expiry_12345678") {
		t.Errorf("expected D-Mail name in legacy render, got %q", result)
	}
	if !strings.Contains(result, "(specification)") {
		t.Errorf("expected kind annotation in legacy render, got %q", result)
	}
	if !strings.Contains(result, "Add session expiry enforcement (legacy)") {
		t.Errorf("expected description in legacy render, got %q", result)
	}
	// Legacy body content surfaces verbatim
	if !strings.Contains(result, "## Actions") {
		t.Errorf("expected legacy Actions heading preserved, got %q", result)
	}
	// Legacy render must NOT introduce a focused Rival Contract heading
	if strings.Contains(result, "# Rival Contract:") {
		t.Errorf("legacy render must not produce Rival Contract heading, got %q", result)
	}
}

// TestRenderExpeditionPrompt_IncludesRivalContractSection verifies that when
// a Rival Contract specification D-Mail is in the inbox, the expedition
// prompt surfaces the focused contract section to the implementer.
func TestRenderExpeditionPrompt_IncludesRivalContractSection(t *testing.T) {
	// given
	reg := filter.MustDefault()
	dmails := []domain.DMail{
		{
			Name:        "spec-add-expiry_abcdef01",
			Kind:        domain.KindSpecification,
			Description: "Add session expiry enforcement",
			Body:        rivalContractBody,
		},
	}
	data := domain.PromptData{
		Number:         1,
		Timestamp:      "2026-05-03 12:00:00",
		BaseBranch:     "main",
		ReserveSection: "Reserve OK",
		InboxSection:   filter.FormatDMailForPrompt(dmails),
		MissionSection: "mission",
	}

	// when
	prompt := filter.RenderExpeditionPrompt(reg, "en", data)

	// then
	if !strings.Contains(prompt, "# Rival Contract: Add session expiry enforcement") {
		t.Errorf("expected focused Rival Contract heading in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "## Boundaries") {
		t.Errorf("expected Boundaries surfaced in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "Do not add OAuth, refresh tokens, or background cleanup.") {
		t.Errorf("expected boundary content in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "If Boundaries conflict") {
		t.Errorf("expected Boundaries-precedence guidance in prompt, got %q", prompt)
	}
}

// TestRenderExpeditionPrompt_BoundaryAppearsOnce verifies that an exact-match
// duplicate boundary line in the contract body appears exactly once in the
// rendered expedition prompt. FormatRivalContractForPrompt deduplicates
// Boundaries lines on exact match — this test guards against any change in
// the consumer that would defeat that property.
func TestRenderExpeditionPrompt_BoundaryAppearsOnce(t *testing.T) {
	// given — body where the same boundary line appears twice
	bodyWithDuplicate := strings.Replace(
		rivalContractBody,
		"## Boundaries\n- Do not add OAuth, refresh tokens, or background cleanup.\n",
		"## Boundaries\n- Do not add OAuth, refresh tokens, or background cleanup.\n- Do not add OAuth, refresh tokens, or background cleanup.\n",
		1,
	)
	reg := filter.MustDefault()
	dmails := []domain.DMail{
		{
			Name:        "spec-add-expiry_abcdef01",
			Kind:        domain.KindSpecification,
			Description: "Add session expiry enforcement",
			Body:        bodyWithDuplicate,
		},
	}
	data := domain.PromptData{
		Number:         1,
		Timestamp:      "2026-05-03 12:00:00",
		BaseBranch:     "main",
		ReserveSection: "Reserve OK",
		InboxSection:   filter.FormatDMailForPrompt(dmails),
		MissionSection: "mission",
	}

	// when
	prompt := filter.RenderExpeditionPrompt(reg, "en", data)

	// then
	count := strings.Count(prompt, "Do not add OAuth, refresh tokens, or background cleanup.")
	if count != 1 {
		t.Errorf("expected boundary line to appear exactly once, got %d occurrences", count)
	}
}

// TestRenderExpeditionPrompt_WaveTargetStillComesFromWaveSteps verifies that
// even when a Rival Contract spec is in the inbox, wave target selection is
// driven by data.WaveTarget (sourced from wave.steps), NOT by contract Steps.
func TestRenderExpeditionPrompt_WaveTargetStillComesFromWaveSteps(t *testing.T) {
	// given — Rival Contract spec mentions different step phrasing than wave.steps
	reg := filter.MustDefault()
	dmails := []domain.DMail{
		{
			Name:        "spec-add-expiry_abcdef01",
			Kind:        domain.KindSpecification,
			Description: "Add session expiry enforcement",
			Body:        rivalContractBody,
		},
	}
	waveTarget := &domain.ExpeditionTarget{
		ID:          "step-1",
		Title:       "Implement middleware expiry check",
		Description: "wave-add-session-expiry",
		Acceptance:  "expired sessions return 401",
	}
	data := domain.PromptData{
		Number:         2,
		Timestamp:      "2026-05-03 12:00:00",
		BaseBranch:     "main",
		ReserveSection: "Reserve OK",
		InboxSection:   filter.FormatDMailForPrompt(dmails),
		MissionSection: "mission",
		WaveTarget:     waveTarget,
	}

	// when
	prompt := filter.RenderExpeditionPrompt(reg, "en", data)

	// then — Wave Target section comes from data.WaveTarget
	if !strings.Contains(prompt, "## Wave Target") {
		t.Errorf("expected Wave Target section, got %q", prompt)
	}
	if !strings.Contains(prompt, "step-1") {
		t.Errorf("expected wave step ID in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "Implement middleware expiry check") {
		t.Errorf("expected wave step title in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "expired sessions return 401") {
		t.Errorf("expected wave acceptance in prompt, got %q", prompt)
	}
	// Contract sections also appear (they supplement, not replace).
	if !strings.Contains(prompt, "# Rival Contract:") {
		t.Errorf("expected Rival Contract section to coexist with Wave Target, got %q", prompt)
	}
}
