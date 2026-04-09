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
