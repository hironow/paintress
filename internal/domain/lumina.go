package domain

// Lumina represents a learned passive skill extracted from past expedition journals.
type Lumina struct {
	Pattern string // The learned pattern / lesson
	Source  string // Which journal(s) contributed
	Uses    int    // How many times this pattern appeared
}
