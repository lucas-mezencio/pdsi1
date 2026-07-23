package doctor

import "testing"

func TestNewDoctor(t *testing.T) {
	tests := []struct {
		name          string
		doctorName    string
		email         string
		phone         string
		specialty     string
		licenseNumber string
		wantErr       bool
	}{
		{
			name:          "valid doctor",
			doctorName:    "Dr. Smith",
			email:         "smith@hospital.com",
			phone:         "+1234567890",
			specialty:     "Cardiology",
			licenseNumber: "MED-12345",
			wantErr:       false,
		},
		{
			name:          "missing name",
			doctorName:    "",
			email:         "smith@hospital.com",
			phone:         "+1234567890",
			specialty:     "Cardiology",
			licenseNumber: "MED-12345",
			wantErr:       true,
		},
		{
			name:          "missing email",
			doctorName:    "Dr. Smith",
			email:         "",
			phone:         "+1234567890",
			specialty:     "Cardiology",
			licenseNumber: "MED-12345",
			wantErr:       true,
		},
		{
			name:          "missing phone",
			doctorName:    "Dr. Smith",
			email:         "smith@hospital.com",
			phone:         "",
			specialty:     "Cardiology",
			licenseNumber: "MED-12345",
			wantErr:       true,
		},
		{
			name:          "missing license number",
			doctorName:    "Dr. Smith",
			email:         "smith@hospital.com",
			phone:         "+1234567890",
			specialty:     "Cardiology",
			licenseNumber: "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doctor, err := NewDoctor(tt.doctorName, tt.email, tt.phone, tt.specialty, tt.licenseNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDoctor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if doctor.ID == "" {
					t.Error("expected doctor ID to be generated")
				}
				if doctor.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
			}
		})
	}
}

func TestDoctor_Update(t *testing.T) {
	doctor := &Doctor{
		ID:            "test-id",
		Name:          "Dr. Smith",
		Email:         "smith@hospital.com",
		Phone:         "+1234567890",
		Specialty:     "Cardiology",
		LicenseNumber: "MED-12345",
	}

	err := doctor.Update("Dr. Jones", "jones@hospital.com", "+0987654321", "Neurology")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if doctor.Name != "Dr. Jones" {
		t.Errorf("expected name to be updated to 'Dr. Jones', got %s", doctor.Name)
	}
	if doctor.Email != "jones@hospital.com" {
		t.Errorf("expected email to be updated to 'jones@hospital.com', got %s", doctor.Email)
	}
	if doctor.Specialty != "Neurology" {
		t.Errorf("expected specialty to be updated to 'Neurology', got %s", doctor.Specialty)
	}
	if doctor.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}
