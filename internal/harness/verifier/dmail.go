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

// ParseDMail validates a DMail against D-Mail schema v1, returning the validated DMail or an error.
func ParseDMail(d domain.DMail) (domain.DMail, error) {
	if d.SchemaVersion == "" {
		return domain.DMail{}, fmt.Errorf("dmail: dmail-schema-version is required")
	}
	if d.SchemaVersion != domain.DMailSchemaVersion {
		return domain.DMail{}, fmt.Errorf("dmail: unsupported dmail-schema-version: %q (want %q)", d.SchemaVersion, domain.DMailSchemaVersion)
	}
	if d.Name == "" {
		return domain.DMail{}, fmt.Errorf("dmail: name is required")
	}
	if d.Kind == "" {
		return domain.DMail{}, fmt.Errorf("dmail: kind is required")
	}
	if _, err := domain.ParseKind(d.Kind); err != nil {
		return domain.DMail{}, err
	}
	if d.Description == "" {
		return domain.DMail{}, fmt.Errorf("dmail: description is required")
	}
	if d.Action != "" && !validActions[d.Action] {
		return domain.DMail{}, fmt.Errorf("dmail: invalid action %q (valid: retry, escalate, resolve)", d.Action)
	}
	return d, nil
}

// ValidateDMail checks that a DMail conforms to D-Mail schema v1.
//
// Deprecated: prefer ParseDMail which returns the validated DMail.
func ValidateDMail(d domain.DMail) error { // nosemgrep: parse-dont-validate.validate-returns-error-only-go -- backward-compat wrapper; ParseDMail is the canonical parse function [permanent]
	_, err := ParseDMail(d)
	return err
}
