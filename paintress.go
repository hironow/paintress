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
