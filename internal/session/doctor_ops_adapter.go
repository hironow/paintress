package session

import "github.com/hironow/paintress/internal/domain"

type doctorOps struct{}

// NewDoctorOps returns a port.DoctorOps implementation.
func NewDoctorOps() *doctorOps {
	return &doctorOps{}
}

func (*doctorOps) RunDoctor(claudeCmd string, continent string, repair bool, mode domain.TrackingMode) []domain.DoctorCheck {
	return RunDoctor(claudeCmd, continent, repair, mode)
}
