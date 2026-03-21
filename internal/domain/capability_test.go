package domain_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

// TestClassifyCapabilityViolation_KnownViolations verifies all 11 named violation types.
func TestClassifyCapabilityViolation_KnownViolations(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantType domain.CapabilityViolationType
	}{
		{
			name:     "network_access_denied",
			input:    "curl: (6) Could not resolve host: api.example.com",
			wantType: domain.CapabilityViolationNetwork,
		},
		{
			name:     "no_network_connection",
			input:    "dial tcp: no such host",
			wantType: domain.CapabilityViolationNetwork,
		},
		{
			name:     "file_system_permission_denied",
			input:    "open /etc/shadow: permission denied",
			wantType: domain.CapabilityViolationFilesystem,
		},
		{
			name:     "read_only_file_system",
			input:    "read-only file system",
			wantType: domain.CapabilityViolationFilesystem,
		},
		{
			name:     "command_not_found",
			input:    "bash: docker: command not found",
			wantType: domain.CapabilityViolationMissingTool,
		},
		{
			name:     "executable_not_found",
			input:    "exec: \"kubectl\": executable file not found in $PATH",
			wantType: domain.CapabilityViolationMissingTool,
		},
		{
			name:     "docker_unavailable",
			input:    "Cannot connect to the Docker daemon",
			wantType: domain.CapabilityViolationDocker,
		},
		{
			name:     "docker_socket_error",
			input:    "dial unix /var/run/docker.sock: no such file or directory",
			wantType: domain.CapabilityViolationDocker,
		},
		{
			name:     "github_auth_failure",
			input:    "gh: To use GitHub CLI, please authenticate first",
			wantType: domain.CapabilityViolationAuth,
		},
		{
			name:     "auth_token_missing",
			input:    "Error: authentication token not found",
			wantType: domain.CapabilityViolationAuth,
		},
		{
			name:     "resource_limit_oom",
			input:    "signal: killed (OOM)",
			wantType: domain.CapabilityViolationResourceLimit,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := domain.ClassifyCapabilityViolation(tc.input)

			// then
			if got != tc.wantType {
				t.Errorf("ClassifyCapabilityViolation(%q) = %v, want %v", tc.input, got, tc.wantType)
			}
		})
	}
}

// TestClassifyCapabilityViolation_UnknownViolation verifies unknown input returns None.
func TestClassifyCapabilityViolation_UnknownViolation(t *testing.T) {
	// given
	input := "some random error message with no capability signal"

	// when
	got := domain.ClassifyCapabilityViolation(input)

	// then
	if got != domain.CapabilityViolationNone {
		t.Errorf("ClassifyCapabilityViolation = %v, want CapabilityViolationNone", got)
	}
}

// TestClassifyCapabilityViolation_CaseInsensitive verifies case-insensitive matching.
func TestClassifyCapabilityViolation_CaseInsensitive(t *testing.T) {
	// given: uppercase signal
	input := "PERMISSION DENIED accessing /etc/hosts"

	// when
	got := domain.ClassifyCapabilityViolation(input)

	// then
	if got != domain.CapabilityViolationFilesystem {
		t.Errorf("ClassifyCapabilityViolation = %v, want CapabilityViolationFilesystem", got)
	}
}

// TestScanJournalsForCapabilityViolations_NoEntries verifies empty journals return empty slice.
func TestScanJournalsForCapabilityViolations_NoEntries(t *testing.T) {
	// given: no journal entries
	entries := []domain.JournalEntry{}

	// when
	violations := domain.ScanJournalsForCapabilityViolations(entries)

	// then
	if len(violations) != 0 {
		t.Errorf("ScanJournalsForCapabilityViolations = %v, want empty", violations)
	}
}

// TestScanJournalsForCapabilityViolations_DetectsNetworkViolation verifies journal scanning.
func TestScanJournalsForCapabilityViolations_DetectsNetworkViolation(t *testing.T) {
	// given
	entries := []domain.JournalEntry{
		{
			Status: "failed",
			Reason: "curl: (6) Could not resolve host: api.example.com",
		},
	}

	// when
	violations := domain.ScanJournalsForCapabilityViolations(entries)

	// then
	if len(violations) == 0 {
		t.Fatal("ScanJournalsForCapabilityViolations should detect network violation")
	}
	if violations[0].Type != domain.CapabilityViolationNetwork {
		t.Errorf("violation type = %v, want CapabilityViolationNetwork", violations[0].Type)
	}
}

// TestScanJournalsForCapabilityViolations_IgnoresSuccessEntries verifies only failures are scanned.
func TestScanJournalsForCapabilityViolations_IgnoresSuccessEntries(t *testing.T) {
	// given
	entries := []domain.JournalEntry{
		{
			Status: "success",
			Reason: "command not found error was bypassed",
		},
	}

	// when
	violations := domain.ScanJournalsForCapabilityViolations(entries)

	// then: success entries should not trigger violations
	if len(violations) != 0 {
		t.Errorf("success entries should not produce violations, got: %v", violations)
	}
}

// TestFormatCapabilityViolationsSection_Empty verifies empty violations produce empty section.
func TestFormatCapabilityViolationsSection_Empty(t *testing.T) {
	// given
	var violations []domain.CapabilityViolation

	// when
	section := domain.FormatCapabilityViolationsSection(violations)

	// then
	if strings.TrimSpace(section) != "" {
		t.Errorf("FormatCapabilityViolationsSection(empty) = %q, want empty", section)
	}
}

// TestFormatCapabilityViolationsSection_WithViolation verifies section rendering.
func TestFormatCapabilityViolationsSection_WithViolation(t *testing.T) {
	// given
	violations := []domain.CapabilityViolation{
		{Type: domain.CapabilityViolationNetwork, Message: "Could not resolve host"},
	}

	// when
	section := domain.FormatCapabilityViolationsSection(violations)

	// then
	if section == "" {
		t.Fatal("FormatCapabilityViolationsSection should not be empty")
	}
	if !strings.Contains(section, "network") && !strings.Contains(section, "Network") {
		t.Errorf("section should mention network violation: %q", section)
	}
	if !strings.Contains(section, "Could not resolve host") {
		t.Errorf("section should contain message: %q", section)
	}
}
