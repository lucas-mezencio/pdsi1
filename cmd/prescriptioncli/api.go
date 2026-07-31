package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const timeOffset = 3 * time.Hour

type Prescription struct {
	UserID    string
	MedicID   string
	Name      string
	Dosage    string
	Frequency string
	StartTime string
	Doses     int
}

type PrescriptionResponse struct {
	ID uuid.UUID `json:"id"`
}

type APIError struct {
	Status  int
	Message string
	Details string
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

type API struct {
	BaseURL string
	Secret  string
	HTTP    *http.Client
}

func New(baseURL, secret string) *API {
	return &API{BaseURL: baseURL, Secret: secret, HTTP: http.DefaultClient}
}

type loginResponse struct {
	ID string `json:"id"`
}

func (a *API) Login(ctx context.Context, email, pw string) (string, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": pw})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", decodeAPIError(resp.StatusCode, data)
	}

	var lr loginResponse
	if err := json.Unmarshal(data, &lr); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}
	if lr.ID == "" {
		return "", fmt.Errorf("login response missing id")
	}
	return lr.ID, nil
}

func (a *API) CreatePrescription(ctx context.Context, p Prescription) (*PrescriptionResponse, error) {
	shifted, err := shiftStartTime(p.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start time %q: %w", p.StartTime, err)
	}

	payload := map[string]any{
		"user_id":  p.UserID,
		"medic_id": p.MedicID,
		"medicaments": []map[string]any{
			{
				"name":      p.Name,
				"dosage":    p.Dosage,
				"frequency": p.Frequency,
				"time":      []string{shifted},
				"doses":     p.Doses,
			},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/prescriptions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build prescription request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", a.Secret)

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create prescription failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return nil, decodeAPIError(resp.StatusCode, data)
	}

	var pr PrescriptionResponse
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("decode prescription response: %w", err)
	}
	if pr.ID.String() == "" {
		return nil, fmt.Errorf("prescription response missing id")
	}
	return &pr, nil
}

func decodeAPIError(status int, body []byte) error {
	var env struct {
		Error   string `json:"error"`
		Details string `json:"details"`
	}
	_ = json.Unmarshal(body, &env)
	return &APIError{Status: status, Message: env.Error, Details: env.Details}
}

func shiftStartTime(in string) (string, error) {
	t, err := time.Parse("15:04", in)
	if err != nil {
		return "", err
	}
	return t.Add(timeOffset).Format("15:04"), nil
}
