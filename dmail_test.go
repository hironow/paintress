package paintress

import (
	"testing"
)

func TestParseDMail_ValidFrontmatter(t *testing.T) {
	// given — full d-mail with all fields populated
	input := []byte(`---
name: "spec-my-42"
kind: specification
description: "Issue MY-42 is ready for implementation with explicit DoD"
issues:
  - MY-42
severity: medium
metadata:
  created_at: "2026-02-20T12:00:00Z"
---

# Rate Limiting Implementation

## Definition of Done
- Token bucket algorithm with configurable rate per API key
`)

	// when
	dm, err := ParseDMail(input)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dm.Name != "spec-my-42" {
		t.Errorf("Name = %q, want %q", dm.Name, "spec-my-42")
	}
	if dm.Kind != "specification" {
		t.Errorf("Kind = %q, want %q", dm.Kind, "specification")
	}
	if dm.Description != "Issue MY-42 is ready for implementation with explicit DoD" {
		t.Errorf("Description = %q, want %q", dm.Description, "Issue MY-42 is ready for implementation with explicit DoD")
	}
	if len(dm.Issues) != 1 || dm.Issues[0] != "MY-42" {
		t.Errorf("Issues = %v, want [MY-42]", dm.Issues)
	}
	if dm.Severity != "medium" {
		t.Errorf("Severity = %q, want %q", dm.Severity, "medium")
	}
	if dm.Metadata == nil {
		t.Fatal("Metadata is nil, want non-nil map")
	}
	if dm.Metadata["created_at"] != "2026-02-20T12:00:00Z" {
		t.Errorf("Metadata[created_at] = %q, want %q", dm.Metadata["created_at"], "2026-02-20T12:00:00Z")
	}
	if !containsStr(dm.Body, "# Rate Limiting Implementation") {
		t.Errorf("Body should contain heading, got %q", dm.Body)
	}
	if !containsStr(dm.Body, "Token bucket algorithm") {
		t.Errorf("Body should contain DoD content, got %q", dm.Body)
	}
}

func TestParseDMail_MinimalFields(t *testing.T) {
	// given — only required fields, no body
	input := []byte(`---
name: "report-my-99"
kind: report
description: "Minimal report"
---
`)

	// when
	dm, err := ParseDMail(input)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dm.Name != "report-my-99" {
		t.Errorf("Name = %q, want %q", dm.Name, "report-my-99")
	}
	if dm.Kind != "report" {
		t.Errorf("Kind = %q, want %q", dm.Kind, "report")
	}
	if dm.Description != "Minimal report" {
		t.Errorf("Description = %q, want %q", dm.Description, "Minimal report")
	}
	if dm.Issues != nil {
		t.Errorf("Issues = %v, want nil", dm.Issues)
	}
	if dm.Severity != "" {
		t.Errorf("Severity = %q, want empty", dm.Severity)
	}
	if dm.Metadata != nil {
		t.Errorf("Metadata = %v, want nil", dm.Metadata)
	}
	if dm.Body != "" {
		t.Errorf("Body = %q, want empty", dm.Body)
	}
}

func TestParseDMail_InvalidYAML(t *testing.T) {
	// given — malformed YAML between delimiters
	input := []byte(`---
name: [invalid yaml
  this: is: broken
---
`)

	// when
	_, err := ParseDMail(input)

	// then
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParseDMail_MissingDelimiter(t *testing.T) {
	// given — no opening --- at all
	input := []byte(`name: "no-delimiters"
kind: specification
description: "This has no frontmatter delimiters"
`)

	// when
	_, err := ParseDMail(input)

	// then
	if err == nil {
		t.Fatal("expected error for missing delimiter, got nil")
	}
}

func TestParseDMail_MissingClosingDelimiter(t *testing.T) {
	// given — opening --- but no closing ---
	input := []byte(`---
name: "no-closing"
kind: specification
description: "Missing closing delimiter"
`)

	// when
	_, err := ParseDMail(input)

	// then
	if err == nil {
		t.Fatal("expected error for missing closing delimiter, got nil")
	}
}

func TestDMailMarshal_RoundTrip(t *testing.T) {
	// given — DMail with all fields populated
	original := DMail{
		Name:        "spec-my-42",
		Kind:        "specification",
		Description: "Issue MY-42 is ready for implementation",
		Issues:      []string{"MY-42", "MY-43"},
		Severity:    "high",
		Metadata:    map[string]string{"created_at": "2026-02-20T12:00:00Z"},
		Body:        "# Implementation Plan\n\n- Step 1\n- Step 2\n",
	}

	// when — marshal then parse
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	parsed, err := ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}

	// then — all fields match
	if parsed.Name != original.Name {
		t.Errorf("Name = %q, want %q", parsed.Name, original.Name)
	}
	if parsed.Kind != original.Kind {
		t.Errorf("Kind = %q, want %q", parsed.Kind, original.Kind)
	}
	if parsed.Description != original.Description {
		t.Errorf("Description = %q, want %q", parsed.Description, original.Description)
	}
	if len(parsed.Issues) != len(original.Issues) {
		t.Fatalf("Issues length = %d, want %d", len(parsed.Issues), len(original.Issues))
	}
	for i, issue := range parsed.Issues {
		if issue != original.Issues[i] {
			t.Errorf("Issues[%d] = %q, want %q", i, issue, original.Issues[i])
		}
	}
	if parsed.Severity != original.Severity {
		t.Errorf("Severity = %q, want %q", parsed.Severity, original.Severity)
	}
	if len(parsed.Metadata) != len(original.Metadata) {
		t.Fatalf("Metadata length = %d, want %d", len(parsed.Metadata), len(original.Metadata))
	}
	for k, v := range original.Metadata {
		if parsed.Metadata[k] != v {
			t.Errorf("Metadata[%s] = %q, want %q", k, parsed.Metadata[k], v)
		}
	}
	if parsed.Body != original.Body {
		t.Errorf("Body = %q, want %q", parsed.Body, original.Body)
	}
}

func TestDMailMarshal_EmptyBody(t *testing.T) {
	// given — DMail with no body
	dm := DMail{
		Name:        "report-my-10",
		Kind:        "report",
		Description: "Empty body report",
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// then — should start with --- and end with --- without extra blank lines
	s := string(data)
	if !containsStr(s, "---\n") {
		t.Errorf("marshaled output missing --- delimiter")
	}

	// round-trip should produce empty body
	parsed, err := ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	if parsed.Body != "" {
		t.Errorf("Body = %q, want empty", parsed.Body)
	}
}
