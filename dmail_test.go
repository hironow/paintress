package paintress

import (
	"os"
	"path/filepath"
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

// === Path Function Tests ===

func TestInboxDir(t *testing.T) {
	// given
	continent := "/tmp/myrepo"

	// when
	got := InboxDir(continent)

	// then
	want := filepath.Join("/tmp/myrepo", ".expedition", "inbox")
	if got != want {
		t.Errorf("InboxDir() = %q, want %q", got, want)
	}
}

func TestOutboxDir(t *testing.T) {
	// given
	continent := "/tmp/myrepo"

	// when
	got := OutboxDir(continent)

	// then
	want := filepath.Join("/tmp/myrepo", ".expedition", "outbox")
	if got != want {
		t.Errorf("OutboxDir() = %q, want %q", got, want)
	}
}

func TestArchiveDir(t *testing.T) {
	// given
	continent := "/tmp/myrepo"

	// when
	got := ArchiveDir(continent)

	// then
	want := filepath.Join("/tmp/myrepo", ".expedition", "archive")
	if got != want {
		t.Errorf("ArchiveDir() = %q, want %q", got, want)
	}
}

// === SendDMail Tests ===

func TestSendDMail_WritesToOutboxAndArchive(t *testing.T) {
	// given
	continent := t.TempDir()
	dm := DMail{
		Name:        "spec-my-42",
		Kind:        "specification",
		Description: "Test sending d-mail",
		Body:        "# Hello\n",
	}

	// when
	err := SendDMail(continent, dm)

	// then
	if err != nil {
		t.Fatalf("SendDMail error: %v", err)
	}

	outboxPath := filepath.Join(OutboxDir(continent), "spec-my-42.md")
	archivePath := filepath.Join(ArchiveDir(continent), "spec-my-42.md")

	outboxData, err := os.ReadFile(outboxPath)
	if err != nil {
		t.Fatalf("failed to read outbox file: %v", err)
	}
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read archive file: %v", err)
	}

	// Content in both locations must be identical
	if string(outboxData) != string(archiveData) {
		t.Errorf("outbox and archive content differ:\noutbox:  %q\narchive: %q", outboxData, archiveData)
	}

	// Verify the content is a valid d-mail
	parsed, err := ParseDMail(outboxData)
	if err != nil {
		t.Fatalf("ParseDMail on outbox file: %v", err)
	}
	if parsed.Name != "spec-my-42" {
		t.Errorf("parsed Name = %q, want %q", parsed.Name, "spec-my-42")
	}
	if parsed.Body != "# Hello\n" {
		t.Errorf("parsed Body = %q, want %q", parsed.Body, "# Hello\n")
	}
}

func TestSendDMail_CreatesDirectories(t *testing.T) {
	// given — clean temp dir with no .expedition at all
	continent := t.TempDir()
	dm := DMail{
		Name:        "report-create-dirs",
		Kind:        "report",
		Description: "Dirs should be auto-created",
	}

	// when
	err := SendDMail(continent, dm)

	// then
	if err != nil {
		t.Fatalf("SendDMail error: %v", err)
	}

	// Both files must exist
	outboxPath := filepath.Join(OutboxDir(continent), "report-create-dirs.md")
	archivePath := filepath.Join(ArchiveDir(continent), "report-create-dirs.md")

	if _, err := os.Stat(outboxPath); err != nil {
		t.Errorf("outbox file not found: %v", err)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archive file not found: %v", err)
	}
}

// === ScanInbox Tests ===

func TestScanInbox_ReadsAllMdFiles(t *testing.T) {
	// given — two d-mails in inbox
	continent := t.TempDir()
	inboxDir := InboxDir(continent)
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatal(err)
	}

	dm1 := DMail{Name: "alpha", Kind: "report", Description: "First"}
	dm2 := DMail{Name: "beta", Kind: "specification", Description: "Second", Body: "Details\n"}

	data1, err := dm1.Marshal()
	if err != nil {
		t.Fatalf("setup: Marshal dm1: %v", err)
	}
	data2, err := dm2.Marshal()
	if err != nil {
		t.Fatalf("setup: Marshal dm2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "alpha.md"), data1, 0644); err != nil {
		t.Fatalf("setup: write alpha.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "beta.md"), data2, 0644); err != nil {
		t.Fatalf("setup: write beta.md: %v", err)
	}

	// when
	results, err := ScanInbox(continent)

	// then
	if err != nil {
		t.Fatalf("ScanInbox error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	// Sorted by filename: alpha before beta
	if results[0].Name != "alpha" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "alpha")
	}
	if results[1].Name != "beta" {
		t.Errorf("results[1].Name = %q, want %q", results[1].Name, "beta")
	}
	if results[1].Body != "Details\n" {
		t.Errorf("results[1].Body = %q, want %q", results[1].Body, "Details\n")
	}
}

func TestScanInbox_EmptyDir(t *testing.T) {
	// given — empty inbox directory
	continent := t.TempDir()
	inboxDir := InboxDir(continent)
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatal(err)
	}

	// when
	results, err := ScanInbox(continent)

	// then
	if err != nil {
		t.Fatalf("ScanInbox error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestScanInbox_SkipsNonMd(t *testing.T) {
	// given — one .md and one .txt in inbox
	continent := t.TempDir()
	inboxDir := InboxDir(continent)
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatal(err)
	}

	dm := DMail{Name: "valid", Kind: "report", Description: "Valid d-mail"}
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("setup: Marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "valid.md"), data, 0644); err != nil {
		t.Fatalf("setup: write valid.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "notes.txt"), []byte("not a d-mail"), 0644); err != nil {
		t.Fatalf("setup: write notes.txt: %v", err)
	}

	// when
	results, err := ScanInbox(continent)

	// then
	if err != nil {
		t.Fatalf("ScanInbox error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Name != "valid" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "valid")
	}
}

func TestScanInbox_NonexistentDir(t *testing.T) {
	// given — no inbox directory exists at all
	continent := t.TempDir()

	// when
	results, err := ScanInbox(continent)

	// then — returns empty slice, not error
	if err != nil {
		t.Fatalf("ScanInbox error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

// === ArchiveInboxDMail Tests ===

func TestArchiveInboxDMail_MovesToArchive(t *testing.T) {
	// given — a d-mail sitting in inbox
	continent := t.TempDir()
	inboxDir := InboxDir(continent)
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatal(err)
	}

	dm := DMail{Name: "move-me", Kind: "report", Description: "To be archived"}
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("setup: Marshal: %v", err)
	}
	inboxPath := filepath.Join(inboxDir, "move-me.md")
	if err := os.WriteFile(inboxPath, data, 0644); err != nil {
		t.Fatalf("setup: write move-me.md: %v", err)
	}

	// when
	err = ArchiveInboxDMail(continent, "move-me")

	// then
	if err != nil {
		t.Fatalf("ArchiveInboxDMail error: %v", err)
	}

	// Gone from inbox
	if _, err := os.Stat(inboxPath); !os.IsNotExist(err) {
		t.Errorf("inbox file still exists after archive")
	}

	// Present in archive
	archivePath := filepath.Join(ArchiveDir(continent), "move-me.md")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("archive file not found: %v", err)
	}

	// Content is preserved
	parsed, err := ParseDMail(archiveData)
	if err != nil {
		t.Fatalf("ParseDMail on archived file: %v", err)
	}
	if parsed.Name != "move-me" {
		t.Errorf("parsed Name = %q, want %q", parsed.Name, "move-me")
	}
}
