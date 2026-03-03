package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestStatusReport_FormatText(t *testing.T) {
	// given
	report := StatusReport{
		Continent:      "/path/to/repo",
		Expeditions:    15,
		Successes:      12,
		Failures:       2,
		SuccessRate:    0.857,
		GradientLevel:  3,
		InboxCount:     1,
		ArchiveCount:   10,
		LastExpedition: time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC),
	}

	// when
	text := report.FormatText()

	// then — verify key lines are present
	expected := []string{
		"paintress status:",
		"Continent:",
		"/path/to/repo",
		"Expeditions:",
		"15",
		"12 success",
		"2 failed",
		"Success rate:",
		"85.7%",
		"Gradient:",
		"level 3",
		"Inbox:",
		"1 pending",
		"Archive:",
		"10 processed",
		"Last expedition:",
	}
	for _, s := range expected {
		if !strings.Contains(text, s) {
			t.Errorf("expected output to contain %q, got:\n%s", s, text)
		}
	}
}

func TestStatusReport_FormatText_NoEvents(t *testing.T) {
	// given — zero-value report
	report := StatusReport{}

	// when
	text := report.FormatText()

	// then
	if !strings.Contains(text, "no expeditions yet") {
		t.Errorf("expected 'no expeditions yet' for zero time, got:\n%s", text)
	}
}

func TestStatusReport_FormatJSON(t *testing.T) {
	// given
	report := StatusReport{
		Continent:      "/path/to/repo",
		Expeditions:    15,
		Successes:      12,
		Failures:       2,
		SuccessRate:    0.857,
		GradientLevel:  3,
		InboxCount:     1,
		ArchiveCount:   10,
		LastExpedition: time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC),
	}

	// when
	data := report.FormatJSON()

	// then — verify it's valid JSON with expected fields
	var parsed map[string]any
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, data)
	}
	if parsed["expeditions"] != float64(15) {
		t.Errorf("expected expeditions=15, got %v", parsed["expeditions"])
	}
	if parsed["successes"] != float64(12) {
		t.Errorf("expected successes=12, got %v", parsed["successes"])
	}
	if parsed["inbox_count"] != float64(1) {
		t.Errorf("expected inbox_count=1, got %v", parsed["inbox_count"])
	}
	if parsed["continent"] != "/path/to/repo" {
		t.Errorf("expected continent=/path/to/repo, got %v", parsed["continent"])
	}
}
