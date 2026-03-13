package domain

import "encoding/json"

// CheckStatus represents the outcome of a single doctor check.
type CheckStatus int

const (
	CheckOK   CheckStatus = iota
	CheckFail
	CheckSkip
	CheckWarn
	CheckFixed
)

// StatusLabel returns a display string for the check status.
func (s CheckStatus) StatusLabel() string {
	switch s {
	case CheckOK:
		return "OK"
	case CheckFail:
		return "FAIL"
	case CheckSkip:
		return "SKIP"
	case CheckWarn:
		return "WARN"
	case CheckFixed:
		return "FIX"
	default:
		return "????"
	}
}

// MarshalJSON serializes CheckStatus as its string label.
func (s CheckStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.StatusLabel())
}

// UnmarshalJSON deserializes a CheckStatus from its string label.
func (s *CheckStatus) UnmarshalJSON(data []byte) error {
	var label string
	if err := json.Unmarshal(data, &label); err != nil {
		return err
	}
	switch label {
	case "OK":
		*s = CheckOK
	case "FAIL":
		*s = CheckFail
	case "SKIP":
		*s = CheckSkip
	case "WARN":
		*s = CheckWarn
	case "FIX":
		*s = CheckFixed
	default:
		*s = CheckOK
	}
	return nil
}
