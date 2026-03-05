package user

import "testing"

func TestNewUser(t *testing.T) {
	tests := []struct {
		name          string
		userName      string
		email         string
		phone         string
		firebaseToken string
		wantErr       bool
	}{
		{
			name:          "valid user",
			userName:      "John Doe",
			email:         "john@example.com",
			phone:         "+1234567890",
			firebaseToken: "firebase-token-123",
			wantErr:       false,
		},
		{
			name:          "missing name",
			userName:      "",
			email:         "john@example.com",
			phone:         "+1234567890",
			firebaseToken: "firebase-token-123",
			wantErr:       true,
		},
		{
			name:          "missing email",
			userName:      "John Doe",
			email:         "",
			phone:         "+1234567890",
			firebaseToken: "firebase-token-123",
			wantErr:       true,
		},
		{
			name:          "missing phone",
			userName:      "John Doe",
			email:         "john@example.com",
			phone:         "",
			firebaseToken: "firebase-token-123",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.userName, tt.email, tt.phone, tt.firebaseToken, RoleElderly)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if user.ID == "" {
					t.Error("expected user ID to be generated")
				}
				if !user.NotificationsEnabled {
					t.Error("expected notifications to be enabled by default")
				}
				if user.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
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
	}

	err := user.Update("Jane Doe", "jane@example.com", "+0987654321")
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
}

func TestUser_EnableDisableNotifications(t *testing.T) {
	user := &User{
		ID:                   "test-id",
		NotificationsEnabled: true,
	}

	user.DisableNotifications()
	if user.NotificationsEnabled {
		t.Error("expected notifications to be disabled")
	}

	user.EnableNotifications()
	if !user.NotificationsEnabled {
		t.Error("expected notifications to be enabled")
	}
}

func TestUser_UpdateFirebaseToken(t *testing.T) {
	user := &User{
		ID:            "test-id",
		FirebaseToken: "old-token",
	}

	user.UpdateFirebaseToken("new-token")
	if user.FirebaseToken != "new-token" {
		t.Errorf("expected firebase token to be 'new-token', got %s", user.FirebaseToken)
	}
	if user.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}
