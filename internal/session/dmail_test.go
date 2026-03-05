package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
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
	dm, err := domain.ParseDMail(input)

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
	if !strings.Contains(dm.Body, "# Rate Limiting Implementation") {
		t.Errorf("Body should contain heading, got %q", dm.Body)
	}
	if !strings.Contains(dm.Body, "Token bucket algorithm") {
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
	dm, err := domain.ParseDMail(input)

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
	_, err := domain.ParseDMail(input)

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
	_, err := domain.ParseDMail(input)

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
	_, err := domain.ParseDMail(input)

	// then
	if err == nil {
		t.Fatal("expected error for missing closing delimiter, got nil")
	}
}

func TestDMailMarshal_RoundTrip(t *testing.T) {
	// given — DMail with all fields populated
	original := domain.DMail{
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

	parsed, err := domain.ParseDMail(data)
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
	for k, v := range original.Metadata {
		if parsed.Metadata[k] != v {
			t.Errorf("Metadata[%s] = %q, want %q", k, parsed.Metadata[k], v)
		}
	}
	if _, ok := parsed.Metadata["idempotency_key"]; !ok {
		t.Error("expected idempotency_key in metadata")
	}
	if parsed.Body != original.Body {
		t.Errorf("Body = %q, want %q", parsed.Body, original.Body)
	}
}

func TestDMailMarshal_EmptyBody(t *testing.T) {
	// given — DMail with no body
	dm := domain.DMail{
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
	if !strings.Contains(s, "---\n") {
		t.Errorf("marshaled output missing --- delimiter")
	}

	// round-trip should produce empty body
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	if parsed.Body != "" {
		t.Errorf("Body = %q, want empty", parsed.Body)
	}
}

// === FormatDMailForPrompt Tests ===

func TestFormatDMailForPrompt_EmptySlice(t *testing.T) {
	// given
	var dmails []domain.DMail

	// when
	result := domain.FormatDMailForPrompt(dmails)

	// then
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFormatDMailForPrompt_SingleDMail(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{
			Name:        "spec-my-42",
			Kind:        "specification",
			Description: "Issue MY-42 implementation spec",
			Issues:      []string{"MY-42"},
			Body:        "# DoD\n- Token bucket algorithm\n",
		},
	}

	// when
	result := domain.FormatDMailForPrompt(dmails)

	// then
	if !strings.Contains(result, "spec-my-42") {
		t.Errorf("should contain name, got %q", result)
	}
	if !strings.Contains(result, "specification") {
		t.Errorf("should contain kind, got %q", result)
	}
	if !strings.Contains(result, "Issue MY-42 implementation spec") {
		t.Errorf("should contain description, got %q", result)
	}
	if !strings.Contains(result, "Token bucket algorithm") {
		t.Errorf("should contain body content, got %q", result)
	}
}

func TestFormatDMailForPrompt_MultipleDMails(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{Name: "spec-my-10", Kind: "specification", Description: "First"},
		{Name: "feedback-d-071", Kind: "feedback", Description: "Second", Severity: "medium"},
	}

	// when
	result := domain.FormatDMailForPrompt(dmails)

	// then
	if !strings.Contains(result, "spec-my-10") {
		t.Errorf("should contain first d-mail name, got %q", result)
	}
	if !strings.Contains(result, "feedback-d-071") {
		t.Errorf("should contain second d-mail name, got %q", result)
	}
	if !strings.Contains(result, "medium") {
		t.Errorf("should contain severity, got %q", result)
	}
}

func TestFormatDMailForPrompt_BodylessDMail(t *testing.T) {
	// given — d-mail with no body (frontmatter only)
	dmails := []domain.DMail{
		{Name: "report-my-99", Kind: "report", Description: "Minimal report"},
	}

	// when
	result := domain.FormatDMailForPrompt(dmails)

	// then — should still contain name and description
	if !strings.Contains(result, "report-my-99") {
		t.Errorf("should contain name, got %q", result)
	}
	if !strings.Contains(result, "Minimal report") {
		t.Errorf("should contain description, got %q", result)
	}
}

// === NewReportDMail Tests ===

func TestNewReportDMail_BasicFields(t *testing.T) {
	// given
	report := &domain.ExpeditionReport{
		Expedition:  3,
		IssueID:     "MY-42",
		IssueTitle:  "Add rate limiting",
		MissionType: "implement",
		PRUrl:       "https://github.com/org/repo/pull/123",
		Status:      "success",
		Reason:      "Implemented token bucket algorithm",
	}

	// when
	dm := domain.NewReportDMail(report)

	// then
	if dm.Kind != "report" {
		t.Errorf("Kind = %q, want %q", dm.Kind, "report")
	}
	if dm.Name != "report-my-42" {
		t.Errorf("Name = %q, want %q", dm.Name, "report-my-42")
	}
	if len(dm.Issues) != 1 || dm.Issues[0] != "MY-42" {
		t.Errorf("Issues = %v, want [MY-42]", dm.Issues)
	}
	if !strings.Contains(dm.Body, "https://github.com/org/repo/pull/123") {
		t.Errorf("Body should contain PR URL, got %q", dm.Body)
	}
	if !strings.Contains(dm.Body, "Implemented token bucket algorithm") {
		t.Errorf("Body should contain reason, got %q", dm.Body)
	}
}

func TestNewReportDMail_NameNormalization(t *testing.T) {
	// given — issue ID with uppercase
	report := &domain.ExpeditionReport{
		IssueID:     "MY-100",
		IssueTitle:  "Some issue",
		MissionType: "fix",
		Status:      "success",
	}

	// when
	dm := domain.NewReportDMail(report)

	// then — name should be lowercase
	if dm.Name != "report-my-100" {
		t.Errorf("Name = %q, want %q", dm.Name, "report-my-100")
	}
}

// === Path Function Tests ===

func TestInboxDir(t *testing.T) {
	// given
	continent := "/tmp/myrepo"

	// when
	got := domain.InboxDir(continent)

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
	got := domain.OutboxDir(continent)

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
	got := domain.ArchiveDir(continent)

	// then
	want := filepath.Join("/tmp/myrepo", ".expedition", "archive")
	if got != want {
		t.Errorf("ArchiveDir() = %q, want %q", got, want)
	}
}

// === ParseDMail Edge Cases ===

func TestParseDMail_BodyContainsFrontmatterDelimiter(t *testing.T) {
	// given — body contains "---" which could confuse a naive parser
	input := []byte(`---
name: "tricky-body"
kind: specification
description: "Body has --- inside"
---

# Section

---

More content after horizontal rule.
`)

	// when
	dm, err := domain.ParseDMail(input)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dm.Name != "tricky-body" {
		t.Errorf("Name = %q, want %q", dm.Name, "tricky-body")
	}
	if !strings.Contains(dm.Body, "---") {
		t.Errorf("Body should contain --- horizontal rule, got %q", dm.Body)
	}
	if !strings.Contains(dm.Body, "More content after horizontal rule.") {
		t.Errorf("Body should contain content after ---, got %q", dm.Body)
	}
}

func TestParseDMail_NoTrailingNewlineAfterClosing(t *testing.T) {
	// given — closing --- at EOF with no trailing newline
	input := []byte("---\nname: eof-test\nkind: report\ndescription: no trailing newline\n---")

	// when
	dm, err := domain.ParseDMail(input)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dm.Name != "eof-test" {
		t.Errorf("Name = %q, want %q", dm.Name, "eof-test")
	}
	if dm.Body != "" {
		t.Errorf("Body = %q, want empty", dm.Body)
	}
}

func TestParseDMail_MultipleIssues(t *testing.T) {
	// given — d-mail referencing multiple issues
	input := []byte(`---
name: "multi-issue"
kind: feedback
description: "Affects multiple issues"
issues:
  - MY-10
  - MY-20
  - MY-30
---
`)

	// when
	dm, err := domain.ParseDMail(input)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dm.Issues) != 3 {
		t.Fatalf("Issues length = %d, want 3", len(dm.Issues))
	}
	want := []string{"MY-10", "MY-20", "MY-30"}
	for i, w := range want {
		if dm.Issues[i] != w {
			t.Errorf("Issues[%d] = %q, want %q", i, dm.Issues[i], w)
		}
	}
}

