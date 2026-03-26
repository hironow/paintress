package domain

import "time"

// RecoveryAction describes what the expedition loop should do.
type RecoveryAction string

const (
	RecoveryRetry RecoveryAction = "retry"
	RecoveryHalt  RecoveryAction = "halt"
)

// RecoveryDecision is the aggregate's verdict on what to do after Gommage.
type RecoveryDecision struct {
	RecoveryKind RecoveryAction
	Class       GommageClass
	Cooldown    time.Duration
	RetryNum    int
	MaxRetry    int
	KeepWorkDir bool
}

// IsRetry returns true if the decision is to retry.
func (d RecoveryDecision) IsRetry() bool {
	return d.RecoveryKind == RecoveryRetry
}

// cooldown base values per class: [retry1, retry2].
var cooldownBase = map[GommageClass][2]time.Duration{
	GommageClassTimeout:    {30 * time.Second, 90 * time.Second},
	GommageClassRateLimit:  {60 * time.Second, 180 * time.Second},
	GommageClassParseError: {5 * time.Second, 15 * time.Second},
}

// CooldownForClass returns the cooldown duration for a given class and retry number (1-indexed).
func CooldownForClass(class GommageClass, retryNum int) time.Duration {
	base, ok := cooldownBase[class]
	if !ok {
		return 0
	}
	if retryNum <= 1 {
		return base[0]
	}
	return base[1]
}
