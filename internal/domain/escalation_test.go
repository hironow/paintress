package domain_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestNewEscalationDMail_Kind(t *testing.T) {
	// given
	exp := 5
	failures := 3

	// when
	dm := domain.NewEscalationDMail(exp, failures)

	// then
	if dm.Kind != "feedback" {
		t.Errorf("Kind = %q, want %q", dm.Kind, "feedback")
	}
}

func TestNewEscalationDMail_Severity(t *testing.T) {
	// given
	exp := 5
	failures := 3

	// when
	dm := domain.NewEscalationDMail(exp, failures)

	// then
	if dm.Severity != "high" {
		t.Errorf("Severity = %q, want %q", dm.Severity, "high")
	}
}

func TestNewEscalationDMail_SchemaVersion(t *testing.T) {
	// when
	dm := domain.NewEscalationDMail(3, 3)

	// then
	if dm.SchemaVersion != domain.DMailSchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", dm.SchemaVersion, domain.DMailSchemaVersion)
	}
}

func TestNewEscalationDMail_NameContainsExpedition(t *testing.T) {
	// when
	dm := domain.NewEscalationDMail(7, 3)

	// then
	if !strings.Contains(dm.Name, "7") {
		t.Errorf("Name = %q, want to contain expedition number 7", dm.Name)
	}
	if !strings.HasPrefix(dm.Name, "feedback-escalation-") {
		t.Errorf("Name = %q, want prefix %q", dm.Name, "feedback-escalation-")
	}
}

func TestNewEscalationDMail_BodyContainsFailureCount(t *testing.T) {
	// when
	dm := domain.NewEscalationDMail(5, 3)

	// then
	if !strings.Contains(dm.Body, "3") {
		t.Errorf("Body should contain failure count 3, got %q", dm.Body)
	}
}

func TestNewEscalationDMail_DescriptionMentionsEscalation(t *testing.T) {
	// when
	dm := domain.NewEscalationDMail(10, 3)

	// then
	if !strings.Contains(strings.ToLower(dm.Description), "escalation") {
		t.Errorf("Description = %q, want to contain 'escalation'", dm.Description)
	}
}

func TestNewEscalationDMail_Marshal_Roundtrip(t *testing.T) {
	// given
	dm := domain.NewEscalationDMail(5, 3)

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail: %v", err)
	}

	// then
	if parsed.Kind != "feedback" {
		t.Errorf("parsed Kind = %q, want %q", parsed.Kind, "feedback")
	}
	if parsed.Severity != "high" {
		t.Errorf("parsed Severity = %q, want %q", parsed.Severity, "high")
	}
	if parsed.Name != dm.Name {
		t.Errorf("parsed Name = %q, want %q", parsed.Name, dm.Name)
	}
}
