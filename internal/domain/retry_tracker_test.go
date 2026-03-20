package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestRetryTracker_Track_CapsAtMaxRetries(t *testing.T) {
	// given
	rt := domain.NewRetryTrackerWithMax(3)
	issues := []string{"ISS-1", "ISS-2"}

	// when: track 5 times
	var lastCount int
	for range 5 {
		lastCount = rt.Track(issues)
	}

	// then: count keeps incrementing (Track always increments)
	if lastCount != 5 {
		t.Errorf("Track should always increment; got count=%d, want 5", lastCount)
	}
}

func TestRetryTracker_Exhausted_TrueAfterMax(t *testing.T) {
	// given
	rt := domain.NewRetryTrackerWithMax(2)
	issues := []string{"ISS-1"}
	rt.Track(issues)
	rt.Track(issues)

	// when / then
	if !rt.Exhausted(issues) {
		t.Fatal("expected Exhausted=true after 2 tracks with max=2")
	}
}

func TestRetryTracker_Exhausted_FalseBeforeMax(t *testing.T) {
	// given
	rt := domain.NewRetryTrackerWithMax(3)
	issues := []string{"ISS-1"}
	rt.Track(issues)

	// when / then
	if rt.Exhausted(issues) {
		t.Fatal("expected Exhausted=false with 1 track and max=3")
	}
}

func TestRetryTracker_ZeroMax_MeansUnlimited(t *testing.T) {
	// given: default constructor (max=0)
	rt := domain.NewRetryTracker()
	issues := []string{"ISS-1"}
	for range 100 {
		rt.Track(issues)
	}

	// when / then: never exhausted
	if rt.Exhausted(issues) {
		t.Fatal("expected Exhausted=false with max=0 (unlimited)")
	}
}
