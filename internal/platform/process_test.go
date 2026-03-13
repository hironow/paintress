package platform

// white-box-reason: tests internal isProcessAlive logic across platforms (EPERM, invalid PID)

import (
	"os"
	"testing"
)

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	t.Parallel()

	// given: our own PID is always alive
	pid := os.Getpid()

	// when
	alive := IsProcessAlive(pid)

	// then
	if !alive {
		t.Errorf("IsProcessAlive(%d) = false, want true (own process)", pid)
	}
}

func TestIsProcessAlive_ZeroPID(t *testing.T) {
	t.Parallel()

	// given: PID 0 is not a valid user process
	// when
	alive := IsProcessAlive(0)

	// then
	if alive {
		t.Error("IsProcessAlive(0) = true, want false")
	}
}

func TestIsProcessAlive_NegativePID(t *testing.T) {
	t.Parallel()

	// given: negative PID is invalid
	// when
	alive := IsProcessAlive(-1)

	// then
	if alive {
		t.Error("IsProcessAlive(-1) = true, want false")
	}
}

func TestIsProcessAlive_UnlikelyHighPID(t *testing.T) {
	t.Parallel()

	// given: a PID that almost certainly doesn't exist
	// when
	alive := IsProcessAlive(4194304)

	// then
	if alive {
		t.Error("IsProcessAlive(4194304) = true, want false (unlikely PID)")
	}
}
