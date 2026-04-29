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

// DMailKind is the type-safe representation of D-Mail message kinds.
type DMailKind string

const (
	KindSpecification    DMailKind = "specification"
	KindReport           DMailKind = "report"
	KindDesignFeedback   DMailKind = "design-feedback"
	KindImplFeedback     DMailKind = "implementation-feedback"
	KindConvergence      DMailKind = "convergence"
	KindCIResult         DMailKind = "ci-result"
	KindStallEscalation  DMailKind = "stall-escalation"
)

// ValidDMailKinds is the canonical set of allowed D-Mail kinds per schema v1.
var ValidDMailKinds = map[DMailKind]bool{
	KindSpecification:   true,
	KindReport:          true,
	KindDesignFeedback:  true,
	KindImplFeedback:    true,
	KindConvergence:     true,
	KindCIResult:        true,
	KindStallEscalation: true,
}

// IsValidDMailKind returns true if the given kind is in the canonical set.
func IsValidDMailKind(kind DMailKind) bool {
	return ValidDMailKinds[kind]
}

// ErrDMailKindInvalid is returned when a D-Mail kind is not in the canonical set.
var ErrDMailKindInvalid = errors.New("dmail: invalid kind")

// ParseKind parses and validates a D-Mail kind, returning the validated kind or an error.
func ParseKind(kind DMailKind) (DMailKind, error) {
	if !IsValidDMailKind(kind) {
		return "", fmt.Errorf("invalid D-Mail kind %q: %w", kind, ErrDMailKindInvalid)
	}
	return kind, nil
}

// ValidateKind checks that kind is one of the allowed D-Mail kinds.
//
// Deprecated: prefer ParseKind which returns the validated kind.
func ValidateKind(kind DMailKind) error { // nosemgrep: parse-dont-validate.validate-returns-error-only-go -- backward-compat wrapper; ParseKind is the canonical parse function [permanent]
	_, err := ParseKind(kind)
	return err
}

// WaveStepDef defines a single step within a wave specification.
type WaveStepDef struct { // nosemgrep: first-class-collection.raw-slice-field-domain-go,structure.multiple-exported-structs-go -- Targets/Prerequisites are YAML-serialized spec fields (no FCC benefit); D-Mail wire-format DTO family is cohesive YAML/JSON serialization set [permanent]
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
type WaveReference struct { // nosemgrep: first-class-collection.raw-slice-field-domain-go,structure.multiple-exported-structs-go -- Steps is a YAML-serialized spec field (no FCC benefit); D-Mail wire-format DTO family cohesive set; see WaveStepDef [permanent]
	ID    string        `yaml:"id" json:"id"`
	Step  string        `yaml:"step,omitempty" json:"step,omitempty"`
	Steps []WaveStepDef `yaml:"steps,omitempty" json:"steps,omitempty"`
}

// DMail represents a d-mail message with YAML frontmatter fields and a Markdown body.
type DMail struct { // nosemgrep: first-class-collection.raw-slice-field-domain-go -- Issues is a YAML-serialized field; FCC wrapping would break marshaling [permanent]
	Name          string            `yaml:"name"`
	Kind          DMailKind         `yaml:"kind"`
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

// SanitizeDMailKey normalizes a key for use in D-Mail filenames.
func SanitizeDMailKey(key string) string {
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
