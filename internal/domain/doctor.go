package domain

// CheckStatus represents the outcome of a single doctor check.
type CheckStatus int

const (
	CheckOK CheckStatus = iota
	CheckFail
	CheckSkip
	CheckWarn
	CheckFixed
)

// DoctorCheck holds the outcome of a single doctor check.
type DoctorCheck struct {
	Name    string
	Status  CheckStatus
	Message string
	Hint    string // optional remediation hint shown on failure
}

// DoctorMetrics holds computed metrics for a repository.
type DoctorMetrics struct {
	SuccessRate string `json:"success_rate"`
}

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