func TestParseDMail_EmptyFrontmatter(t *testing.T) {
	// given — valid delimiters with empty YAML between them (needs newline between)
	input := []byte("---\n\n---\n")

	// when
	dm, err := domain.ParseDMail(input)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dm.Name != "" {
		t.Errorf("Name = %q, want empty", dm.Name)
	}
	if dm.Kind != "" {
		t.Errorf("Kind = %q, want empty", dm.Kind)
	}
}

// === ScanInbox Edge Cases ===

func TestScanInbox_ErrorOnMalformedFile(t *testing.T) {
	// given — inbox contains a .md file with invalid frontmatter
	continent := t.TempDir()
	inboxDir := domain.InboxDir(continent)
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(inboxDir, "bad.md"), []byte("no frontmatter here"), 0644)

	// when
	_, err := ScanInbox(continent)

	// then — error should propagate (not silently skip)
	if err == nil {
		t.Fatal("expected error for malformed d-mail file, got nil")
	}
	if !strings.Contains(err.Error(), "bad.md") {
		t.Errorf("error should mention filename, got %q", err.Error())
	}
}

func TestScanInbox_SkipsSubdirectories(t *testing.T) {
	// given — inbox contains a subdirectory and one valid file
	continent := t.TempDir()
	inboxDir := domain.InboxDir(continent)
	os.MkdirAll(filepath.Join(inboxDir, "subdir"), 0755)

	dm := domain.DMail{Name: "valid-file", Kind: "report", Description: "Should be found"}
	data, _ := dm.Marshal()
	os.WriteFile(filepath.Join(inboxDir, "valid-file.md"), data, 0644)

	// when
	results, err := ScanInbox(continent)

	// then
	if err != nil {
		t.Fatalf("ScanInbox error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (subdir should be skipped)", len(results))
	}
	if results[0].Name != "valid-file" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "valid-file")
	}
}

func TestScanInbox_SortedByFilename(t *testing.T) {
	// given — three files that sort alphabetically: a, b, c
	continent := t.TempDir()
	inboxDir := domain.InboxDir(continent)
	os.MkdirAll(inboxDir, 0755)

	for _, name := range []string{"charlie", "alpha", "bravo"} {
		dm := domain.DMail{Name: name, Kind: "report", Description: name}
		data, _ := dm.Marshal()
		os.WriteFile(filepath.Join(inboxDir, name+".md"), data, 0644)
	}

	// when
	results, err := ScanInbox(continent)

	// then — sorted by filename (alpha.md < bravo.md < charlie.md)
	if err != nil {
		t.Fatalf("ScanInbox error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	wantOrder := []string{"alpha", "bravo", "charlie"}
	for i, w := range wantOrder {
		if results[i].Name != w {
			t.Errorf("results[%d].Name = %q, want %q", i, results[i].Name, w)
		}
	}
}

// === SendDMail Tests ===

func TestSendDMail_WritesToOutboxAndArchive(t *testing.T) {
	// given
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "spec-my-42",
		Kind:        "specification",
		Description: "Test sending d-mail",
		Body:        "# Hello\n",
	}

	// when
	err := SendDMail(store, dm, nil)

	// then
	if err != nil {
		t.Fatalf("SendDMail error: %v", err)
	}

	outboxPath := filepath.Join(domain.OutboxDir(continent), "spec-my-42.md")
	archivePath := filepath.Join(domain.ArchiveDir(continent), "spec-my-42.md")

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
	parsed, err := domain.ParseDMail(outboxData)
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
	// given — clean temp dir; OutboxStore auto-creates dirs
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "report-create-dirs",
		Kind:        "report",
		Description: "Dirs should be auto-created",
	}

	// when
	err := SendDMail(store, dm, nil)

	// then
	if err != nil {
		t.Fatalf("SendDMail error: %v", err)
	}

	// Both files must exist
	outboxPath := filepath.Join(domain.OutboxDir(continent), "report-create-dirs.md")
	archivePath := filepath.Join(domain.ArchiveDir(continent), "report-create-dirs.md")

	if _, err := os.Stat(outboxPath); err != nil {
		t.Errorf("outbox file not found: %v", err)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archive file not found: %v", err)
	}
}

