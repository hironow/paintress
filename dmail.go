package paintress

import (
	"bytes"
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

// DMail represents a d-mail message with YAML frontmatter fields and a Markdown body.
// The format uses Jekyll/Hugo-style frontmatter delimiters (---).
type DMail struct {
	Name        string            `yaml:"name"`
	Kind        string            `yaml:"kind"`
	Description string            `yaml:"description"`
	Issues      []string          `yaml:"issues,omitempty"`
	Severity    string            `yaml:"severity,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
	Body        string            `yaml:"-"`
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
