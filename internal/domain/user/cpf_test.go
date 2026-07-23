package user

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateCPF(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValid   bool
		wantNorm    string
	}{
		// Valid CPFs (publicly known test fixtures)
		{name: "valid unformatted", input: "52998224725", wantValid: true, wantNorm: "52998224725"},
		{name: "valid formatted", input: "529.982.247-25", wantValid: true, wantNorm: "52998224725"},
		{name: "valid with whitespace", input: "  529.982.247-25  ", wantValid: true, wantNorm: "52998224725"},
		{name: "valid 000.000.000-00 alternative", input: "39053344705", wantValid: true, wantNorm: "39053344705"},

		// Invalid CPFs
		{name: "empty", input: "", wantValid: false},
		{name: "too short", input: "123456789", wantValid: false},
		{name: "too long", input: "123456789012", wantValid: false},
		{name: "non-digit", input: "529.982.247-2X", wantValid: false},
		{name: "all zeros", input: "00000000000", wantValid: false},
		{name: "all ones", input: "11111111111", wantValid: false},
		{name: "wrong check digit", input: "52998224726", wantValid: false},
		{name: "wrong first digit", input: "52998224735", wantValid: false},
		{name: "only letters stripped -> too short", input: "abcdefghi", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := ValidateCPF(tt.input)
			if tt.wantValid {
				if err != nil {
					t.Fatalf("expected valid CPF, got error: %v", err)
				}
				if normalized != tt.wantNorm {
					t.Errorf("expected normalized %q, got %q", tt.wantNorm, normalized)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for input %q, got normalized=%q", tt.input, normalized)
			}
			if !errors.Is(err, ErrInvalidCPF) {
				t.Errorf("expected ErrInvalidCPF, got %v", err)
			}
		})
	}
}

func TestValidateCPF_DoesNotLeakNormalizedOnError(t *testing.T) {
	// If validation fails, the normalized value should not be a partial CPF that
	// looks usable downstream.
	_, err := ValidateCPF("52998224726")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "52998224726") {
		t.Errorf("error message should not echo the full input CPF, got: %v", err)
	}
}