func TestSendDMail_WritesArchiveAndOutbox(t *testing.T) {
	// given — with OutboxStore, both archive and outbox are written in Flush()
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "report-order-test",
		Kind:        "report",
		Description: "Verify both archive and outbox are written",
	}

	// when
	err := SendDMail(store, dm, nil)

	// then
	if err != nil {
		t.Fatalf("SendDMail error: %v", err)
	}

	// Both files must exist after Flush
	archivePath := filepath.Join(domain.ArchiveDir(continent), "report-order-test.md")
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archive file should exist: %v", err)
	}
	outboxPath := filepath.Join(domain.OutboxDir(continent), "report-order-test.md")
	if _, err := os.Stat(outboxPath); err != nil {
		t.Errorf("outbox file should exist: %v", err)
	}
}

// === ScanInbox Tests ===

func TestScanInbox_ReadsAllMdFiles(t *testing.T) {
	// given — two d-mails in inbox
	continent := t.TempDir()
	inboxDir := domain.InboxDir(continent)
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatal(err)
	}

	dm1 := domain.DMail{Name: "alpha", Kind: "report", Description: "First"}
	dm2 := domain.DMail{Name: "beta", Kind: "specification", Description: "Second", Body: "Details\n"}

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
	inboxDir := domain.InboxDir(continent)
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
	inboxDir := domain.InboxDir(continent)
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatal(err)
	}

	dm := domain.DMail{Name: "valid", Kind: "report", Description: "Valid d-mail"}
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
	inboxDir := domain.InboxDir(continent)
	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatal(err)
	}

	dm := domain.DMail{Name: "move-me", Kind: "report", Description: "To be archived"}
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("setup: Marshal: %v", err)
	}
	inboxPath := filepath.Join(inboxDir, "move-me.md")
	if err := os.WriteFile(inboxPath, data, 0644); err != nil {
		t.Fatalf("setup: write move-me.md: %v", err)
	}

	// when
	err = ArchiveInboxDMail(continent, "move-me",  nil)

	// then
	if err != nil {
		t.Fatalf("ArchiveInboxDMail error: %v", err)
	}

	// Gone from inbox
	if _, err := os.Stat(inboxPath); !os.IsNotExist(err) {
		t.Errorf("inbox file still exists after archive")
	}

	// Present in archive
	archivePath := filepath.Join(domain.ArchiveDir(continent), "move-me.md")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("archive file not found: %v", err)
	}

	// Content is preserved
	parsed, err := domain.ParseDMail(archiveData)
	if err != nil {
		t.Fatalf("ParseDMail on archived file: %v", err)
	}
	if parsed.Name != "move-me" {
		t.Errorf("parsed Name = %q, want %q", parsed.Name, "move-me")
	}
}

func TestArchiveInboxDMail_SourceNotFound_NotInArchive(t *testing.T) {
	// given — no file in inbox AND no file in archive (wrong name)
	continent := t.TempDir()
	os.MkdirAll(domain.InboxDir(continent), 0755)

	// when
	err := ArchiveInboxDMail(continent, "nonexistent",  nil)

	// then — error: source missing and not in archive means wrong name
	if err == nil {
		t.Fatal("expected error for missing source not in archive, got nil")
	}
	if !strings.Contains(err.Error(), "not in archive") {
		t.Fatalf("expected 'not in archive' error, got: %v", err)
	}
}

func TestArchiveInboxDMail_SourceNotFound_AlreadyInArchive(t *testing.T) {
	// given — file already in archive (previously archived by another worker)
	continent := t.TempDir()
	os.MkdirAll(domain.InboxDir(continent), 0755)
	arcDir := domain.ArchiveDir(continent)
	os.MkdirAll(arcDir, 0755)
	dm := domain.DMail{Name: "already-moved", Kind: "report", Description: "already archived"}
	data, _ := dm.Marshal()
	os.WriteFile(filepath.Join(arcDir, "already-moved.md"), data, 0644)

	// when
	err := ArchiveInboxDMail(continent, "already-moved",  nil)

	// then — idempotent: dst exists proves another worker archived it
	if err != nil {
		t.Fatalf("expected nil for already-archived d-mail, got: %v", err)
	}
}

func TestArchiveInboxDMail_ConcurrentIdempotent(t *testing.T) {
	// given — one file in inbox, two goroutines try to archive it
	continent := t.TempDir()
	inboxDir := domain.InboxDir(continent)
	os.MkdirAll(inboxDir, 0755)

	dm := domain.DMail{Name: "race-me", Kind: "report", Description: "concurrent test"}
	data, _ := dm.Marshal()
	os.WriteFile(filepath.Join(inboxDir, "race-me.md"), data, 0644)

	// when — two goroutines race to archive
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			errs <- ArchiveInboxDMail(continent, "race-me",  nil)
		}()
	}

	// then — both should return nil (one renames, one gets already-gone)
	for range 2 {
		if err := <-errs; err != nil {
			t.Errorf("concurrent archive should be idempotent, got: %v", err)
		}
	}
}

