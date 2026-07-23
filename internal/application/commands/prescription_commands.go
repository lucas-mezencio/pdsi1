package commands

import (
	"context"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// CreatePrescriptionCommand holds data to create a new prescription.
type CreatePrescriptionCommand struct {
	UserID      string
	MedicID     string
	Medicaments []prescription.Medicament
}

// UpdatePrescriptionCommand holds data to update prescription medicaments.
type UpdatePrescriptionCommand struct {
	ID          string
	Medicaments []prescription.Medicament
}

// ActivatePrescriptionCommand activates a prescription.
type ActivatePrescriptionCommand struct {
	ID string
}

// DeactivatePrescriptionCommand deactivates a prescription.
type DeactivatePrescriptionCommand struct {
	ID string
}

// DeletePrescriptionCommand removes a prescription.
type DeletePrescriptionCommand struct {
	ID string
}

// PrescriptionCommandHandler handles prescription write operations.
type PrescriptionCommandHandler struct {
	repo       prescription.Repository
	userRepo   user.Repository
	doctorRepo doctor.Repository
	scheduler  application.NotificationScheduler
}

// NewPrescriptionCommandHandler creates a PrescriptionCommandHandler.
func NewPrescriptionCommandHandler(
	repo prescription.Repository,
	userRepo user.Repository,
	doctorRepo doctor.Repository,
	scheduler application.NotificationScheduler,
) *PrescriptionCommandHandler {
	return &PrescriptionCommandHandler{
		repo:       repo,
		userRepo:   userRepo,
		doctorRepo: doctorRepo,
		scheduler:  scheduler,
	}
}

// Create creates a new prescription and schedules notifications.
func (h *PrescriptionCommandHandler) Create(ctx context.Context, cmd CreatePrescriptionCommand) (*prescription.Prescription, error) {
	if cmd.UserID == "" || cmd.MedicID == "" {
		return nil, application.ErrInvalidInput
	}

	userExists, err := h.userRepo.Exists(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if !userExists {
		return nil, application.ErrUserNotFound
	}

	doctorExists, err := h.doctorRepo.Exists(ctx, cmd.MedicID)
	if err != nil {
		return nil, err
	}
	if !doctorExists {
		return nil, application.ErrDoctorNotFound
	}

	entity, err := prescription.NewPrescription(cmd.UserID, cmd.MedicID, cmd.Medicaments)
	if err != nil {
		return nil, err
	}

	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	if err := h.scheduleNotifications(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// UpdateMedicaments updates medicaments and reschedules notifications.
func (h *PrescriptionCommandHandler) UpdateMedicaments(ctx context.Context, cmd UpdatePrescriptionCommand) (*prescription.Prescription, error) {
	if cmd.ID == "" || len(cmd.Medicaments) == 0 {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		if errors.Is(err, prescription.ErrPrescriptionNotFound) {
			return nil, application.ErrPrescriptionNotFound
		}
		return nil, err
	}

	if err := entity.UpdateMedicaments(cmd.Medicaments); err != nil {
		return nil, err
	}

	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	if err := h.scheduler.CancelByPrescriptionID(ctx, entity.ID); err != nil {
		return nil, err
	}
	if err := h.scheduleNotifications(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// Activate activates a prescription and schedules notifications.
func (h *PrescriptionCommandHandler) Activate(ctx context.Context, cmd ActivatePrescriptionCommand) (*prescription.Prescription, error) {
	if cmd.ID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		if errors.Is(err, prescription.ErrPrescriptionNotFound) {
			return nil, application.ErrPrescriptionNotFound
		}
		return nil, err
	}

	entity.Activate()
	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	if err := h.scheduleNotifications(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// Deactivate deactivates a prescription and cancels notifications.
func (h *PrescriptionCommandHandler) Deactivate(ctx context.Context, cmd DeactivatePrescriptionCommand) (*prescription.Prescription, error) {
	if cmd.ID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		if errors.Is(err, prescription.ErrPrescriptionNotFound) {
			return nil, application.ErrPrescriptionNotFound
		}
		return nil, err
	}

	entity.Deactivate()
	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	if err := h.scheduler.CancelByPrescriptionID(ctx, entity.ID); err != nil {
		return nil, err
	}

	return entity, nil
}

// Delete removes a prescription and cancels notifications.
func (h *PrescriptionCommandHandler) Delete(ctx context.Context, cmd DeletePrescriptionCommand) error {
	if cmd.ID == "" {
		return application.ErrInvalidInput
	}

	exists, err := h.repo.Exists(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if !exists {
		return application.ErrPrescriptionNotFound
	}

	if err := h.scheduler.CancelByPrescriptionID(ctx, cmd.ID); err != nil {
		return err
	}

	return h.repo.Delete(ctx, cmd.ID)
}

func (h *PrescriptionCommandHandler) scheduleNotifications(ctx context.Context, entity *prescription.Prescription) error {
	if h.scheduler == nil || !entity.Active {
		return nil
	}

	startDate := entity.CreatedAt
	for _, schedule := range entity.GetAllNotificationTimes() {
		if err := h.scheduler.Schedule(ctx, schedule, startDate); err != nil {
			return err
		}
	}

	return nil
}
