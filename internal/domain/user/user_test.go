package user

import (
	"errors"
	"testing"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name      string
		userName  string
		email     string
		phone     string
		cpf       string
		role      Role
		wantErr   bool
		wantErrIs error
		wantCPF   string
	}{
		{
			name:     "valid user",
			userName: "John Doe",
			email:    "john@example.com",
			phone:    "+1234567890",
			cpf:      "52998224725",
			role:     RoleElderly,
			wantErr:  false,
			wantCPF:  "52998224725",
		},
		{
			name:     "valid user with formatted cpf",
			userName: "John Doe",
			email:    "john@example.com",
			phone:    "+1234567890",
			cpf:      "529.982.247-25",
			role:     RoleCaregiver,
			wantErr:  false,
			wantCPF:  "52998224725",
		},
		{
			name:     "missing name",
			userName: "",
			email:    "john@example.com",
			phone:    "+1234567890",
			cpf:      "52998224725",
			role:     RoleElderly,
			wantErr:  true,
		},
		{
			name:     "missing email",
			userName: "John Doe",
			email:    "",
			phone:    "+1234567890",
			cpf:      "52998224725",
			role:     RoleElderly,
			wantErr:  true,
		},
		{
			name:     "missing phone",
			userName: "John Doe",
			email:    "john@example.com",
			phone:    "",
			cpf:      "52998224725",
			role:     RoleElderly,
			wantErr:  true,
		},
		{
			name:     "invalid CPF",
			userName: "John Doe",
			email:    "john@example.com",
			phone:    "+1234567890",
			cpf:      "12345678900",
			role:     RoleElderly,
			wantErr:  true,
			wantErrIs: ErrInvalidCPF,
		},
		{
			name:     "empty CPF is allowed",
			userName: "John Doe",
			email:    "john@example.com",
			phone:    "+1234567890",
			cpf:      "",
			role:     RoleElderly,
			wantErr:  false,
			wantCPF:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, err := NewUser(tt.userName, tt.email, tt.phone, tt.cpf, tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
			}
			if !tt.wantErr {
				if entity.ID == "" {
					t.Error("expected user ID to be generated")
				}
				if entity.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
				if entity.CPF != tt.wantCPF {
					t.Errorf("expected CPF %q, got %q", tt.wantCPF, entity.CPF)
				}
			}
		})
	}
}

func TestUser_Update(t *testing.T) {
	user := &User{
		ID:    "test-id",
		Name:  "John Doe",
		Email: "john@example.com",
		Phone: "+1234567890",
		CPF:   "52998224725",
	}

	err := user.Update("Jane Doe", "jane@example.com", "+0987654321", "39053344705")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if user.Name != "Jane Doe" {
		t.Errorf("expected name to be updated to 'Jane Doe', got %s", user.Name)
	}
	if user.Email != "jane@example.com" {
		t.Errorf("expected email to be updated to 'jane@example.com', got %s", user.Email)
	}
	if user.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	if user.CPF != "39053344705" {
		t.Errorf("expected CPF to be updated, got %s", user.CPF)
	}
}

func TestUser_Update_InvalidCPF(t *testing.T) {
	user := &User{
		ID:    "test-id",
		Name:  "John Doe",
		Email: "john@example.com",
		Phone: "+1234567890",
		CPF:   "52998224725",
	}

	err := user.Update("Jane Doe", "jane@example.com", "+0987654321", "12345678900")
	if !errors.Is(err, ErrInvalidCPF) {
		t.Errorf("expected ErrInvalidCPF, got %v", err)
	}
	if user.CPF != "52998224725" {
		t.Errorf("expected CPF unchanged on invalid update, got %s", user.CPF)
	}
}