func TestArchiveInboxDMail_CreatesArchiveDir(t *testing.T) {
	// given — archive/ does not exist yet
	continent := t.TempDir()
	inboxDir := domain.InboxDir(continent)
	os.MkdirAll(inboxDir, 0755)

	dm := domain.DMail{Name: "auto-dir", Kind: "report", Description: "Archive dir auto-created"}
	data, _ := dm.Marshal()
	os.WriteFile(filepath.Join(inboxDir, "auto-dir.md"), data, 0644)

	// when
	err := ArchiveInboxDMail(continent, "auto-dir",  nil)

	// then
	if err != nil {
		t.Fatalf("ArchiveInboxDMail error: %v", err)
	}
	archivePath := filepath.Join(domain.ArchiveDir(continent), "auto-dir.md")
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archive file not found after auto-dir creation: %v", err)
	}
}

// === SendDMail Edge Cases ===

func TestSendDMail_ArchiveDirFailure_NoOutbox(t *testing.T) {
	// given — archive dir is unwritable so atomicWrite in Flush fails
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)

	// Make archive dir unwritable after store creation
	archiveDir := domain.ArchiveDir(continent)
	os.Chmod(archiveDir, 0555)
	t.Cleanup(func() { os.Chmod(archiveDir, 0755) })

	dm := domain.DMail{Name: "fail-early", Kind: "report", Description: "Should fail at flush"}

	// when
	err := SendDMail(store, dm, nil)

	// then — error at flush stage (atomicWrite to archive fails)
	if err == nil {
		t.Fatal("expected error when archive is unwritable")
	}

	// Outbox file must NOT exist (flush failed)
	outboxPath := filepath.Join(domain.OutboxDir(continent), "fail-early.md")
	if _, statErr := os.Stat(outboxPath); statErr == nil {
		t.Error("outbox file should not exist when flush failed")
	}
}

func TestSendDMail_ContentMatchesAfterParse(t *testing.T) {
	// given — d-mail with all fields including body and metadata
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "full-content",
		Kind:        "feedback",
		Description: "Complete content verification",
		Issues:      []string{"MY-1", "MY-2"},
		Severity:    "high",
		Metadata:    map[string]string{"source": "sightjack"},
		Body:        "# Analysis\n\nDrift detected in module X.\n",
	}

	// when
	err := SendDMail(store, dm, nil)

	// then
	if err != nil {
		t.Fatalf("SendDMail error: %v", err)
	}

	// Verify both locations are parseable and match
	for _, dir := range []string{domain.ArchiveDir(continent), domain.OutboxDir(continent)} {
		data, err := os.ReadFile(filepath.Join(dir, "full-content.md"))
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		parsed, err := domain.ParseDMail(data)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		if parsed.Name != dm.Name {
			t.Errorf("%s: Name = %q, want %q", dir, parsed.Name, dm.Name)
		}
		if parsed.Kind != dm.Kind {
			t.Errorf("%s: Kind = %q, want %q", dir, parsed.Kind, dm.Kind)
		}
		if parsed.Severity != dm.Severity {
			t.Errorf("%s: Severity = %q, want %q", dir, parsed.Severity, dm.Severity)
		}
		if len(parsed.Issues) != 2 {
			t.Errorf("%s: Issues = %v, want 2 elements", dir, parsed.Issues)
		}
		if parsed.Metadata["source"] != "sightjack" {
			t.Errorf("%s: Metadata[source] = %q, want sightjack", dir, parsed.Metadata["source"])
		}
		if !strings.Contains(parsed.Body, "Drift detected") {
			t.Errorf("%s: Body should contain 'Drift detected', got %q", dir, parsed.Body)
		}
	}
}

// === NewReportDMail Edge Cases ===

func TestNewReportDMail_NoPRUrl(t *testing.T) {
	// given — report with empty PRUrl
	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "MY-50",
		IssueTitle:  "CLI tool",
		MissionType: "implement",
		Status:      "success",
		PRUrl:       "",
		Reason:      "Implemented successfully",
	}

	// when
	dm := domain.NewReportDMail(report)

	// then — body should not contain PR line
	if strings.Contains(dm.Body, "**PR:**") {
		t.Errorf("Body should not contain PR line when PRUrl is empty, got %q", dm.Body)
	}
}

func TestNewReportDMail_PRUrlNone(t *testing.T) {
	// given — report with PRUrl = "none"
	report := &domain.ExpeditionReport{
		Expedition:  2,
		IssueID:     "MY-51",
		IssueTitle:  "Verify styling",
		MissionType: "verify",
		Status:      "success",
		PRUrl:       "none",
	}

	// when
	dm := domain.NewReportDMail(report)

	// then — "none" is also excluded
	if strings.Contains(dm.Body, "**PR:**") {
		t.Errorf("Body should not contain PR line when PRUrl is 'none', got %q", dm.Body)
	}
}

func TestNewReportDMail_NoReason(t *testing.T) {
	// given — report with empty Reason
	report := &domain.ExpeditionReport{
		Expedition:  3,
		IssueID:     "MY-52",
		IssueTitle:  "Fix bug",
		MissionType: "fix",
		Status:      "success",
		Reason:      "",
	}

	// when
	dm := domain.NewReportDMail(report)

	// then — body should not contain Summary section
	if strings.Contains(dm.Body, "## Summary") {
		t.Errorf("Body should not contain Summary section when Reason is empty, got %q", dm.Body)
	}
}

