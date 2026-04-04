package verifier

import (
	"fmt"

	"github.com/hironow/paintress/internal/domain"
)

// validActions is the set of valid action values per D-Mail schema v1.
// Strict on send, liberal on receive (Postel's law / S0021).
var validActions = map[string]bool{
	"retry":    true,
	"escalate": true,
	"resolve":  true,
}

// ValidateDMail checks that a DMail conforms to D-Mail schema v1.
func ValidateDMail(d domain.DMail) error {
	if d.SchemaVersion == "" {
		return fmt.Errorf("dmail: dmail-schema-version is required")
	}
	if d.SchemaVersion != domain.DMailSchemaVersion {
		return fmt.Errorf("dmail: unsupported dmail-schema-version: %q (want %q)", d.SchemaVersion, domain.DMailSchemaVersion)
	}
	if d.Name == "" {
		return fmt.Errorf("dmail: name is required")
	}
	if d.Kind == "" {
		return fmt.Errorf("dmail: kind is required")
	}
	if d.Description == "" {
		return fmt.Errorf("dmail: description is required")
	}
	if d.Action != "" && !validActions[d.Action] {
		return fmt.Errorf("dmail: invalid action %q (valid: retry, escalate, resolve)", d.Action)
	}
	return nil
}
