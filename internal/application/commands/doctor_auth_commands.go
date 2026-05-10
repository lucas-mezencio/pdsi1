package commands

import (
	"context"
	"errors"
	"strings"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

// RegisterDoctorCommand holds data to create doctor account in Firebase and local DB.
type RegisterDoctorCommand struct {
	Name          string
	Email         string
	Phone         string
	Password      string
	Specialty     string
	LicenseNumber string
}

// LoginDoctorCommand holds doctor login credentials.
type LoginDoctorCommand struct {
	Email    string
	Password string
}

// DoctorAuthCommandHandler handles doctor register/login operations.
type DoctorAuthCommandHandler struct {
	repo         doctor.Repository
	authProvider AuthenticationProvider
}

// NewDoctorAuthCommandHandler creates a DoctorAuthCommandHandler.
func NewDoctorAuthCommandHandler(repo doctor.Repository, authProvider AuthenticationProvider) *DoctorAuthCommandHandler {
	return &DoctorAuthCommandHandler{
		repo:         repo,
		authProvider: authProvider,
	}
}

// Register creates doctor account in Firebase Auth and links it to doctors.firebase_id.
func (h *DoctorAuthCommandHandler) Register(ctx context.Context, cmd RegisterDoctorCommand) (*doctor.Doctor, error) {
	if h.authProvider == nil {
		return nil, application.ErrAuthNotConfigured
	}
	if strings.TrimSpace(cmd.Name) == "" ||
		strings.TrimSpace(cmd.Email) == "" ||
		strings.TrimSpace(cmd.Phone) == "" ||
		strings.TrimSpace(cmd.Password) == "" ||
		strings.TrimSpace(cmd.LicenseNumber) == "" {
		return nil, application.ErrInvalidInput
	}

	email := strings.TrimSpace(cmd.Email)
	license := strings.TrimSpace(cmd.LicenseNumber)

	_, err := h.repo.FindByEmail(ctx, email)
	if err == nil {
		return nil, application.ErrEmailAlreadyInUse
	}
	if !errors.Is(err, doctor.ErrDoctorNotFound) {
		return nil, err
	}

	_, err = h.repo.FindByLicenseNumber(ctx, license)
	if err == nil {
		return nil, application.ErrLicenseAlreadyInUse
	}
	if !errors.Is(err, doctor.ErrDoctorNotFound) {
		return nil, err
	}

	firebaseID, err := h.authProvider.CreateUser(ctx, email, cmd.Password)
	if err != nil {
		return nil, err
	}

	entity, err := doctor.NewDoctor(
		strings.TrimSpace(cmd.Name),
		email,
		strings.TrimSpace(cmd.Phone),
		strings.TrimSpace(cmd.Specialty),
		license,
	)
	if err != nil {
		_ = h.authProvider.DeleteUser(ctx, firebaseID)
		return nil, err
	}
	entity.LinkFirebaseAccount(firebaseID)

	if err := h.repo.Save(ctx, entity); err != nil {
		_ = h.authProvider.DeleteUser(ctx, firebaseID)
		return nil, err
	}

	return entity, nil
}

// Login validates doctor credentials on Firebase and returns linked local doctor.
func (h *DoctorAuthCommandHandler) Login(ctx context.Context, cmd LoginDoctorCommand) (*doctor.Doctor, error) {
	if h.authProvider == nil {
		return nil, application.ErrAuthNotConfigured
	}
	email := strings.TrimSpace(cmd.Email)
	password := strings.TrimSpace(cmd.Password)
	if email == "" || password == "" {
		return nil, application.ErrInvalidInput
	}

	firebaseID, err := h.authProvider.SignIn(ctx, email, password)
	if err != nil {
		return nil, err
	}

	entity, err := h.repo.FindByFirebaseID(ctx, firebaseID)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, doctor.ErrDoctorNotFound) {
		return nil, err
	}

	// Backfill legacy doctors created before firebase_id existed.
	entity, err = h.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, doctor.ErrDoctorNotFound) {
			return nil, application.ErrDoctorNotFound
		}
		return nil, err
	}

	entity.LinkFirebaseAccount(firebaseID)
	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}
