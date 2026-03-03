package domain

import (
	"encoding/json"
	"errors"
	"fmt"
)

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
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var de *DeviationError
	if errors.As(err, &de) {
		return 2
	}
	return 1
}

// PruneResult holds the outcome of an archive prune operation.
type PruneResult struct {
	Candidates []string // basenames of files older than threshold
	Deleted    int      // number of files actually removed (0 in dry-run)
}

// DoctorCheck represents the result of checking a single external command.
type DoctorCheck struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Path     string `json:"path"`
	Version  string `json:"version"`
	OK       bool   `json:"ok"`
}

// FormatDoctorJSON returns the checks as a JSON array string.
func FormatDoctorJSON(checks []DoctorCheck) (string, error) {
	data, err := json.Marshal(checks)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DoctorOutput is the structured output for the doctor command.
// When metrics are not available (no repo-path), Metrics is nil and omitted from JSON.
type DoctorOutput struct {
	Checks  []DoctorCheck  `json:"checks"`
	Metrics *DoctorMetrics `json:"metrics,omitempty"`
}

// DoctorMetrics holds computed metrics for a repository.
type DoctorMetrics struct {
	SuccessRate string `json:"success_rate"`
}

// FormatDoctorOutputJSON returns the DoctorOutput as a pretty-printed JSON string.
func FormatDoctorOutputJSON(output DoctorOutput) (string, error) {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