func TestNewReportDMail_MarshalRoundTrip(t *testing.T) {
	// given — create a report d-mail and marshal it
	report := &domain.ExpeditionReport{
		Expedition:  5,
		IssueID:     "MY-77",
		IssueTitle:  "Add caching",
		MissionType: "implement",
		PRUrl:       "https://github.com/org/repo/pull/42",
		Status:      "success",
		Reason:      "Added Redis caching layer",
	}

	dm := domain.NewReportDMail(report)

	// when — marshal then parse
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}

	// then
	if parsed.Name != "report-my-77" {
		t.Errorf("Name = %q, want %q", parsed.Name, "report-my-77")
	}
	if parsed.Kind != "report" {
		t.Errorf("Kind = %q, want %q", parsed.Kind, "report")
	}
	if len(parsed.Issues) != 1 || parsed.Issues[0] != "MY-77" {
		t.Errorf("Issues = %v, want [MY-77]", parsed.Issues)
	}
	if !strings.Contains(parsed.Body, "https://github.com/org/repo/pull/42") {
		t.Errorf("Body should contain PR URL after roundtrip")
	}
	if !strings.Contains(parsed.Body, "Added Redis caching layer") {
		t.Errorf("Body should contain reason after roundtrip")
	}
	if parsed.SchemaVersion != domain.DMailSchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q after roundtrip", parsed.SchemaVersion, domain.DMailSchemaVersion)
	}
}

// === SchemaVersion Tests ===

func TestParseDMail_SchemaVersion(t *testing.T) {
	// given — d-mail with dmail-schema-version in frontmatter
	input := []byte(`---
name: "versioned-msg"
kind: specification
description: "Has schema version"
dmail-schema-version: "1"
---

Body content.
`)

	// when
	dm, err := domain.ParseDMail(input)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dm.SchemaVersion != domain.DMailSchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", dm.SchemaVersion, domain.DMailSchemaVersion)
	}
}

func TestDMailMarshal_SchemaVersionRoundTrip(t *testing.T) {
	// given — DMail with SchemaVersion set
	original := domain.DMail{
		Name:          "schema-v1",
		Kind:          "report",
		Description:   "Round-trip schema version",
		SchemaVersion: domain.DMailSchemaVersion,
		Body:          "Content.\n",
	}

	// when — marshal then parse
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}

	// then
	if parsed.SchemaVersion != domain.DMailSchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", parsed.SchemaVersion, domain.DMailSchemaVersion)
	}
}

func TestNewReportDMail_SetsSchemaVersion(t *testing.T) {
	// given
	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "MY-99",
		IssueTitle:  "Test issue",
		MissionType: "implement",
		Status:      "success",
	}

	// when
	dm := domain.NewReportDMail(report)

	// then
	if dm.SchemaVersion != domain.DMailSchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", dm.SchemaVersion, domain.DMailSchemaVersion)
	}
}

func TestSendDMail_StampsSchemaVersion(t *testing.T) {
	// given — DMail without SchemaVersion set
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "no-version",
		Kind:        "report",
		Description: "Missing schema version should be stamped",
	}

	// when
	err := SendDMail(store, dm, nil)

	// then
	if err != nil {
		t.Fatalf("SendDMail error: %v", err)
	}

	// Read back and verify version was stamped
	data, err := os.ReadFile(filepath.Join(domain.OutboxDir(continent), "no-version.md"))
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("parse outbox: %v", err)
	}
	if parsed.SchemaVersion != domain.DMailSchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q (should be stamped by SendDMail)", parsed.SchemaVersion, domain.DMailSchemaVersion)
	}
}

// === Marshal Edge Cases ===

func TestDMailMarshal_BodyWithoutTrailingNewline(t *testing.T) {
	// given — body with no trailing newline
	dm := domain.DMail{
		Name:        "no-trailing-nl",
		Kind:        "report",
		Description: "Body without trailing newline",
		Body:        "Content without newline at end",
	}

	// when
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// then — marshaled output should still end with newline
	s := string(data)
	if !strings.HasSuffix(s, "\n") {
		t.Errorf("marshaled output should end with newline, got %q", s[len(s)-20:])
	}

	// round-trip should have trailing newline in body
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	if !strings.HasSuffix(parsed.Body, "\n") {
		t.Errorf("parsed Body should end with newline, got %q", parsed.Body)
	}
}

func TestDMailMarshal_UnicodeContent(t *testing.T) {
	// given — d-mail with Japanese content
	dm := domain.DMail{
		Name:        "unicode-test",
		Kind:        "specification",
		Description: "日本語の説明文",
		Body:        "# 実装計画\n\n- ステップ1: 設計\n- ステップ2: テスト\n",
	}

	// when — marshal then parse
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}

	// then
	if parsed.Description != "日本語の説明文" {
		t.Errorf("Description = %q, want %q", parsed.Description, "日本語の説明文")
	}
	if !strings.Contains(parsed.Body, "実装計画") {
		t.Errorf("Body should contain Japanese content, got %q", parsed.Body)
	}
}

// === D-Mail Lifecycle Integration Test ===
//
// Exercises the full lifecycle on a real filesystem:
//   External writes to inbox → ScanInbox → FormatDMailForPrompt →
//   (expedition success) → NewReportDMail → SendDMail →
//   ArchiveInboxDMail → final state verification

