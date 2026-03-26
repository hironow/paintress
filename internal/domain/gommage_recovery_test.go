package domain_test

import (
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestCooldownForClass_Timeout(t *testing.T) {
	got := domain.CooldownForClass(domain.GommageClassTimeout, 1)
	if got != 30*time.Second {
		t.Errorf("timeout retry 1: got %v, want 30s", got)
	}
	got = domain.CooldownForClass(domain.GommageClassTimeout, 2)
	if got != 90*time.Second {
		t.Errorf("timeout retry 2: got %v, want 90s", got)
	}
}

func TestCooldownForClass_RateLimit(t *testing.T) {
	got := domain.CooldownForClass(domain.GommageClassRateLimit, 1)
	if got != 60*time.Second {
		t.Errorf("rate_limit retry 1: got %v, want 60s", got)
	}
	got = domain.CooldownForClass(domain.GommageClassRateLimit, 2)
	if got != 180*time.Second {
		t.Errorf("rate_limit retry 2: got %v, want 180s", got)
	}
}

func TestCooldownForClass_ParseError(t *testing.T) {
	got := domain.CooldownForClass(domain.GommageClassParseError, 1)
	if got != 5*time.Second {
		t.Errorf("parse_error retry 1: got %v, want 5s", got)
	}
	got = domain.CooldownForClass(domain.GommageClassParseError, 2)
	if got != 15*time.Second {
		t.Errorf("parse_error retry 2: got %v, want 15s", got)
	}
}

func TestRecoveryDecision_IsRetry(t *testing.T) {
	d := domain.RecoveryDecision{Action: domain.RecoveryRetry}
	if !d.IsRetry() {
		t.Error("expected IsRetry true")
	}
	d2 := domain.RecoveryDecision{Action: domain.RecoveryHalt}
	if d2.IsRetry() {
		t.Error("expected IsRetry false for halt")
	}
}
