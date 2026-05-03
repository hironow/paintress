package filter_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/filter"
)

// Phase 1.1A — Expedition prompt DomainStyle branch tests (Rival Contract v1.1).
//
// Plan: refs/plans/2026-05-03-rival-contract-v1-1-extensions.md §"Phase 1.1A"
//
// When an inbox specification D-Mail's metadata declares
// `domain_style: event-sourced`, the rendered expedition prompt MUST include
// a canonical command/event/read-model glossary preamble. For missing/
// generic/mixed (or no Rival Contract spec at all), the prompt MUST be
// bit-identical to v1 (no glossary preamble).

// rivalContractBodyDS mirrors dmail_test.go's rivalContractBody so the two
// test files do not share test-only symbols across files. Keep the fixture
// in sync with internal/harness/policy/testdata/rival/event-sourced-v1.md.
const rivalContractBodyDS = `# Contract: Add session expiry enforcement

## Intent
- Prevent expired sessions from authorizing API calls.

## Domain
- Command: ValidateSessionForRequest.
- Event: SessionValidationFailed.
- Read model: AuthMiddleware sees session status and expiry timestamp.

## Decisions
- Enforce expiry in middleware before handler execution.

## Steps
1. Add expiry check to auth middleware.

## Boundaries
- Do not add OAuth.

## Evidence
- test: just test
`

// glossaryMarker is a stable substring that MUST appear in the rendered
// prompt when (and only when) an inbox D-Mail declares
// domain_style: event-sourced. It is a load-bearing marker the production
// templates emit verbatim; tests assert presence/absence on this exact text.
const glossaryMarker = "Event-Sourced Domain Glossary"

// renderExpeditionPromptForLang is a small helper that builds a PromptData
// whose only varying input is the inbox D-Mail set, then renders the prompt
// in the requested language. It mirrors the call site in
// internal/session/expedition.go: the caller pre-renders the inbox section
// and computes HasEventSourcedContract from the same D-Mail slice, both
// derived from one inspection of the inbox.
func renderExpeditionPromptForLang(t *testing.T, lang string, dmails []domain.DMail) string {
	t.Helper()
	reg := filter.MustDefault()
	data := domain.PromptData{
		Number:                 1,
		Timestamp:              "2026-05-03 12:00:00",
		BaseBranch:             "main",
		ReserveSection:         "Reserve OK",
		InboxSection:           filter.FormatDMailForPrompt(dmails),
		MissionSection:         "mission",
		HasEventSourcedContract: filter.HasEventSourcedContract(dmails),
	}
	return filter.RenderExpeditionPrompt(reg, lang, data)
}

// TestRenderExpeditionPrompt_DomainStyleSwitchesGlossary asserts that the
// glossary preamble appears in all three languages when (and only when) the
// inbox contains a Rival Contract spec D-Mail with domain_style:event-sourced.
func TestRenderExpeditionPrompt_DomainStyleSwitchesGlossary(t *testing.T) {
	// given a Rival Contract spec D-Mail tagged event-sourced
	eventSourced := domain.DMail{
		Name:        "spec-add-expiry_abcdef01",
		Kind:        domain.KindSpecification,
		Description: "Add session expiry enforcement",
		Body:        rivalContractBodyDS,
		Metadata: map[string]string{
			"contract_schema":   "rival-contract-v1",
			"contract_id":       "auth-x",
			"contract_revision": "1",
			"domain_style":      "event-sourced",
		},
	}

	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			// when
			prompt := renderExpeditionPromptForLang(t, lang, []domain.DMail{eventSourced})

			// then — the glossary preamble is present
			if !strings.Contains(prompt, glossaryMarker) {
				t.Errorf("expected %q in rendered prompt for lang=%s, got:\n%s", glossaryMarker, lang, prompt)
			}
		})
	}
}

// TestRenderExpeditionPrompt_LegacyV1IdenticalToV1 is the regression guard
// that locks the v1 surface for missing/generic/mixed: in those cases the
// rendered expedition prompt MUST NOT contain the glossary preamble, and
// MUST be byte-identical across the four legacy variants (no inbox at all,
// missing domain_style, domain_style:generic, domain_style:mixed).
func TestRenderExpeditionPrompt_LegacyV1IdenticalToV1(t *testing.T) {
	// given four "legacy" inboxes that all MUST render identically:
	//  - no Rival Contract spec at all (empty inbox)
	//  - Rival Contract spec without domain_style key
	//  - Rival Contract spec with domain_style: generic
	//  - Rival Contract spec with domain_style: mixed
	emptyInbox := []domain.DMail{}

	missingStyle := []domain.DMail{
		{
			Name:        "spec-add-expiry_abcdef01",
			Kind:        domain.KindSpecification,
			Description: "Add session expiry enforcement",
			Body:        rivalContractBodyDS,
			Metadata: map[string]string{
				"contract_schema":   "rival-contract-v1",
				"contract_id":       "auth-x",
				"contract_revision": "1",
			},
		},
	}

	genericStyle := []domain.DMail{cloneDMailWithStyle(missingStyle[0], "generic")}
	mixedStyle := []domain.DMail{cloneDMailWithStyle(missingStyle[0], "mixed")}

	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			renderedEmpty := renderExpeditionPromptForLang(t, lang, emptyInbox)
			renderedMissing := renderExpeditionPromptForLang(t, lang, missingStyle)
			renderedGeneric := renderExpeditionPromptForLang(t, lang, genericStyle)
			renderedMixed := renderExpeditionPromptForLang(t, lang, mixedStyle)

			// then — none of the legacy renders contains the glossary marker
			for label, p := range map[string]string{
				"empty inbox":          renderedEmpty,
				"missing domain_style": renderedMissing,
				"generic domain_style": renderedGeneric,
				"mixed domain_style":   renderedMixed,
			} {
				if strings.Contains(p, glossaryMarker) {
					t.Errorf("legacy render %q for lang=%s leaked glossary marker; prompt:\n%s", label, lang, p)
				}
			}

			// then — the three Rival-Contract-but-non-event-sourced renders
			// MUST be byte-identical (the contract body itself is unchanged
			// across the three; only metadata.domain_style differs and that
			// MUST NOT influence the rendered prompt).
			if renderedMissing != renderedGeneric {
				t.Errorf("missing vs generic render diverged for lang=%s\nmissing:\n%s\ngeneric:\n%s", lang, renderedMissing, renderedGeneric)
			}
			if renderedMissing != renderedMixed {
				t.Errorf("missing vs mixed render diverged for lang=%s\nmissing:\n%s\nmixed:\n%s", lang, renderedMissing, renderedMixed)
			}
		})
	}
}

// cloneDMailWithStyle returns a copy of d with metadata.domain_style set to
// the given value. The map is freshly allocated so callers cannot leak
// mutations across cases.
func cloneDMailWithStyle(d domain.DMail, style string) domain.DMail {
	meta := make(map[string]string, len(d.Metadata)+1)
	for k, v := range d.Metadata {
		meta[k] = v
	}
	meta["domain_style"] = style
	d.Metadata = meta
	return d
}
