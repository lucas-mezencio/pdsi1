package prescription

import (
	"testing"
	"time"
)

func TestMedicament_Validate(t *testing.T) {
	tests := []struct {
		name       string
		medicament Medicament
		wantErr    bool
	}{
		{
			name: "valid medicament - once daily",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
			},
			wantErr: false,
		},
		{
			name: "valid medicament - twice daily",
			medicament: Medicament{
				Name:      "Lisinopril",
				Dosage:    "10mg",
				Frequency: "12:00",
				Times:     []string{"08:00", "20:00"},
			},
			wantErr: false,
		},
		{
			name: "valid medicament - three times daily",
			medicament: Medicament{
				Name:      "Example",
				Dosage:    "100mg",
				Frequency: "08:00",
				Times:     []string{"06:00", "14:00", "22:00"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			medicament: Medicament{
				Name:      "",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
			},
			wantErr: true,
		},
		{
			name: "missing dosage",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "",
				Frequency: "24:00",
				Times:     []string{"08:00"},
			},
			wantErr: true,
		},
		{
			name: "missing frequency",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "",
				Times:     []string{"08:00"},
			},
			wantErr: true,
		},
		{
			name: "missing times",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid time format",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"8:00"},
			},
			wantErr: true,
		},
		{
			name: "invalid frequency format",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24",
				Times:     []string{"08:00"},
			},
			wantErr: true,
		},
		{
			name: "frequency times mismatch",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00", "20:00"}, // Should be 1 time for 24h frequency
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.medicament.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Medicament.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTimeFormat(t *testing.T) {
	tests := []struct {
		name    string
		timeStr string
		wantErr bool
	}{
		{"valid time", "08:00", false},
		{"valid time midnight", "00:00", false},
		{"valid time noon", "12:00", false},
		{"valid time end of day", "23:59", false},
		{"invalid - missing colon", "0800", true},
		{"invalid - single digit hour", "8:00", true},
		{"invalid - hour too high", "24:00", true},
		{"invalid - minute too high", "08:60", true},
		{"invalid - negative hour", "-01:00", true},
		{"invalid - letters", "ab:cd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTimeFormat(tt.timeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTimeFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMedicament_GetNextNotificationTime(t *testing.T) {
	tests := []struct {
		name       string
		medicament Medicament
		now        time.Time
		wantHour   int
		wantMinute int
		wantDay    int // 0 = today, 1 = tomorrow
	}{
		{
			name: "next time is later today",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "12:00",
				Times:     []string{"08:00", "20:00"},
			},
			now:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			wantHour:   20,
			wantMinute: 0,
			wantDay:    0,
		},
		{
			name: "next time is tomorrow",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
			},
			now:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			wantHour:   8,
			wantMinute: 0,
			wantDay:    1,
		},
		{
			name: "next time is first time today",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "08:00",
				Times:     []string{"06:00", "14:00", "22:00"},
			},
			now:        time.Date(2024, 1, 1, 5, 0, 0, 0, time.UTC),
			wantHour:   6,
			wantMinute: 0,
			wantDay:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextTime, err := tt.medicament.GetNextNotificationTime(tt.now)
			if err != nil {
				t.Errorf("GetNextNotificationTime() error = %v", err)
				return
			}

			expectedDay := tt.now.Day()
			if tt.wantDay == 1 {
				expectedDay = tt.now.Add(24 * time.Hour).Day()
			}

			if nextTime.Hour() != tt.wantHour {
				t.Errorf("expected hour %d, got %d", tt.wantHour, nextTime.Hour())
			}
			if nextTime.Minute() != tt.wantMinute {
				t.Errorf("expected minute %d, got %d", tt.wantMinute, nextTime.Minute())
			}
			if nextTime.Day() != expectedDay {
				t.Errorf("expected day %d, got %d", expectedDay, nextTime.Day())
			}
		})
	}
}
