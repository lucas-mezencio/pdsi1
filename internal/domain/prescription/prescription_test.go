package prescription

import (
	"testing"
	"time"
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
					Doses:     7,
				},
			},
			wantErr: false,
		},
		{
			name:    "valid prescription with multiple medicaments",
			userID:  "user-123",
			medicID: "medic-456",
			medicaments: []Medicament{
				{
					Name:      "Aspirin",
					Dosage:    "100mg",
					Frequency: "24:00",
					Times:     []string{"08:00"},
					Doses:     7,
				},
				{
					Name:      "Lisinopril",
					Dosage:    "10mg",
					Frequency: "12:00",
					Times:     []string{"08:00", "20:00"},
					Doses:     14,
				},
			},
			wantErr: false,
		},
		{
			name:        "missing user ID",
			userID:      "",
			medicID:     "medic-456",
			medicaments: []Medicament{{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7}},
			wantErr:     true,
		},
		{
			name:        "missing medic ID",
			userID:      "user-123",
			medicID:     "",
			medicaments: []Medicament{{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7}},
			wantErr:     true,
		},
		{
			name:        "no medicaments",
			userID:      "user-123",
			medicID:     "medic-456",
			medicaments: []Medicament{},
			wantErr:     true,
		},
		{
			name:    "invalid medicament - missing doses",
			userID:  "user-123",
			medicID: "medic-456",
			medicaments: []Medicament{
				{
					Name:      "Aspirin",
					Dosage:    "100mg",
					Frequency: "24:00",
					Times:     []string{"08:00"},
					Doses:     0,
				},
			},
			wantErr: true,
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
				Doses:     7,
			},
			{
				Name:      "Lisinopril",
				Dosage:    "10mg",
				Frequency: "12:00",
				Times:     []string{"08:00", "20:00"},
				Doses:     14,
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
	if schedules[0].TotalDoses != 7 {
		t.Errorf("expected first schedule to have 7 total doses, got %d", schedules[0].TotalDoses)
	}

	// Verify Lisinopril schedules
	if schedules[1].MedicamentName != "Lisinopril" {
		t.Errorf("expected second schedule to be for Lisinopril, got %s", schedules[1].MedicamentName)
	}
	if schedules[1].TotalDoses != 14 {
		t.Errorf("expected Lisinopril to have 14 total doses, got %d", schedules[1].TotalDoses)
	}
}

