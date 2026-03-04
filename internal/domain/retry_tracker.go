package domain

import (
	"sort"
	"strings"
	"sync"
)

// RetryTracker tracks retry attempts per issue key set.
type RetryTracker struct {
	mu      sync.Mutex
	tracker map[string]int
}

// NewRetryTracker returns an initialized RetryTracker.
func NewRetryTracker() *RetryTracker {
	return &RetryTracker{tracker: make(map[string]int)}
}

// Track increments and returns the retry count for the given issue set.
func (rt *RetryTracker) Track(issues []string) int {
	sorted := make([]string, len(issues))
	copy(sorted, issues)
	sort.Strings(sorted)
	key := strings.Join(sorted, ",")

	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.tracker[key]++
	return rt.tracker[key]
}

// RetryKey returns the canonical key for an issue set (sorted, comma-joined).
func RetryKey(issues []string) string {
	sorted := make([]string, len(issues))
	copy(sorted, issues)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}
