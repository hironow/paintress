package domain

import "sync"

// IssueClaimRegistry prevents multiple parallel workers from working on the
// same issue simultaneously. Thread-safe via mutex.
type IssueClaimRegistry struct {
	mu     sync.Mutex
	claims map[string]int // issueID -> expedition number
}

// NewIssueClaimRegistry creates an empty registry.
func NewIssueClaimRegistry() *IssueClaimRegistry {
	return &IssueClaimRegistry{claims: make(map[string]int)}
}

// TryClaim attempts to claim an issue for the given expedition.
// Returns (true, 0) on success, or (false, holderExpedition) if already claimed.
func (r *IssueClaimRegistry) TryClaim(issueID string, expedition int) (bool, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if holder, exists := r.claims[issueID]; exists {
		return false, holder
	}
	r.claims[issueID] = expedition
	return true, 0
}

// Release removes the claim on an issue. Safe to call for unclaimed issues.
func (r *IssueClaimRegistry) Release(issueID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.claims, issueID)
}

// ActiveClaims returns the number of currently claimed issues.
func (r *IssueClaimRegistry) ActiveClaims() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.claims)
}
