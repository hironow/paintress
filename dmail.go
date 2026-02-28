package paintress

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// FormatDMailForPrompt formats d-mails as a human-readable Markdown section
// for injection into expedition prompts. Returns empty string for empty input.
func FormatDMailForPrompt(dmails []DMail) string {
	if len(dmails) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, dm := range dmails {
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
	}
	return buf.String()
}

// NewReportDMail creates a report d-mail from an ExpeditionReport.
func NewReportDMail(report *ExpeditionReport) DMail {
	name := "report-" + strings.ToLower(report.IssueID)

	var body strings.Builder
	fmt.Fprintf(&body, "# Expedition #%d Report: %s\n\n", report.Expedition, report.IssueTitle)
	fmt.Fprintf(&body, "- **Issue:** %s\n", report.IssueID)
	fmt.Fprintf(&body, "- **Mission:** %s\n", report.MissionType)
	fmt.Fprintf(&body, "- **Status:** %s\n", report.Status)
	if report.PRUrl != "" && report.PRUrl != "none" {
		fmt.Fprintf(&body, "- **PR:** %s\n", report.PRUrl)
	}
	if report.Reason != "" {
		fmt.Fprintf(&body, "\n## Summary\n\n%s\n", report.Reason)
	}

	return DMail{
		Name:          name,
		Kind:          "report",
		Description:   fmt.Sprintf("Expedition #%d completed %s for %s", report.Expedition, report.MissionType, report.IssueID),
		Issues:        []string{report.IssueID},
		SchemaVersion: DMailSchemaVersion,
		Body:          body.String(),
	}
}

// BuildFollowUpPrompt builds a follow-up prompt for issue-matched D-Mails
// received mid-expedition. Returns empty string for empty input.
func BuildFollowUpPrompt(dmails []DMail) string {
	if len(dmails) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("The following D-Mail(s) arrived during this expedition and are related to the issue you just worked on.\n")
	buf.WriteString("Review them and take any additional action if needed. If no action is required, briefly acknowledge.\n\n")
	buf.WriteString(FormatDMailForPrompt(dmails))
	return buf.String()
}

// InboxDir returns the path to the d-mail inbox directory.
func InboxDir(continent string) string {
	return filepath.Join(continent, ".expedition", "inbox")
}

// OutboxDir returns the path to the d-mail outbox directory.
func OutboxDir(continent string) string {
	return filepath.Join(continent, ".expedition", "outbox")
}

// ArchiveDir returns the path to the d-mail archive directory.
func ArchiveDir(continent string) string {
	return filepath.Join(continent, ".expedition", "archive")
}

// DMailSchemaVersion is the current D-Mail protocol schema version.
const DMailSchemaVersion = "1"

// DMail represents a d-mail message with YAML frontmatter fields and a Markdown body.
type DMail struct {
	Name          string            `yaml:"name"`
	Kind          string            `yaml:"kind"`
	Description   string            `yaml:"description"`
	Issues        []string          `yaml:"issues,omitempty"`
	Severity      string            `yaml:"severity,omitempty"`
	SchemaVersion string            `yaml:"dmail-schema-version,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	Body          string            `yaml:"-"`
}

var (
	errMissingOpeningDelimiter = errors.New("dmail: missing opening --- delimiter")
	errMissingClosingDelimiter = errors.New("dmail: missing closing --- delimiter")
)

// ParseDMail parses a d-mail from bytes containing YAML frontmatter and optional Markdown body.
func ParseDMail(data []byte) (DMail, error) {
	s := string(data)

	if !strings.HasPrefix(s, "---\n") {
		return DMail{}, errMissingOpeningDelimiter
	}

	rest := s[4:]
	closingIdx := strings.Index(rest, "\n---\n")
	if closingIdx < 0 {
		if strings.HasSuffix(rest, "\n---") {
			closingIdx = len(rest) - 4
		} else {
			return DMail{}, errMissingClosingDelimiter
		}
	}

	yamlContent := rest[:closingIdx]
	afterClosing := rest[closingIdx+4:]

	var dm DMail
	if err := yaml.Unmarshal([]byte(yamlContent), &dm); err != nil {
		return DMail{}, err
	}

	dm.Body = strings.TrimLeft(afterClosing, "\n")

	return dm, nil
}

// DMailIdempotencyKey computes a SHA256 content-based idempotency key from
// the core fields of a DMail (name, kind, description, body).
func DMailIdempotencyKey(d DMail) string {
	h := sha256.New()
	h.Write([]byte(d.Name))
	h.Write([]byte{0})
	h.Write([]byte(d.Kind))
	h.Write([]byte{0})
	h.Write([]byte(d.Description))
	h.Write([]byte{0})
	h.Write([]byte(d.Body))
	return hex.EncodeToString(h.Sum(nil))
}

// Marshal produces the d-mail wire format: "---\n" + YAML + "---\n\n" + Body.
// Automatically injects an idempotency_key into metadata based on content hash.
func (d DMail) Marshal() ([]byte, error) {
	cp := d
	meta := make(map[string]string, len(d.Metadata)+1)
	for k, v := range d.Metadata {
		meta[k] = v
	}
	meta["idempotency_key"] = DMailIdempotencyKey(d)
	cp.Metadata = meta

	yamlData, err := yaml.Marshal(cp)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yamlData)
	buf.WriteString("---\n")

	if d.Body != "" {
		buf.WriteString("\n")
		buf.WriteString(d.Body)
		if !strings.HasSuffix(d.Body, "\n") {
			buf.WriteString("\n")
		}
	}

	return buf.Bytes(), nil
}

// FilterHighSeverity returns only HIGH severity d-mails from the input slice.
func FilterHighSeverity(dmails []DMail) []DMail {
	var high []DMail
	for _, dm := range dmails {
		if dm.Severity == "high" {
			high = append(high, dm)
		}
	}
	return high
}
