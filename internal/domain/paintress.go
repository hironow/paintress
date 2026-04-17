package domain

import (
	"encoding/json"
	"errors"
	"fmt"
)

// StateDir is the name of the paintress state directory.
const StateDir = ".expedition"

// RunSummary holds the results of a paintress loop run.
type RunSummary struct {
	Total           int64  `json:"total"`
	Success         int64  `json:"success"`
	Skipped         int64  `json:"skipped"`
	Failed          int64  `json:"failed"`
	Bugs            int64  `json:"bugs"`
	MidHighSeverity int64  `json:"mid_high_severity"`
	Gradient        string `json:"gradient"`
}

// FormatSummaryJSON returns the summary as a JSON string.
func FormatSummaryJSON(s RunSummary) (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeviationError is returned when expedition finds deviations (failures detected).
// Callers can use errors.As to distinguish deviation from runtime errors.
type DeviationError struct {
	Failed int
}

func (e *DeviationError) Error() string {
	return fmt.Sprintf("deviation detected: %d failure(s)", e.Failed)
}

// ExitCode maps an error to a process exit code.
//
//	nil             → 0 (success)
//	DeviationError  → 2 (deviation detected)
//	other           → 1 (runtime error)
//
// SilentError wraps an error whose message has already been printed to stderr
// by the command itself. main.go should suppress output for this error
// while still honouring the exit code via ExitCode.
type SilentError struct{ Err error }

func (e *SilentError) Error() string { return e.Err.Error() }
func (e *SilentError) Unwrap() error { return e.Err }

// exitCoder is an interface for errors that carry a specific exit code.
// cmd.ExitError implements this without importing domain (avoids cycles).
type exitCoder interface {
	ExitCode() int
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	// Check for errors with explicit exit codes (e.g. cmd.ExitError)
	var ec exitCoder
	if errors.As(err, &ec) {
		return ec.ExitCode()
	}
	var de *DeviationError
	if errors.As(err, &de) {
		return 2
	}
	return 1
}

// PruneResult holds the outcome of an archive prune operation.
type PruneResult struct { // nosemgrep: first-class-collection.raw-slice-field-domain-go -- Candidates is an operation result slice, not a domain aggregate; FCC wrapping adds complexity with no safety benefit [permanent]
	Candidates []string // basenames of files older than threshold
	Deleted    int      // number of files actually removed (0 in dry-run)
}

