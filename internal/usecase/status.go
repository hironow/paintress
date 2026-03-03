package usecase

import (
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// GetStatus collects current operational status from the event store and filesystem.
// baseDir is the repository root (the "continent" containing .expedition/).
func GetStatus(baseDir string) domain.StatusReport {
	return session.Status(baseDir)
}