func TestDMailLifecycle_FullFlow(t *testing.T) {
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	inboxDir := domain.InboxDir(continent)
	outboxDir := domain.OutboxDir(continent)
	archiveDir := domain.ArchiveDir(continent)

	// ── Phase 1: External tool writes specification and feedback to inbox ──

	spec := domain.DMail{
		Name:        "spec-my-42",
		Kind:        "specification",
		Description: "Implement rate limiting for API",
		Issues:      []string{"MY-42"},
		Severity:    "medium",
		Body:        "# Definition of Done\n\n- Token bucket algorithm\n- Per-key rate limiting\n",
	}
	feedback := domain.DMail{
		Name:        "feedback-d-071",
		Kind:        "feedback",
		Description: "Architecture drift in auth module",
		Severity:    "high",
		Body:        "## Findings\n\nSession handling does not match design doc.\n",
	}

	for _, dm := range []domain.DMail{spec, feedback} {
		data, err := dm.Marshal()
		if err != nil {
			t.Fatalf("setup: Marshal %s: %v", dm.Name, err)
		}
		if err := os.WriteFile(filepath.Join(inboxDir, dm.Name+".md"), data, 0644); err != nil {
			t.Fatalf("setup: write %s: %v", dm.Name, err)
		}
	}

	// ── Phase 2: Paintress scans inbox (expedition startup) ──

	scanned, err := ScanInbox(continent)
	if err != nil {
		t.Fatalf("ScanInbox: %v", err)
	}
	if len(scanned) != 2 {
		t.Fatalf("ScanInbox returned %d d-mails, want 2", len(scanned))
	}
	// Sorted by filename: feedback-d-071.md < spec-my-42.md
	if scanned[0].Name != "feedback-d-071" {
		t.Errorf("scanned[0].Name = %q, want feedback-d-071", scanned[0].Name)
	}
	if scanned[1].Name != "spec-my-42" {
		t.Errorf("scanned[1].Name = %q, want spec-my-42", scanned[1].Name)
	}

	// ── Phase 3: Format for prompt injection ──

	promptSection := domain.FormatDMailForPrompt(scanned)
	if promptSection == "" {
		t.Fatal("FormatDMailForPrompt returned empty string")
	}
	// Must contain both d-mails
	for _, name := range []string{"spec-my-42", "feedback-d-071"} {
		if !strings.Contains(promptSection, name) {
			t.Errorf("prompt section should contain %q", name)
		}
	}
	// Must contain severity
	if !strings.Contains(promptSection, "high") {
		t.Error("prompt section should contain severity 'high'")
	}
	// Must contain body content
	if !strings.Contains(promptSection, "Token bucket algorithm") {
		t.Error("prompt section should contain spec body")
	}

	// ── Phase 4: Expedition succeeds — create and send report ──

	report := &domain.ExpeditionReport{
		Expedition:  7,
		IssueID:     "MY-42",
		IssueTitle:  "Add rate limiting",
		MissionType: "implement",
		PRUrl:       "https://github.com/org/repo/pull/99",
		Status:      "success",
		Reason:      "Implemented token bucket with Redis backend",
	}
	reportDMail := domain.NewReportDMail(report)

	if reportDMail.Name != "report-my-42" {
		t.Errorf("report Name = %q, want report-my-42", reportDMail.Name)
	}
	if reportDMail.Kind != "report" {
		t.Errorf("report Kind = %q, want report", reportDMail.Kind)
	}

	// Send report via outbox store (Stage → Flush)
	if err := SendDMail(store, reportDMail,  nil); err != nil {
		t.Fatalf("SendDMail: %v", err)
	}

	// Verify report exists in both outbox and archive
	for _, dir := range []string{outboxDir, archiveDir} {
		path := filepath.Join(dir, "report-my-42.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("report not found in %s: %v", dir, err)
		}
		parsed, err := domain.ParseDMail(data)
		if err != nil {
			t.Fatalf("report in %s not parseable: %v", dir, err)
		}
		if parsed.Name != "report-my-42" {
			t.Errorf("%s: parsed.Name = %q, want report-my-42", dir, parsed.Name)
		}
	}

	// ── Phase 5: Archive inbox d-mails (post-expedition) ──

	for _, dm := range scanned {
		if err := ArchiveInboxDMail(continent, dm.Name,  nil); err != nil {
			t.Fatalf("ArchiveInboxDMail(%s): %v", dm.Name, err)
		}
	}

	// ── Phase 6: Final state verification ──

	// inbox/ should be empty
	inboxEntries, err := os.ReadDir(inboxDir)
	if err != nil {
		t.Fatalf("ReadDir inbox: %v", err)
	}
	if len(inboxEntries) != 0 {
		names := make([]string, len(inboxEntries))
		for i, e := range inboxEntries {
			names[i] = e.Name()
		}
		t.Errorf("inbox should be empty after archiving, still has: %v", names)
	}

	// outbox/ should have exactly the report
	outboxEntries, err := os.ReadDir(outboxDir)
	if err != nil {
		t.Fatalf("ReadDir outbox: %v", err)
	}
	if len(outboxEntries) != 1 {
		t.Errorf("outbox should have 1 file (report), got %d", len(outboxEntries))
	}
	if outboxEntries[0].Name() != "report-my-42.md" {
		t.Errorf("outbox file = %q, want report-my-42.md", outboxEntries[0].Name())
	}

	// archive/ should have 3 files: spec + feedback + report
	archiveEntries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("ReadDir archive: %v", err)
	}
	if len(archiveEntries) != 3 {
		names := make([]string, len(archiveEntries))
		for i, e := range archiveEntries {
			names[i] = e.Name()
		}
		t.Fatalf("archive should have 3 files, got %d: %v", len(archiveEntries), names)
	}

	// Verify each archived file is parseable and content-correct
	wantArchived := map[string]string{
		"feedback-d-071.md": "feedback",
		"report-my-42.md":   "report",
		"spec-my-42.md":     "specification",
	}
	for _, entry := range archiveEntries {
		expectedKind, ok := wantArchived[entry.Name()]
		if !ok {
			t.Errorf("unexpected file in archive: %s", entry.Name())
			continue
		}
		data, err := os.ReadFile(filepath.Join(archiveDir, entry.Name()))
		if err != nil {
			t.Errorf("read archived %s: %v", entry.Name(), err)
			continue
		}
		parsed, err := domain.ParseDMail(data)
		if err != nil {
			t.Errorf("parse archived %s: %v", entry.Name(), err)
			continue
		}
		if parsed.Kind != expectedKind {
			t.Errorf("archived %s: Kind = %q, want %q", entry.Name(), parsed.Kind, expectedKind)
		}
	}
}

