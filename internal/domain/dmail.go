package domain

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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
// gaugeLevel is the current GradientGauge level and determines the Severity field.
func NewReportDMail(report *ExpeditionReport, gaugeLevel int) DMail {
	name := "pt-report-" + sanitizeDMailKey(report.IssueID) + "_" + DMailUUIDFunc()

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

	dm := DMail{
		Name:          name,
		Kind:          "report",
		Description:   fmt.Sprintf("Expedition #%d completed %s for %s", report.Expedition, report.MissionType, report.IssueID),
		Issues:        []string{report.IssueID},
		Severity:      ReportSeverity(gaugeLevel),
		SchemaVersion: DMailSchemaVersion,
		Body:          body.String(),
	}

	if report.Insight != "" {
		dm.Context = &InsightContext{
			Insights: []InsightSummary{
				{Source: report.IssueID, Summary: report.Insight},
			},
		}
	}

	// Wave-centric mode: attach wave reference for archive projection
	if report.WaveID != "" {
		dm.Wave = &WaveReference{
			ID:   report.WaveID,
			Step: report.StepID,
		}
		// Override name to include wave/step for uniqueness
		if report.StepID != "" {
			dm.Name = "pt-report-" + sanitizeDMailKey(report.WaveID+"-"+report.StepID) + "_" + DMailUUIDFunc()
		} else {
			dm.Name = "pt-report-" + sanitizeDMailKey(report.WaveID) + "_" + DMailUUIDFunc()
		}
	}

	return dm
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
	return filepath.Join(continent, StateDir, "inbox")
}

// OutboxDir returns the path to the d-mail outbox directory.
func OutboxDir(continent string) string {
	return filepath.Join(continent, StateDir, "outbox")
}

// ArchiveDir returns the path to the d-mail archive directory.
func ArchiveDir(continent string) string {
	return filepath.Join(continent, StateDir, "archive")
}

// InsightsDir returns the path to the insights directory.
func InsightsDir(continent string) string {
	return filepath.Join(continent, StateDir, "insights")
}

// RunDir returns the path to the run directory (SQLite, locks, logs).
func RunDir(continent string) string {
	return filepath.Join(continent, StateDir, ".run")
}

// DMailSchemaVersion is the current D-Mail protocol schema version.
const DMailSchemaVersion = "1"

// WaveStepDef defines a single step within a wave specification.
type WaveStepDef struct {
	ID            string   `yaml:"id" json:"id"`
	Title         string   `yaml:"title" json:"title"`
	Description   string   `yaml:"description,omitempty" json:"description,omitempty"`
	Targets       []string `yaml:"targets,omitempty" json:"targets,omitempty"`
	Acceptance    string   `yaml:"acceptance,omitempty" json:"acceptance,omitempty"`
	Prerequisites []string `yaml:"prerequisites,omitempty" json:"prerequisites,omitempty"`
}

// WaveReference links a D-Mail to a wave and optionally a specific step.
// In specification D-Mails: Steps contains the full step definitions.
// In report/feedback D-Mails: Step identifies the specific step.
type WaveReference struct {
	ID    string        `yaml:"id" json:"id"`
	Step  string        `yaml:"step,omitempty" json:"step,omitempty"`
	Steps []WaveStepDef `yaml:"steps,omitempty" json:"steps,omitempty"`
}

// DMail represents a d-mail message with YAML frontmatter fields and a Markdown body.
type DMail struct {
	Name          string            `yaml:"name"`
	Kind          string            `yaml:"kind"`
	Description   string            `yaml:"description"`
	Issues        []string          `yaml:"issues,omitempty"`
	Severity      string            `yaml:"severity,omitempty"`
	Action        string            `yaml:"action,omitempty"`
	Priority      int               `yaml:"priority,omitempty"`
	SchemaVersion string            `yaml:"dmail-schema-version,omitempty"`
	Wave          *WaveReference    `yaml:"wave,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	Context       *InsightContext   `yaml:"context,omitempty" json:"context,omitempty"`
	Body          string            `yaml:"-"`
}

// validActions is the set of valid action values per D-Mail schema v1.
// Strict on send, liberal on receive (Postel's law / S0021).
var validActions = map[string]bool{
	"retry":    true,
	"escalate": true,
	"resolve":  true,
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

// ValidateDMail checks that a DMail conforms to D-Mail schema v1.
func ValidateDMail(d DMail) error {
	if d.SchemaVersion == "" {
		return fmt.Errorf("dmail: dmail-schema-version is required")
	}
	if d.SchemaVersion != DMailSchemaVersion {
		return fmt.Errorf("dmail: unsupported dmail-schema-version: %q (want %q)", d.SchemaVersion, DMailSchemaVersion)
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

// DMailUUIDFunc is the UUID generator for D-Mail filenames. Override in tests.
var DMailUUIDFunc = shortDMailUUID

func shortDMailUUID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x", buf[:4])
}

func sanitizeDMailKey(key string) string {
	var b strings.Builder
	prev := rune(0)
	for _, r := range strings.ToLower(key) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prev = r
		case r == '-', r == ':', r == ' ', r == '_':
			if prev != '-' {
				b.WriteRune('-')
				prev = '-'
			}
		default:
			// skip non-ascii
		}
	}
	return strings.Trim(b.String(), "-")
}
