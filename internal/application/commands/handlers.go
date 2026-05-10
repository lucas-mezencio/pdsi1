package commands

import (
	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// Handlers aggregates command handlers for the application layer.
type Handlers struct {
	Users         *UserCommandHandler
	Doctors       *DoctorCommandHandler
	Prescriptions *PrescriptionCommandHandler
}

// NewHandlers constructs all command handlers with dependencies.
func NewHandlers(
	userRepo user.Repository,
	doctorRepo doctor.Repository,
	prescriptionRepo prescription.Repository,
	scheduler application.NotificationScheduler,
) *Handlers {
	return &Handlers{
		Users:         NewUserCommandHandler(userRepo),
		Doctors:       NewDoctorCommandHandler(doctorRepo),
		Prescriptions: NewPrescriptionCommandHandler(prescriptionRepo, userRepo, doctorRepo, scheduler),
	}
}
