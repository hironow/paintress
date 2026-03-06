package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestExitCode_Nil(t *testing.T) {
	if code := domain.ExitCode(nil); code != 0 {
		t.Errorf("ExitCode(nil) = %d, want 0", code)
	}
}

func TestExitCode_DeviationError(t *testing.T) {
	err := &domain.DeviationError{Failed: 3}
	if code := domain.ExitCode(err); code != 2 {
		t.Errorf("ExitCode(DeviationError) = %d, want 2", code)
	}
}

func TestExitCode_RegularError(t *testing.T) {
	err := fmt.Errorf("something broke")
	if code := domain.ExitCode(err); code != 1 {
		t.Errorf("ExitCode(regular) = %d, want 1", code)
	}
}

func TestExitCode_WrappedDeviationError(t *testing.T) {
	inner := &domain.DeviationError{Failed: 1}
	wrapped := fmt.Errorf("run: %w", inner)

	if code := domain.ExitCode(wrapped); code != 2 {
		t.Errorf("ExitCode(wrapped DeviationError) = %d, want 2", code)
	}
}

func TestDeviationError_ErrorMessage(t *testing.T) {
	err := &domain.DeviationError{Failed: 5}
	if !errors.Is(err, err) {
		t.Error("DeviationError should satisfy errors.Is with itself")
	}
	if err.Error() == "" {
		t.Error("DeviationError.Error() should not be empty")
	}
}
