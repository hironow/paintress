package filter_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/filter"
)

// TestRenderExpeditionPrompt_EventSourcedRenderShape is a shape-only smoke
// test that locks the structural placement of the v1.1 glossary preamble:
// the glossary must appear AFTER the inbox section and BEFORE the existing
// "Contract Boundaries (Rival Contract v1)" heading. This test does not
// assert on the verbatim glossary text (other tests cover the content); it
// guards against future template edits that would re-order the sections in
// a way that confuses the implementer.
func TestRenderExpeditionPrompt_EventSourcedRenderShape(t *testing.T) {
	body := `# Contract: X

## Intent
- a

## Domain
- Command: C.

## Decisions
- d

## Steps
1. s

## Boundaries
- b

## Evidence
- test: just test
`
	dmail := domain.DMail{
		Name:        "spec-x_abcdef01",
		Kind:        domain.KindSpecification,
		Description: "X",
		Body:        body,
		Metadata: map[string]string{
			"contract_schema":   "rival-contract-v1",
			"contract_id":       "x",
			"contract_revision": "1",
			"domain_style":      "event-sourced",
		},
	}
	reg := filter.MustDefault()
	data := domain.PromptData{
		Number:                  1,
		Timestamp:               "2026-05-03 12:00:00",
		BaseBranch:              "main",
		ReserveSection:          "Reserve OK",
		InboxSection:            filter.FormatDMailForPrompt([]domain.DMail{dmail}),
		MissionSection:          "mission",
		HasEventSourcedContract: true,
	}
	prompt := filter.RenderExpeditionPrompt(reg, "en", data)

	inbox := strings.Index(prompt, "# Rival Contract: X")
	glossary := strings.Index(prompt, "Event-Sourced Domain Glossary")
	boundaries := strings.Index(prompt, "Contract Boundaries (Rival Contract v1)")

	if inbox < 0 || glossary < 0 || boundaries < 0 {
		t.Fatalf("missing one of the markers: inbox=%d glossary=%d boundaries=%d", inbox, glossary, boundaries)
	}
	if !(inbox < glossary && glossary < boundaries) {
		t.Errorf("glossary must appear between inbox and Contract Boundaries; got positions inbox=%d glossary=%d boundaries=%d\nprompt:\n%s", inbox, glossary, boundaries, prompt)
	}
}
