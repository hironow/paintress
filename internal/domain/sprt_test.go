package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestSPRT_AllSuccesses_Pass(t *testing.T) {
	// given: 20 consecutive successes
	results := make([]bool, 20)
	for i := range results {
		results[i] = true
	}

	// when
	verdict, state := domain.SPRT(results, domain.DefaultSPRTConfig())

	// then
	if verdict != domain.SPRTPass {
		t.Errorf("expected PASS for all successes, got %s (lambda=%.3f)", verdict, state.LambdaN)
	}
}

func TestSPRT_AllFailures_Fail(t *testing.T) {
	// given: 20 consecutive failures
	results := make([]bool, 20)

	// when
	verdict, state := domain.SPRT(results, domain.DefaultSPRTConfig())

	// then
	if verdict != domain.SPRTFail {
		t.Errorf("expected FAIL for all failures, got %s (lambda=%.3f)", verdict, state.LambdaN)
	}
}

func TestSPRT_MixedResults_Inconclusive(t *testing.T) {
	// given: exactly 3 results alternating (too few to decide)
	results := []bool{true, false, true}

	// when
	verdict, _ := domain.SPRT(results, domain.DefaultSPRTConfig())

	// then
	if verdict != domain.SPRTInconclusive {
		t.Errorf("expected INCONCLUSIVE for 3 mixed results, got %s", verdict)
	}
}

func TestSPRT_BelowThreshold_Fail(t *testing.T) {
	// given: 70% success rate (at p0 boundary) over many trials
	results := make([]bool, 100)
	for i := range results {
		results[i] = i%10 < 7 // 70% success
	}

	// when
	verdict, state := domain.SPRT(results, domain.DefaultSPRTConfig())

	// then: should eventually reach FAIL (p0=0.70 is the null hypothesis)
	if verdict == domain.SPRTPass {
		t.Errorf("expected FAIL or INCONCLUSIVE at 70%% rate (p0), got PASS (lambda=%.3f)", state.LambdaN)
	}
}

func TestSPRT_AboveThreshold_Pass(t *testing.T) {
	// given: 85% success rate (at p1) over many trials
	results := make([]bool, 100)
	for i := range results {
		results[i] = i%20 < 17 // 85% success
	}

	// when
	verdict, state := domain.SPRT(results, domain.DefaultSPRTConfig())

	// then
	if verdict != domain.SPRTPass {
		t.Errorf("expected PASS at 85%% rate (p1), got %s (lambda=%.3f)", verdict, state.LambdaN)
	}
}

func TestSPRT_EmptyResults_Inconclusive(t *testing.T) {
	verdict, _ := domain.SPRT(nil, domain.DefaultSPRTConfig())
	if verdict != domain.SPRTInconclusive {
		t.Errorf("expected INCONCLUSIVE for empty results, got %s", verdict)
	}
}

func TestSPRT_StateFields(t *testing.T) {
	results := []bool{true, true, false, true}
	_, state := domain.SPRT(results, domain.DefaultSPRTConfig())

	if state.Successes != 3 {
		t.Errorf("Successes: got %d, want 3", state.Successes)
	}
	if state.Failures != 1 {
		t.Errorf("Failures: got %d, want 1", state.Failures)
	}
	if state.UpperBound <= 0 {
		t.Error("UpperBound should be positive")
	}
	if state.LowerBound >= 0 {
		t.Error("LowerBound should be negative")
	}
}
