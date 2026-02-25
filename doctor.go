package paintress

import "encoding/json"

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
