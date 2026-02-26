package paintress

import "encoding/json"

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
