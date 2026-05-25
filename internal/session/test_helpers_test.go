package session

// white-box-reason: test infrastructure: shared helpers constructing unexported types for sibling tests

import "strings"

// containsStr is a simple substring check without importing strings.
func containsStr(s, sub string) bool {
	return strings.Contains(s, sub)
}
