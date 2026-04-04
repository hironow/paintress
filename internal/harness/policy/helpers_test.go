package policy_test

import "strings"

func containsStr(s, sub string) bool {
	return strings.Contains(s, sub)
}