func TestDMailLifecycle_EmptyInbox(t *testing.T) {
	// Full lifecycle with no d-mails — everything should be no-op graceful
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)

	// ── Phase 1: Inbox dir is empty ──

	scanned, err := ScanInbox(continent)
	if err != nil {
		t.Fatalf("ScanInbox: %v", err)
	}
	if len(scanned) != 0 {
		t.Fatalf("expected 0 d-mails, got %d", len(scanned))
	}

	// ── Phase 2: Empty prompt section ──

	promptSection := domain.FormatDMailForPrompt(scanned)
	if promptSection != "" {
		t.Errorf("expected empty prompt section, got %q", promptSection)
	}

	// ── Phase 3: Expedition sends report even with no inbox ──

	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "MY-10",
		IssueTitle:  "Setup project",
		MissionType: "implement",
		Status:      "success",
		Reason:      "Initial setup complete",
	}
	reportDMail := domain.NewReportDMail(report)
	if err := SendDMail(store, reportDMail,  nil); err != nil {
		t.Fatalf("SendDMail: %v", err)
	}

	// outbox and archive should each have the report
	for _, dirFn := range []func(string) string{domain.OutboxDir, domain.ArchiveDir} {
		path := filepath.Join(dirFn(continent), "report-my-10.md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("report not found at %s: %v", path, err)
		}
	}

	// No archiving needed (no inbox d-mails)
}

func TestDMailLifecycle_MultipleExpeditions(t *testing.T) {
	// Simulates 2 consecutive expeditions:
	//   Expedition 1: picks up spec → succeeds → archives
	//   Expedition 2: picks up feedback (arrived between expeditions) → succeeds → archives
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	inboxDir := domain.InboxDir(continent)

	// ── Expedition 1: spec in inbox ──

	spec := domain.DMail{Name: "spec-my-1", Kind: "specification", Description: "First task"}
	data, _ := spec.Marshal()
	os.WriteFile(filepath.Join(inboxDir, "spec-my-1.md"), data, 0644)

	scanned1, err := ScanInbox(continent)
	if err != nil {
		t.Fatalf("Exp1 ScanInbox: %v", err)
	}
	if len(scanned1) != 1 || scanned1[0].Name != "spec-my-1" {
		t.Fatalf("Exp1: unexpected scan result: %v", scanned1)
	}

	// Success → send report + archive inbox
	report1 := domain.NewReportDMail(&domain.ExpeditionReport{
		Expedition: 1, IssueID: "MY-1", IssueTitle: "First", MissionType: "implement", Status: "success",
	})
	if err := SendDMail(store, report1,  nil); err != nil {
		t.Fatalf("Exp1 SendDMail: %v", err)
	}
	if err := ArchiveInboxDMail(continent, "spec-my-1",  nil); err != nil {
		t.Fatalf("Exp1 ArchiveInboxDMail: %v", err)
	}

	// inbox should be empty now
	entries, _ := os.ReadDir(inboxDir)
	if len(entries) != 0 {
		t.Fatalf("Exp1: inbox should be empty, got %d files", len(entries))
	}

	// ── Between expeditions: new feedback arrives ──

	fb := domain.DMail{Name: "feedback-d-001", Kind: "feedback", Description: "Review feedback", Severity: "medium"}
	data, _ = fb.Marshal()
	os.WriteFile(filepath.Join(inboxDir, "feedback-d-001.md"), data, 0644)

	// ── Expedition 2: feedback in inbox ──

	scanned2, err := ScanInbox(continent)
	if err != nil {
		t.Fatalf("Exp2 ScanInbox: %v", err)
	}
	if len(scanned2) != 1 || scanned2[0].Name != "feedback-d-001" {
		t.Fatalf("Exp2: unexpected scan result: %v", scanned2)
	}

	report2 := domain.NewReportDMail(&domain.ExpeditionReport{
		Expedition: 2, IssueID: "MY-2", IssueTitle: "Second", MissionType: "fix", Status: "success",
	})
	if err := SendDMail(store, report2,  nil); err != nil {
		t.Fatalf("Exp2 SendDMail: %v", err)
	}
	if err := ArchiveInboxDMail(continent, "feedback-d-001",  nil); err != nil {
		t.Fatalf("Exp2 ArchiveInboxDMail: %v", err)
	}

	// ── Final state ──

	// inbox: empty
	entries, _ = os.ReadDir(inboxDir)
	if len(entries) != 0 {
		t.Errorf("final: inbox should be empty, got %d files", len(entries))
	}

	// outbox: 2 reports
	outboxEntries, _ := os.ReadDir(domain.OutboxDir(continent))
	if len(outboxEntries) != 2 {
		t.Errorf("final: outbox should have 2 reports, got %d", len(outboxEntries))
	}

	// archive: 4 files (spec + report1 + feedback + report2)
	archiveEntries, _ := os.ReadDir(domain.ArchiveDir(continent))
	if len(archiveEntries) != 4 {
		names := make([]string, len(archiveEntries))
		for i, e := range archiveEntries {
			names[i] = e.Name()
		}
		t.Errorf("final: archive should have 4 files, got %d: %v", len(archiveEntries), names)
	}
}

// === BuildFollowUpPrompt Tests ===

