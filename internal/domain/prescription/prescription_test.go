package prescription

import (
	"testing"
)

func TestNewPrescription(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		medicID     string
		medicaments []Medicament
		wantErr     bool
	}{
		{
			name:    "valid prescription",
			userID:  "user-123",
			medicID: "medic-456",
			medicaments: []Medicament{
				{
					Name:      "Aspirin",
					Dosage:    "100mg",
					Frequency: "24:00",
					Times:     []string{"08:00"},
				},
			},
			wantErr: false,
		},
		{
			name:        "missing user ID",
			userID:      "",
			medicID:     "medic-456",
			medicaments: []Medicament{{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}}},
			wantErr:     true,
		},
		{
			name:        "missing medic ID",
			userID:      "user-123",
			medicID:     "",
			medicaments: []Medicament{{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}}},
			wantErr:     true,
		},
		{
			name:        "no medicaments",
			userID:      "user-123",
			medicID:     "medic-456",
			medicaments: []Medicament{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prescription, err := NewPrescription(tt.userID, tt.medicID, tt.medicaments)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPrescription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if prescription.ID == "" {
					t.Error("expected prescription ID to be generated")
				}
				if !prescription.Active {
					t.Error("expected new prescription to be active")
				}
				if prescription.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
			}
		})
	}
}

func TestPrescription_Activate(t *testing.T) {
	prescription := &Prescription{
		ID:      "test-id",
		UserID:  "user-123",
		MedicID: "medic-456",
		Active:  false,
	}

	prescription.Activate()

	if !prescription.Active {
		t.Error("expected prescription to be active")
	}
	if prescription.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestPrescription_Deactivate(t *testing.T) {
	prescription := &Prescription{
		ID:      "test-id",
		UserID:  "user-123",
		MedicID: "medic-456",
		Active:  true,
	}

	prescription.Deactivate()

	if prescription.Active {
		t.Error("expected prescription to be inactive")
	}
	if prescription.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestPrescription_GetAllNotificationTimes(t *testing.T) {
	prescription := &Prescription{
		ID:      "test-id",
		UserID:  "user-123",
		MedicID: "medic-456",
		Medicaments: []Medicament{
			{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
			},
			{
				Name:      "Lisinopril",
				Dosage:    "10mg",
				Frequency: "12:00",
				Times:     []string{"08:00", "20:00"},
			},
		},
	}

	schedules := prescription.GetAllNotificationTimes()

	expectedCount := 3 // 1 from Aspirin + 2 from Lisinopril
	if len(schedules) != expectedCount {
		t.Errorf("expected %d schedules, got %d", expectedCount, len(schedules))
	}

	// Verify first schedule
	if schedules[0].MedicamentName != "Aspirin" {
		t.Errorf("expected first schedule to be for Aspirin, got %s", schedules[0].MedicamentName)
	}
	if schedules[0].Time != "08:00" {
		t.Errorf("expected first schedule time to be 08:00, got %s", schedules[0].Time)
	}
}

func TestPrescription_UpdateMedicaments(t *testing.T) {
	prescription := &Prescription{
		ID:      "test-id",
		UserID:  "user-123",
		MedicID: "medic-456",
		Medicaments: []Medicament{
			{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}},
		},
	}

	newMedicaments := []Medicament{
		{Name: "Lisinopril", Dosage: "10mg", Frequency: "12:00", Times: []string{"08:00", "20:00"}},
	}

	err := prescription.UpdateMedicaments(newMedicaments)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(prescription.Medicaments) != 1 {
		t.Errorf("expected 1 medicament, got %d", len(prescription.Medicaments))
	}
	if prescription.Medicaments[0].Name != "Lisinopril" {
		t.Errorf("expected medicament name to be Lisinopril, got %s", prescription.Medicaments[0].Name)
	}
}

func TestPrescription_UpdateMedicaments_Empty(t *testing.T) {
	prescription := &Prescription{
		ID:      "test-id",
		UserID:  "user-123",
		MedicID: "medic-456",
		Medicaments: []Medicament{
			{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}},
		},
	}

	err := prescription.UpdateMedicaments([]Medicament{})
	if err == nil {
		t.Error("expected error when updating with empty medicaments")
	}
}
