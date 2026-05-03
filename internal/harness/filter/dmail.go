package filter

import (
	"fmt"
	"strings"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/policy"
)

// HasEventSourcedContract reports whether any inbox D-Mail is a
// Rival Contract v1 specification carrying metadata.domain_style ==
// "event-sourced". The check is pure: it never inspects environment
// variables, ADRs, or any side channel — only the D-Mail metadata as
// parsed by policy.ParseRivalContractMetadata.
//
// Callers (e.g. internal/session/expedition.go) pair this with
// FormatDMailForPrompt: one inbox scan, two derived fields. Consumers of
// PromptData (the prompt template) flip the canonical command/event/
// read-model glossary preamble on this boolean. For missing/generic/mixed
// (or no Rival Contract spec at all), this returns false and the
// rendered prompt remains bit-identical to the legacy v1 surface.
func HasEventSourcedContract(dmails []domain.DMail) bool {
	for _, dm := range dmails {
		if dm.Kind != domain.KindSpecification {
			continue
		}
		meta, ok, err := policy.ParseRivalContractMetadata(dm.Metadata)
		if err != nil || !ok {
			continue
		}
		if meta.DomainStyle == policy.DomainStyleEventSourced {
			return true
		}
	}
	return false
}

// FormatDMailForPrompt formats d-mails as a human-readable Markdown section
// for injection into expedition prompts. Returns empty string for empty input.
//
// Specification D-Mails whose body parses as a Rival Contract v1 contract
// are rendered through the focused FormatRivalContractForPrompt path so
// that Intent, Steps, Boundaries, and Evidence are emphasized over the raw
// per-D-Mail header. Legacy specification bodies (no `# Contract:` heading
// or partial v1 bodies) gracefully fall back to the existing per-D-Mail
// header + body path.
func FormatDMailForPrompt(dmails []domain.DMail) string {
	if len(dmails) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, dm := range dmails {
		if rendered, ok := renderRivalContractDMail(dm); ok {
			buf.WriteString(rendered)
			continue
		}
		buf.WriteString(renderLegacyDMail(dm))
	}
	return buf.String()
}

// renderRivalContractDMail returns a focused Rival Contract section for a
// specification D-Mail whose body parses as a complete Rival Contract v1
// body. The second return value is false when the D-Mail is not a Rival
// Contract (legacy spec, partial body, or non-spec kind), signalling the
// caller to fall back to the legacy render.
func renderRivalContractDMail(dm domain.DMail) (string, bool) {
	if dm.Kind != domain.KindSpecification {
		return "", false
	}
	if dm.Body == "" {
		return "", false
	}
	contract, ok, err := policy.ParseRivalContractBody(dm.Body)
	if err != nil || !ok {
		return "", false
	}
	rendered := policy.FormatRivalContractForPrompt(contract)
	if !strings.HasSuffix(rendered, "\n") {
		rendered += "\n"
	}
	return rendered + "\n", true
}

// renderLegacyDMail formats a D-Mail through the original per-D-Mail header
// + body path. This path remains the canonical render for non-spec D-Mails
// and for legacy specification bodies that pre-date Rival Contract v1.
func renderLegacyDMail(dm domain.DMail) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "### %s (%s)\n\n", dm.Name, dm.Kind)
	fmt.Fprintf(&buf, "**Description:** %s\n", dm.Description)
	if len(dm.Issues) > 0 {
		fmt.Fprintf(&buf, "**Issues:** %s\n", strings.Join(dm.Issues, ", "))
	}
	if dm.Severity != "" {
		fmt.Fprintf(&buf, "**Severity:** %s\n", dm.Severity)
	}
	if dm.Body != "" {
		buf.WriteString("\n")
		buf.WriteString(dm.Body)
		if !strings.HasSuffix(dm.Body, "\n") {
			buf.WriteString("\n")
		}
	}
	buf.WriteString("\n")
	return buf.String()
}

// BuildFollowUpPrompt builds a follow-up prompt for issue-matched D-Mails
// received mid-expedition. Returns empty string for empty input.
func BuildFollowUpPrompt(dmails []domain.DMail) string {
	if len(dmails) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("The following D-Mail(s) arrived during this expedition and are related to the issue you just worked on.\n")
	buf.WriteString("Review them and take any additional action if needed. If no action is required, briefly acknowledge.\n\n")
	buf.WriteString(FormatDMailForPrompt(dmails))
	return buf.String()
}