func TestPrescription_UpdateMedicaments(t *testing.T) {
	prescription := &Prescription{
		ID:      "test-id",
		UserID:  "user-123",
		MedicID: "medic-456",
		Medicaments: []Medicament{
			{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
		},
	}

	newMedicaments := []Medicament{
		{Name: "Lisinopril", Dosage: "10mg", Frequency: "12:00", Times: []string{"08:00", "20:00"}, Doses: 14},
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
	if prescription.Medicaments[0].Doses != 14 {
		t.Errorf("expected medicament doses to be 14, got %d", prescription.Medicaments[0].Doses)
	}
}

func TestPrescription_UpdateMedicaments_Empty(t *testing.T) {
	prescription := &Prescription{
		ID:      "test-id",
		UserID:  "user-123",
		MedicID: "medic-456",
		Medicaments: []Medicament{
			{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
		},
	}

	err := prescription.UpdateMedicaments([]Medicament{})
	if err == nil {
		t.Error("expected error when updating with empty medicaments")
	}
}

func TestPrescription_UpdateMedicaments_InvalidDoses(t *testing.T) {
	prescription := &Prescription{
		ID:      "test-id",
		UserID:  "user-123",
		MedicID: "medic-456",
		Medicaments: []Medicament{
			{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
		},
	}

	newMedicaments := []Medicament{
		{Name: "Lisinopril", Dosage: "10mg", Frequency: "12:00", Times: []string{"08:00", "20:00"}, Doses: 0},
	}

	err := prescription.UpdateMedicaments(newMedicaments)
	if err == nil {
		t.Error("expected error when updating with invalid doses")
	}
}

func TestPrescription_IsCompleted(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		prescription *Prescription
		now          time.Time
		want         bool
	}{
		{
			name: "not completed - same day",
			prescription: &Prescription{
				ID:        "test-id",
				UserID:    "user-123",
				MedicID:   "medic-456",
				CreatedAt: startDate,
				Medicaments: []Medicament{
					{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
				},
			},
			now:  startDate,
			want: false,
		},
		{
			name: "completed - all medicaments finished",
			prescription: &Prescription{
				ID:        "test-id",
				UserID:    "user-123",
				MedicID:   "medic-456",
				CreatedAt: startDate,
				Medicaments: []Medicament{
					{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
				},
			},
			now:  startDate.AddDate(0, 0, 10),
			want: true,
		},
		{
			name: "not completed - one medicament still active",
			prescription: &Prescription{
				ID:        "test-id",
				UserID:    "user-123",
				MedicID:   "medic-456",
				CreatedAt: startDate,
				Medicaments: []Medicament{
					{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
					{Name: "LongTerm", Dosage: "50mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 30},
				},
			},
			now:  startDate.AddDate(0, 0, 10), // Aspirin done, LongTerm not
			want: false,
		},
		{
			name: "completed - all medicaments finished (multiple)",
			prescription: &Prescription{
				ID:        "test-id",
				UserID:    "user-123",
				MedicID:   "medic-456",
				CreatedAt: startDate,
				Medicaments: []Medicament{
					{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
					{Name: "Lisinopril", Dosage: "10mg", Frequency: "12:00", Times: []string{"08:00", "20:00"}, Doses: 14},
				},
			},
			now:  startDate.AddDate(0, 0, 10), // Both done (7 days each)
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed := tt.prescription.IsCompleted(tt.now)
			if completed != tt.want {
				t.Errorf("IsCompleted() = %v, want %v", completed, tt.want)
			}
		})
	}
}

func TestPrescription_GetEndDate(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		prescription *Prescription
		wantDays     int
	}{
		{
			name: "single medicament - 7 days",
			prescription: &Prescription{
				ID:        "test-id",
				UserID:    "user-123",
				MedicID:   "medic-456",
				CreatedAt: startDate,
				Medicaments: []Medicament{
					{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
				},
			},
			wantDays: 6, // Day 7 is 6 days after day 1
		},
		{
			name: "multiple medicaments - takes the longest",
			prescription: &Prescription{
				ID:        "test-id",
				UserID:    "user-123",
				MedicID:   "medic-456",
				CreatedAt: startDate,
				Medicaments: []Medicament{
					{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
					{Name: "LongTerm", Dosage: "50mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 30},
				},
			},
			wantDays: 29, // 30 days
		},
		{
			name: "multiple medicaments same duration",
			prescription: &Prescription{
				ID:        "test-id",
				UserID:    "user-123",
				MedicID:   "medic-456",
				CreatedAt: startDate,
				Medicaments: []Medicament{
					{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
					{Name: "Lisinopril", Dosage: "10mg", Frequency: "12:00", Times: []string{"08:00", "20:00"}, Doses: 14},
				},
			},
			wantDays: 6, // Both are 7 days
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endDate := tt.prescription.GetEndDate()
			expectedEndDate := startDate.AddDate(0, 0, tt.wantDays)

			if !endDate.Equal(expectedEndDate) {
				t.Errorf("GetEndDate() = %v, want %v", endDate, expectedEndDate)
			}
		})
	}
}

func TestPrescription_Validate(t *testing.T) {
	tests := []struct {
		name         string
		prescription *Prescription
		wantErr      bool
	}{
		{
			name: "valid prescription",
			prescription: &Prescription{
				ID:      "test-id",
				UserID:  "user-123",
				MedicID: "medic-456",
				Medicaments: []Medicament{
					{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
				},
			},
			wantErr: false,
		},
		{
			name: "missing user ID",
			prescription: &Prescription{
				ID:      "test-id",
				UserID:  "",
				MedicID: "medic-456",
				Medicaments: []Medicament{
					{Name: "Aspirin", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid medicament",
			prescription: &Prescription{
				ID:      "test-id",
				UserID:  "user-123",
				MedicID: "medic-456",
				Medicaments: []Medicament{
					{Name: "", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 7},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.prescription.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
