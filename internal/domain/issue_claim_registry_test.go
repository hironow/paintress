package domain_test

import (
	"sync"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestIssueClaimRegistry_TryClaim_Success(t *testing.T) {
	// given
	r := domain.NewIssueClaimRegistry()

	// when
	ok, holder := r.TryClaim("ISSUE-1", 1)

	// then
	if !ok {
		t.Fatal("expected claim to succeed")
	}
	if holder != 0 {
		t.Errorf("holder should be 0 for successful claim, got %d", holder)
	}
}

func TestIssueClaimRegistry_TryClaim_AlreadyClaimed(t *testing.T) {
	// given
	r := domain.NewIssueClaimRegistry()
	r.TryClaim("ISSUE-1", 1)

	// when
	ok, holder := r.TryClaim("ISSUE-1", 2)

	// then
	if ok {
		t.Fatal("expected claim to fail")
	}
	if holder != 1 {
		t.Errorf("holder should be 1, got %d", holder)
	}
}

func TestIssueClaimRegistry_Release_AllowsReclaim(t *testing.T) {
	// given
	r := domain.NewIssueClaimRegistry()
	r.TryClaim("ISSUE-1", 1)
	r.Release("ISSUE-1")

	// when
	ok, _ := r.TryClaim("ISSUE-1", 2)

	// then
	if !ok {
		t.Fatal("expected reclaim to succeed after release")
	}
}

func TestIssueClaimRegistry_ActiveClaims(t *testing.T) {
	// given
	r := domain.NewIssueClaimRegistry()
	r.TryClaim("ISSUE-1", 1)
	r.TryClaim("ISSUE-2", 2)
	r.TryClaim("ISSUE-3", 3)
	r.Release("ISSUE-2")

	// when
	count := r.ActiveClaims()

	// then
	if count != 2 {
		t.Errorf("expected 2 active claims, got %d", count)
	}
}

func TestIssueClaimRegistry_ConcurrentAccess(t *testing.T) {
	// given
	r := domain.NewIssueClaimRegistry()
	var wg sync.WaitGroup
	successes := make(chan int, 10)

	// when: 10 goroutines try to claim the same issue
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(exp int) {
			defer wg.Done()
			if ok, _ := r.TryClaim("ISSUE-1", exp); ok {
				successes <- exp
			}
		}(i)
	}
	wg.Wait()
	close(successes)

	// then: exactly 1 should succeed
	count := 0
	for range successes {
		count++
	}
	if count != 1 {
		t.Errorf("expected exactly 1 successful claim, got %d", count)
	}
}

func TestIssueClaimRegistry_Release_IdempotentForUnclaimed(t *testing.T) {
	// given
	r := domain.NewIssueClaimRegistry()

	// when / then: should not panic
	r.Release("NEVER-CLAIMED")
}
