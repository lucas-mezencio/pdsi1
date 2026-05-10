package firebaseauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"firebase.google.com/go/v4"
	firebaseadminauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
)

const signInWithPasswordEndpoint = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword"

type Service struct {
	authClient *firebaseadminauth.Client
	apiKey     string
	httpClient *http.Client
}

func NewService(ctx context.Context, credentialsFile, apiKey string) (*Service, error) {
	if strings.TrimSpace(credentialsFile) == "" || strings.TrimSpace(apiKey) == "" {
		return nil, application.ErrAuthNotConfigured
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("firebase init failed: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase auth init failed: %w", err)
	}

	return &Service{
		authClient: authClient,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (s *Service) CreateUser(ctx context.Context, email, password string) (string, error) {
	userRecord, err := s.authClient.CreateUser(ctx, (&firebaseadminauth.UserToCreate{}).
		Email(email).
		Password(password))
	if err != nil {
		if firebaseadminauth.IsEmailAlreadyExists(err) {
			return "", application.ErrEmailAlreadyInUse
		}
		return "", fmt.Errorf("firebase create user failed: %w", err)
	}

	return userRecord.UID, nil
}

func (s *Service) DeleteUser(ctx context.Context, firebaseID string) error {
	if strings.TrimSpace(firebaseID) == "" {
		return nil
	}

	if err := s.authClient.DeleteUser(ctx, firebaseID); err != nil {
		if firebaseadminauth.IsUserNotFound(err) {
			return nil
		}
		return fmt.Errorf("firebase delete user failed: %w", err)
	}
	return nil
}

func (s *Service) SignIn(ctx context.Context, email, password string) (string, error) {
	requestBody, err := json.Marshal(map[string]any{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode sign in request: %w", err)
	}

	endpoint := signInWithPasswordEndpoint + "?key=" + url.QueryEscape(s.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to build sign in request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("firebase sign in request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var payload struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		if isInvalidCredentialsError(payload.Error.Message) {
			return "", application.ErrAuthenticationFailed
		}
		if payload.Error.Message == "" {
			return "", errors.New("firebase sign in failed")
		}
		return "", fmt.Errorf("firebase sign in failed: %s", payload.Error.Message)
	}

	var payload struct {
		LocalID string `json:"localId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode firebase sign in response: %w", err)
	}
	if strings.TrimSpace(payload.LocalID) == "" {
		return "", errors.New("firebase sign in response missing localId")
	}
	return payload.LocalID, nil
}

func isInvalidCredentialsError(message string) bool {
	switch strings.ToUpper(strings.TrimSpace(message)) {
	case "INVALID_LOGIN_CREDENTIALS", "EMAIL_NOT_FOUND", "INVALID_PASSWORD", "USER_DISABLED":
		return true
	default:
		return false
	}
}
