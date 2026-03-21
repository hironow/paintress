package platform_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// TestExpeditionJaTemplate_ContainsCapabilityBoundarySection verifies that
// the Japanese expedition template includes the Capability Boundary section.
func TestExpeditionJaTemplate_ContainsCapabilityBoundarySection(t *testing.T) {
	// given: minimal prompt data
	data := domain.PromptData{
		Number:    1,
		Timestamp: "2026-01-01",
		Bt:        "`",
		Cb:        "```",
	}

	// when
	prompt := platform.RenderExpeditionPrompt("ja", data)

	// then: template must include capability boundary section
	if !strings.Contains(prompt, "Capability Boundary") && !strings.Contains(prompt, "ケイパビリティ境界") {
		t.Errorf("expedition_ja.md.tmpl missing Capability Boundary section.\nPrompt excerpt:\n%s", prompt[:min(500, len(prompt))])
	}
}

// TestExpeditionJaTemplate_CapabilityBoundarySectionHasMinimumLines verifies
// the capability boundary section contains at least 18 lines of content.
func TestExpeditionJaTemplate_CapabilityBoundarySectionHasMinimumLines(t *testing.T) {
	// given: minimal prompt data
	data := domain.PromptData{
		Number:    1,
		Timestamp: "2026-01-01",
		Bt:        "`",
		Cb:        "```",
	}

	// when
	prompt := platform.RenderExpeditionPrompt("ja", data)

	// then: find the capability boundary section and count its lines
	var sectionStart int
	for _, marker := range []string{"## Capability Boundary", "## ケイパビリティ境界"} {
		idx := strings.Index(prompt, marker)
		if idx != -1 {
			sectionStart = idx
			break
		}
	}

	if sectionStart == 0 && !strings.HasPrefix(prompt, "##") {
		t.Fatal("Capability Boundary section not found in expedition_ja.md.tmpl")
	}

	// Extract from section start to next section or end
	sectionText := prompt[sectionStart:]
	// Find next section header
	nextSection := strings.Index(sectionText[3:], "\n## ")
	if nextSection != -1 {
		sectionText = sectionText[:nextSection+3]
	}

	lines := strings.Split(strings.TrimSpace(sectionText), "\n")
	if len(lines) < 18 {
		t.Errorf("Capability Boundary section has %d lines, want at least 18.\nSection:\n%s", len(lines), sectionText)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
