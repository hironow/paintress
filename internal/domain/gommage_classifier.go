package domain

import "strings"

// GommageClass identifies the dominant failure pattern in a streak.
type GommageClass string

const (
	GommageClassTimeout    GommageClass = "timeout"
	GommageClassParseError GommageClass = "parse_error"
	GommageClassRateLimit  GommageClass = "rate_limit"
	GommageClassBlocker    GommageClass = "blocker"
	GommageClassSystematic GommageClass = "systematic"
)

// classKeywords maps each class to its detection keywords.
// A reason matches a class if it contains ANY of the keywords.
var classKeywords = map[GommageClass][]string{
	GommageClassTimeout:    {"timeout"},
	GommageClassRateLimit:  {"rate_limit", "429", "quota", "too many requests"},
	GommageClassParseError: {"parse_error", "markers not found"},
	GommageClassBlocker:    {"blocker"},
}

// ClassifyGommage inspects recent failure reasons and returns the dominant class.
// Majority-vote over keyword matching. If no majority, returns GommageClassSystematic.
func ClassifyGommage(reasons []string) GommageClass {
	if len(reasons) == 0 {
		return GommageClassSystematic
	}
	counts := make(map[GommageClass]int)
	for _, reason := range reasons {
		lower := strings.ToLower(reason)
		for class, keywords := range classKeywords {
			matched := false
			for _, kw := range keywords {
				if strings.Contains(lower, kw) {
					matched = true
					break
				}
			}
			if matched {
				counts[class]++
				break // one class per reason
			}
		}
	}
	// Strict majority: more than half. For len=2, need >1; for len=4, need >2.
	// This prevents nondeterministic classification from map iteration order.
	half := len(reasons) / 2
	for class, count := range counts {
		if count > half {
			return class
		}
	}
	return GommageClassSystematic
}
