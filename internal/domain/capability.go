package domain

import (
	"fmt"
	"strings"
)

// CapabilityViolationType identifies the category of a capability violation.
type CapabilityViolationType string

const (
	// CapabilityViolationNone indicates no capability violation was detected.
	CapabilityViolationNone CapabilityViolationType = "none"
	// CapabilityViolationNetwork indicates a network access capability violation.
	CapabilityViolationNetwork CapabilityViolationType = "network"
	// CapabilityViolationFilesystem indicates a filesystem permission violation.
	CapabilityViolationFilesystem CapabilityViolationType = "filesystem"
	// CapabilityViolationMissingTool indicates a required tool is not installed.
	CapabilityViolationMissingTool CapabilityViolationType = "missing-tool"
	// CapabilityViolationDocker indicates Docker is unavailable.
	CapabilityViolationDocker CapabilityViolationType = "docker"
	// CapabilityViolationAuth indicates an authentication failure.
	CapabilityViolationAuth CapabilityViolationType = "auth"
	// CapabilityViolationResourceLimit indicates a resource limit was hit.
	CapabilityViolationResourceLimit CapabilityViolationType = "resource-limit"
)

// capabilityRule maps signal strings to violation types.
type capabilityRule struct {
	signal    string
	violation CapabilityViolationType
}

// capabilityRules is the ordered list of detection rules.
// More specific rules should appear before more general ones.
var capabilityRules = []capabilityRule{
	// Docker
	{"cannot connect to the docker daemon", CapabilityViolationDocker},
	{"docker.sock", CapabilityViolationDocker},

	// Network
	{"could not resolve host", CapabilityViolationNetwork},
	{"no such host", CapabilityViolationNetwork},
	{"connection refused", CapabilityViolationNetwork},
	{"network unreachable", CapabilityViolationNetwork},

	// Filesystem
	{"permission denied", CapabilityViolationFilesystem},
	{"read-only file system", CapabilityViolationFilesystem},

	// Missing tool
	{"command not found", CapabilityViolationMissingTool},
	{"executable file not found", CapabilityViolationMissingTool},
	{"no such file or directory", CapabilityViolationMissingTool},

	// Auth
	{"please authenticate", CapabilityViolationAuth},
	{"authentication token not found", CapabilityViolationAuth},
	{"authentication required", CapabilityViolationAuth},
	{"unauthorized", CapabilityViolationAuth},

	// Resource limit
	{"signal: killed", CapabilityViolationResourceLimit},
	{"out of memory", CapabilityViolationResourceLimit},
	{"oom", CapabilityViolationResourceLimit},
}

// ClassifyCapabilityViolation detects which capability violation type, if any,
// is present in the given error output. Matching is case-insensitive.
// Returns CapabilityViolationNone when no known signals are found.
func ClassifyCapabilityViolation(output string) CapabilityViolationType {
	lower := strings.ToLower(output)
	for _, rule := range capabilityRules {
		if strings.Contains(lower, rule.signal) {
			return rule.violation
		}
	}
	return CapabilityViolationNone
}

// CapabilityViolation is a detected capability boundary violation from a journal entry.
type CapabilityViolation struct {
	Type    CapabilityViolationType
	Message string
}

// ScanJournalsForCapabilityViolations scans failed journal entries for capability violations.
// Only entries with status "failed" are examined.
func ScanJournalsForCapabilityViolations(entries []JournalEntry) []CapabilityViolation {
	var violations []CapabilityViolation
	for _, entry := range entries {
		if entry.Status != "failed" {
			continue
		}
		violationType := ClassifyCapabilityViolation(entry.Reason)
		if violationType == CapabilityViolationNone {
			continue
		}
		violations = append(violations, CapabilityViolation{
			Type:    violationType,
			Message: entry.Reason,
		})
	}
	return violations
}

// FormatCapabilityViolationsSection renders detected capability violations as a prompt section.
// Returns an empty string when violations is empty.
func FormatCapabilityViolationsSection(violations []CapabilityViolation) string {
	if len(violations) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Capability Boundary Violations\n\n")
	sb.WriteString("The following capability limitations were detected in past expeditions:\n\n")
	for _, v := range violations {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", v.Type, v.Message))
	}
	sb.WriteString("\nAvoid actions that would trigger these limitations.\n")
	return sb.String()
}
