package prescription

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Prescription represents a medical prescription with multiple medicaments
type Prescription struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	MedicID     string       `json:"medic_id"`
	Medicaments []Medicament `json:"medicaments"`
	Active      bool         `json:"active"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// NewPrescription creates a new Prescription with validation
func NewPrescription(userID, medicID string, medicaments []Medicament) (*Prescription, error) {
	if err := validatePrescription(userID, medicID, medicaments); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Prescription{
		ID:          uuid.New().String(),
		UserID:      userID,
		MedicID:     medicID,
		Medicaments: medicaments,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Validate validates the prescription
func (p *Prescription) Validate() error {
	return validatePrescription(p.UserID, p.MedicID, p.Medicaments)
}

// Activate activates the prescription
func (p *Prescription) Activate() {
	p.Active = true
	p.UpdatedAt = time.Now()
}

// Deactivate deactivates the prescription
func (p *Prescription) Deactivate() {
	p.Active = false
	p.UpdatedAt = time.Now()
}

// UpdateMedicaments updates the medicaments in the prescription
func (p *Prescription) UpdateMedicaments(medicaments []Medicament) error {
	if len(medicaments) == 0 {
		return ErrNoMedicaments
	}

	for _, m := range medicaments {
		if err := m.Validate(); err != nil {
			return err
		}
	}

	p.Medicaments = medicaments
	p.UpdatedAt = time.Now()
	return nil
}

// GetAllNotificationTimes returns all scheduled notification times for this prescription
func (p *Prescription) GetAllNotificationTimes() []NotificationSchedule {
	var schedules []NotificationSchedule

	for _, medicament := range p.Medicaments {
		for _, timeStr := range medicament.Times {
			schedules = append(schedules, NotificationSchedule{
				PrescriptionID: p.ID,
				UserID:         p.UserID,
				MedicamentName: medicament.Name,
				Dosage:         medicament.Dosage,
				Time:           timeStr,
			})
		}
	}

	return schedules
}

// NotificationSchedule represents a scheduled notification for a medicament
type NotificationSchedule struct {
	PrescriptionID string
	UserID         string
	MedicamentName string
	Dosage         string
	Time           string // HH:MM format
}

// validatePrescription validates prescription fields
func validatePrescription(userID, medicID string, medicaments []Medicament) error {
	if userID == "" {
		return ErrInvalidUserID
	}
	if medicID == "" {
		return ErrInvalidMedicID
	}
	if len(medicaments) == 0 {
		return ErrNoMedicaments
	}

	// Validate each medicament
	for i, m := range medicaments {
		if err := m.Validate(); err != nil {
			return errors.New("medicament " + string(rune(i)) + ": " + err.Error())
		}
	}

	return nil
}

// Domain errors
var (
	ErrPrescriptionNotFound = errors.New("prescription not found")
	ErrInvalidUserID        = errors.New("invalid user ID")
	ErrInvalidMedicID       = errors.New("invalid medic ID")
	ErrNoMedicaments        = errors.New("prescription must have at least one medicament")
)
