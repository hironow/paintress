package paintress

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
// The name is normalized to lowercase (e.g., "MY-42" → "report-my-42").
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
		SchemaVersion: "1",
		Body:          body.String(),
	}
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

// DMail represents a d-mail message with YAML frontmatter fields and a Markdown body.
// The format uses Jekyll/Hugo-style frontmatter delimiters (---).
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
// The input must start with "---\n", followed by YAML, followed by "\n---\n",
// followed by an optional Markdown body.
func ParseDMail(data []byte) (DMail, error) {
	s := string(data)

	// Must start with opening delimiter
	if !strings.HasPrefix(s, "---\n") {
		return DMail{}, errMissingOpeningDelimiter
	}

	// Find closing delimiter after the opening one
	rest := s[4:] // skip "---\n"
	closingIdx := strings.Index(rest, "\n---\n")
	if closingIdx < 0 {
		// Also accept "\n---" at end of input (no trailing newline after closing delimiter)
		if strings.HasSuffix(rest, "\n---") {
			closingIdx = len(rest) - 4 // len("\n---") == 4
		} else {
			return DMail{}, errMissingClosingDelimiter
		}
	}

	yamlContent := rest[:closingIdx]
	afterClosing := rest[closingIdx+4:] // skip "\n---"

	var dm DMail
	if err := yaml.Unmarshal([]byte(yamlContent), &dm); err != nil {
		return DMail{}, err
	}

	// Trim the leading newline/whitespace between closing --- and body content
	dm.Body = strings.TrimLeft(afterClosing, "\n")

	return dm, nil
}

// Marshal produces the d-mail wire format: "---\n" + YAML + "---\n\n" + Body.
// If the body is empty, only the frontmatter is produced (no extra blank lines).
// A non-empty body is guaranteed to end with a trailing newline.
func (d DMail) Marshal() ([]byte, error) {
	yamlData, err := yaml.Marshal(d)
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
		// Ensure trailing newline
		if !strings.HasSuffix(d.Body, "\n") {
			buf.WriteString("\n")
		}
	}

	return buf.Bytes(), nil
}

// SendDMail writes a d-mail to archive/ first, then outbox/.
// Archive-first ensures the permanent record survives even if the outbox write fails.
// Creates directories if needed. Filename: <d.Name>.md
func SendDMail(continent string, d DMail) error {
	if d.SchemaVersion == "" {
		d.SchemaVersion = "1"
	}
	data, err := d.Marshal()
	if err != nil {
		return fmt.Errorf("dmail: marshal: %w", err)
	}

	filename := d.Name + ".md"

	for _, dir := range []string{ArchiveDir(continent), OutboxDir(continent)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("dmail: mkdir %s: %w", dir, err)
		}
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("dmail: write %s: %w", path, err)
		}
	}

	return nil
}

// ScanInbox reads all .md files in inbox/, parses each as DMail.
// Returns parsed d-mails sorted by filename. Returns empty slice for empty
// or non-existent directory. Skips non-.md files.
func ScanInbox(continent string) ([]DMail, error) {
	dir := InboxDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []DMail{}, nil
		}
		return nil, fmt.Errorf("dmail: read inbox: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var dmails []DMail
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("dmail: read %s: %w", e.Name(), err)
		}
		dm, err := ParseDMail(data)
		if err != nil {
			return nil, fmt.Errorf("dmail: parse %s: %w", e.Name(), err)
		}
		dmails = append(dmails, dm)
	}

	if dmails == nil {
		return []DMail{}, nil
	}
	return dmails, nil
}

// ArchiveInboxDMail moves a d-mail from inbox/ to archive/.
// Uses os.Rename for atomic move. Creates archive dir if needed.
func ArchiveInboxDMail(continent, name string) error {
	filename := name + ".md"
	src := filepath.Join(InboxDir(continent), filename)
	arcDir := ArchiveDir(continent)
	dst := filepath.Join(arcDir, filename)

	if err := os.MkdirAll(arcDir, 0755); err != nil {
		return fmt.Errorf("dmail: mkdir archive: %w", err)
	}

	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("dmail: archive %s: %w", name, err)
	}

	return nil
}