func TestBuildFollowUpPrompt_SingleDMail(t *testing.T) {
	dmails := []domain.DMail{
		{Name: "spec-my-42", Kind: "specification", Description: "Rate limiting spec", Issues: []string{"MY-42"}, Body: "# DoD\n- Token bucket\n"},
	}

	prompt := domain.BuildFollowUpPrompt(dmails)

	if !strings.Contains(prompt, "spec-my-42") {
		t.Error("prompt should contain d-mail name")
	}
	if !strings.Contains(prompt, "Token bucket") {
		t.Error("prompt should contain d-mail body content")
	}
	if !strings.Contains(prompt, "D-Mail") {
		t.Error("prompt should mention D-Mail")
	}
}

func TestBuildFollowUpPrompt_MultipleDMails(t *testing.T) {
	dmails := []domain.DMail{
		{Name: "spec-my-42", Kind: "specification", Description: "Rate limiting"},
		{Name: "feedback-d-001", Kind: "feedback", Description: "Review feedback", Severity: "medium"},
	}

	prompt := domain.BuildFollowUpPrompt(dmails)

	if !strings.Contains(prompt, "spec-my-42") {
		t.Error("prompt should contain first d-mail")
	}
	if !strings.Contains(prompt, "feedback-d-001") {
		t.Error("prompt should contain second d-mail")
	}
}

func TestBuildFollowUpPrompt_EmptySlice(t *testing.T) {
	prompt := domain.BuildFollowUpPrompt(nil)
	if prompt != "" {
		t.Errorf("empty input should return empty string, got %q", prompt)
	}
}

func TestFilterHighSeverity_NoHighSeverity(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{Name: "report-1", Kind: "report", Severity: ""},
		{Name: "spec-2", Kind: "specification", Severity: "low"},
	}

	// when
	high := domain.FilterHighSeverity(dmails)

	// then
	if len(high) != 0 {
		t.Errorf("expected 0 HIGH severity d-mails, got %d", len(high))
	}
}

func TestFilterHighSeverity_MixedSeverity(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{Name: "report-1", Kind: "report", Severity: ""},
		{Name: "alert-1", Kind: "alert", Severity: "high"},
		{Name: "spec-1", Kind: "specification", Severity: "low"},
		{Name: "alert-2", Kind: "alert", Severity: "high"},
	}

	// when
	high := domain.FilterHighSeverity(dmails)

	// then
	if len(high) != 2 {
		t.Fatalf("expected 2 HIGH severity d-mails, got %d", len(high))
	}
	if high[0].Name != "alert-1" || high[1].Name != "alert-2" {
		t.Errorf("unexpected high d-mails: %v", high)
	}
}

func TestFilterHighSeverity_EmptySlice(t *testing.T) {
	// given
	var dmails []domain.DMail

	// when
	high := domain.FilterHighSeverity(dmails)

	// then
	if len(high) != 0 {
		t.Errorf("expected 0 for nil/empty, got %d", len(high))
	}
}

// === Emitter Error Propagation Tests ===
//
// D-Mail events (dmail.staged, dmail.flushed, dmail.archived) are critical
// domain events in the event sourcing model. Errors must not be silently dropped.

// failingEmitter is a minimal ExpeditionEventEmitter that always returns an error.
type failingEmitter struct {
	err error
}

func (f *failingEmitter) EmitStartExpedition(_, _ int, _ string, _ time.Time) error { return f.err }
func (f *failingEmitter) EmitCompleteExpedition(_ int, _, _, _ string, _ time.Time) error {
	return f.err
}
func (f *failingEmitter) EmitInboxReceived(_, _ string, _ time.Time) error  { return f.err }
func (f *failingEmitter) EmitGommage(_ int, _ time.Time) error              { return f.err }
func (f *failingEmitter) EmitGradientChange(_ int, _ string, _ time.Time) error {
	return f.err
}
func (f *failingEmitter) EmitRetryAttempted(_ string, _ int, _ time.Time) error { return f.err }
func (f *failingEmitter) EmitEscalated(_ string, _ []string, _ time.Time) error { return f.err }
func (f *failingEmitter) EmitDMailStaged(_ string, _ time.Time) error           { return f.err }
func (f *failingEmitter) EmitDMailFlushed(_ int, _ time.Time) error             { return f.err }
func (f *failingEmitter) EmitDMailArchived(_ string, _ time.Time) error         { return f.err }

func TestSendDMail_PropagatesEmitterError(t *testing.T) {
	// given — an outbox store that works, but an emitter that fails
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	outboxStore := testOutboxStore(t, continent)
	emitter := &failingEmitter{err: fmt.Errorf("disk full")}

	dm := domain.DMail{
		Name:        "report-es-fail",
		Kind:        "report",
		Description: "Test that emitter errors propagate",
	}

	// when
	err := SendDMail(outboxStore, dm, emitter)

	// then — error from emitter must be propagated
	if err == nil {
		t.Fatal("expected error from failing emitter, got nil")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("error should contain root cause, got: %s", err.Error())
	}
}

func TestArchiveInboxDMail_PropagatesEmitterError(t *testing.T) {
	// given — a d-mail in inbox, and an emitter that fails
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	emitter := &failingEmitter{err: fmt.Errorf("readonly fs")}

	// Write a d-mail file into inbox for archiving
	dm := domain.DMail{
		Name:        "spec-es-fail",
		Kind:        "specification",
		Description: "Test archive event propagation",
	}
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(domain.InboxDir(continent), "spec-es-fail.md"), data, 0644); err != nil {
		t.Fatalf("write inbox: %v", err)
	}

	// when
	err = ArchiveInboxDMail(continent, "spec-es-fail", emitter)

	// then — error from emitter must be propagated
	if err == nil {
		t.Fatal("expected error from failing emitter, got nil")
	}
	if !strings.Contains(err.Error(), "readonly fs") {
		t.Errorf("error should contain root cause, got: %s", err.Error())
	}
}
