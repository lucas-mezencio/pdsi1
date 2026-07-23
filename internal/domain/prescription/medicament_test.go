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
			name: "valid medicament - once with seconds",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "00:00:01",
				Times:     []string{"08:00:05"},
				Doses:     1,
			},
			wantErr: false,
		},
		{
			name: "valid medicament - once daily for 7 days",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     7,
			},
			wantErr: false,
		},
		{
			name: "valid medicament - twice daily for 7 days",
			medicament: Medicament{
				Name:      "Lisinopril",
				Dosage:    "10mg",
				Frequency: "12:00",
				Times:     []string{"08:00", "20:00"},
				Doses:     14,
			},
			wantErr: false,
		},
		{
			name: "valid medicament - three times daily for 7 days",
			medicament: Medicament{
				Name:      "Example",
				Dosage:    "100mg",
				Frequency: "08:00",
				Times:     []string{"06:00", "14:00", "22:00"},
				Doses:     21,
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
				Doses:     7,
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
				Doses:     7,
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
				Doses:     7,
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
				Doses:     7,
			},
			wantErr: true,
		},
		{
			name: "zero doses",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     0,
			},
			wantErr: true,
		},
		{
			name: "negative doses",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     -1,
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
				Doses:     7,
			},
			wantErr: true,
		},
		{
			name: "invalid time format with seconds",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00:5"},
				Doses:     7,
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
				Doses:     7,
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
				Doses:     7,
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
		{"valid time with seconds", "08:00:01", false},
		{"valid time midnight", "00:00", false},
		{"valid time noon", "12:00", false},
		{"valid time end of day", "23:59", false},
		{"invalid - missing colon", "0800", true},
		{"invalid - single digit hour", "8:00", true},
		{"invalid - single digit seconds", "08:00:1", true},
		{"invalid - hour too high", "24:00", true},
		{"invalid - minute too high", "08:60", true},
		{"invalid - second too high", "08:00:60", true},
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
		wantSecond int
		wantDay    int // 0 = today, 1 = tomorrow
	}{
		{
			name: "next time is later today",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "12:00",
				Times:     []string{"08:00", "20:00"},
				Doses:     14,
			},
			now:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			wantHour:   20,
			wantMinute: 0,
			wantSecond: 0,
			wantDay:    0,
		},
		{
			name: "next time is tomorrow",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     7,
			},
			now:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			wantHour:   8,
			wantMinute: 0,
			wantSecond: 0,
			wantDay:    1,
		},
		{
			name: "next time is first time today",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "08:00",
				Times:     []string{"06:00", "14:00", "22:00"},
				Doses:     21,
			},
			now:        time.Date(2024, 1, 1, 5, 0, 0, 0, time.UTC),
			wantHour:   6,
			wantMinute: 0,
			wantSecond: 0,
			wantDay:    0,
		},
		{
			name: "next time includes seconds",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "12:00",
				Times:     []string{"08:00:05", "20:00:15"},
				Doses:     14,
			},
			now:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			wantHour:   20,
			wantMinute: 0,
			wantSecond: 15,
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
			if nextTime.Second() != tt.wantSecond {
				t.Errorf("expected second %d, got %d", tt.wantSecond, nextTime.Second())
			}
			if nextTime.Day() != expectedDay {
				t.Errorf("expected day %d, got %d", expectedDay, nextTime.Day())
			}
		})
	}
}

func TestMedicament_CalculateEndDate(t *testing.T) {
	tests := []struct {
		name       string
		medicament Medicament
		startDate  time.Time
		wantDays   int // Expected days from start date
	}{
		{
			name: "once daily for 7 days",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     7,
			},
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantDays:  6, // Day 1-7, so end date is 6 days after start
		},
		{
			name: "twice daily for 7 days (14 doses)",
			medicament: Medicament{
				Name:      "Lisinopril",
				Dosage:    "10mg",
				Frequency: "12:00",
				Times:     []string{"08:00", "20:00"},
				Doses:     14,
			},
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantDays:  6, // 14 doses / 2 per day = 7 days
		},
		{
			name: "three times daily for 7 days (21 doses)",
			medicament: Medicament{
				Name:      "Metformin",
				Dosage:    "500mg",
				Frequency: "08:00",
				Times:     []string{"06:00", "14:00", "22:00"},
				Doses:     21,
			},
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantDays:  6, // 21 doses / 3 per day = 7 days
		},
		{
			name: "single dose",
			medicament: Medicament{
				Name:      "Emergency Med",
				Dosage:    "50mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     1,
			},
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantDays:  0, // Same day
		},
		{
			name: "30 days of once daily",
			medicament: Medicament{
				Name:      "Long Term Med",
				Dosage:    "25mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     30,
			},
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantDays:  29, // 30 days
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endDate := tt.medicament.CalculateEndDate(tt.startDate)
			expectedEndDate := tt.startDate.AddDate(0, 0, tt.wantDays)

			if !endDate.Equal(expectedEndDate) {
				t.Errorf("CalculateEndDate() = %v, want %v", endDate, expectedEndDate)
			}
		})
	}
}

func TestMedicament_CalculateDaysRemaining(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		medicament Medicament
		now        time.Time
		wantDays   int
	}{
		{
			name: "full 7 days remaining",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     7,
			},
			now:      startDate,
			wantDays: 6,
		},
		{
			name: "3 days remaining",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     7,
			},
			now:      startDate.AddDate(0, 0, 3),
			wantDays: 3,
		},
		{
			name: "prescription ended",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     7,
			},
			now:      startDate.AddDate(0, 0, 10),
			wantDays: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days := tt.medicament.CalculateDaysRemaining(startDate, tt.now)
			if days != tt.wantDays {
				t.Errorf("CalculateDaysRemaining() = %d, want %d", days, tt.wantDays)
			}
		})
	}
}

func TestMedicament_IsCompleted(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		medicament Medicament
		now        time.Time
		want       bool
	}{
		{
			name: "not completed - same day",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     7,
			},
			now:  startDate,
			want: false,
		},
		{
			name: "not completed - middle of prescription",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     7,
			},
			now:  startDate.AddDate(0, 0, 3),
			want: false,
		},
		{
			name: "completed - after end date",
			medicament: Medicament{
				Name:      "Aspirin",
				Dosage:    "100mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     7,
			},
			now:  startDate.AddDate(0, 0, 10),
			want: true,
		},
		{
			name: "single dose - completed next day",
			medicament: Medicament{
				Name:      "Emergency Med",
				Dosage:    "50mg",
				Frequency: "24:00",
				Times:     []string{"08:00"},
				Doses:     1,
			},
			now:  startDate.AddDate(0, 0, 1),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed := tt.medicament.IsCompleted(startDate, tt.now)
			if completed != tt.want {
				t.Errorf("IsCompleted() = %v, want %v", completed, tt.want)
			}
		})
	}
}
