package queries

import (
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// Handlers aggregates query handlers for the application layer.
type Handlers struct {
	Users         *UserQueryHandler
	Doctors       *DoctorQueryHandler
	Prescriptions *PrescriptionQueryHandler
}

// NewHandlers constructs all query handlers with dependencies.
func NewHandlers(
	userRepo user.Repository,
	doctorRepo doctor.Repository,
	prescriptionRepo prescription.Repository,
) *Handlers {
	return &Handlers{
		Users:         NewUserQueryHandler(userRepo),
		Doctors:       NewDoctorQueryHandler(doctorRepo),
		Prescriptions: NewPrescriptionQueryHandler(prescriptionRepo),
	}
}
