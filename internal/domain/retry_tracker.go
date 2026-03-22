package domain

import (
	"sort"
	"strings"
	"sync"
)

// RetryTracker tracks retry attempts per issue key set.
// When max > 0, Exhausted returns true once the count reaches max.
// Track always increments regardless of max.
type RetryTracker struct {
	mu      sync.Mutex
	tracker map[string]int
	max     int // 0 = unlimited (backward compatible)
}

// NewRetryTracker returns an initialized RetryTracker with no cap (unlimited).
func NewRetryTracker() *RetryTracker {
	return &RetryTracker{tracker: make(map[string]int)}
}

// NewRetryTrackerWithMax returns a RetryTracker with a maximum retry cap.
// max=0 means unlimited (same as NewRetryTracker).
func NewRetryTrackerWithMax(max int) *RetryTracker {
	return &RetryTracker{tracker: make(map[string]int), max: max}
}

// Track increments and returns the retry count for the given issue set.
// Design decision: Track always increments regardless of max. The max cap is
// enforced only by Exhausted(), which is the authoritative check for whether
// retries are exhausted. This separation allows callers to observe the true
// count (e.g. for logging) even after the cap is reached.
func (rt *RetryTracker) Track(issues []string) int {
	key := RetryKey(issues)

	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.tracker[key]++
	return rt.tracker[key]
}

// Exhausted returns true if the retry count for the given issue set has
// reached or exceeded the configured maximum. Always returns false when max=0.
func (rt *RetryTracker) Exhausted(issues []string) bool {
	if rt.max <= 0 {
		return false
	}
	key := RetryKey(issues)
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.tracker[key] >= rt.max
}

// RetryKey returns the canonical key for an issue set (sorted, comma-joined).
func RetryKey(issues []string) string {
	sorted := make([]string, len(issues))
	copy(sorted, issues)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}
