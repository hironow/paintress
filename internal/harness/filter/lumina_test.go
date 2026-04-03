package filter_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/filter"
)

func TestFormatLuminaForPrompt_SingleLumina(t *testing.T) {
	luminas := []domain.Lumina{
		{Pattern: "only one pattern", Source: "failure-pattern", Uses: 1},
	}
	result := filter.FormatLuminaForPrompt(luminas)
	if !strings.Contains(result, "only one pattern") {
		t.Errorf("should contain pattern: %q", result)
	}
	// Should contain section header and bullet
	if !strings.Contains(result, "Defensive") {
		t.Errorf("should contain Defensive header: %q", result)
	}
	if !strings.Contains(result, "- only one pattern") {
		t.Errorf("should contain bulleted pattern: %q", result)
	}
}
