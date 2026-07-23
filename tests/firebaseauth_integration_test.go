//go:build integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"firebase.google.com/go/v4"
	firebaseadminauth "firebase.google.com/go/v4/auth"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/database"
	"github.com.br/lucas-mezencio/pdsi1/tests/testcontainers"
	"google.golang.org/api/option"
)

func TestFirebaseAuthLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. Start postgres container using testcontainers
	db, dsn, cleanup := testcontainers.StartPostgresContainer(ctx)
	if db == nil {
		t.Skip("docker not available")
	}
	defer cleanup()

	// 2. Run migrations
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// 3. Start API subprocess with env vars
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	firebaseCreds := os.Getenv("FIREBASE_CREDENTIALS_FILE")
	if firebaseCreds == "" {
		home, _ := os.UserHomeDir()
		firebaseCreds = home + "/.credentials/firabase-careconnect.json"
	}

	apiCmd := exec.CommandContext(ctx, "go", "run", "./cmd/api")
	apiCmd.Dir = ".."
	apiCmd.Stdout = os.Stdout
	apiCmd.Stderr = os.Stderr
	apiCmd.Env = append(os.Environ(),
		"DATABASE_URL="+dsn,
		"REDIS_ADDR="+redisAddr,
		"FIREBASE_CREDENTIALS_FILE="+firebaseCreds,
		"HTTP_ADDR=:8080",
	)

	if err := apiCmd.Start(); err != nil {
		t.Fatalf("start api failed: %v", err)
	}

	// 9. CLEANUP: ensure API subprocess is killed on exit
	var apiExited bool
	defer func() {
		if !apiExited {
			_ = apiCmd.Process.Kill()
		}
		_ = apiCmd.Wait()
	}()

	// 4. Wait for API to be ready
	apiURL := "http://localhost:8080/health"
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled while waiting for API")
		default:
		}

		resp, err := http.Get(apiURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}
		if i == 29 {
			t.Fatalf("API not ready after 30 attempts")
		}
		time.Sleep(time.Second)
	}

	// Helper to make HTTP requests
	baseURL := "http://localhost:8080/api/v1/auth"
	makeRequest := func(method, path string, body any) (*http.Response, []byte, error) {
		var reqBody []byte
		if body != nil {
			reqBody, _ = json.Marshal(body)
		}
		req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(reqBody))
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return resp, nil, err
		}
		respBody, _ := json.Marshal(map[string]interface{}{})
		if resp.Body != nil {
			defer resp.Body.Close()
			respBody, _, _ = decodeJSONResp(resp.Body)
		}
		return resp, respBody, nil
	}

	// 5. REGISTER: POST /api/v1/auth/register
	registerBody := map[string]interface{}{
		"name":    "Test User",
		"email":   "test@example.com",
		"cpf":     "12345678900",
		"phone":   "+5511999999999",
		"password": "password123",
		"role":    "ELDERLY",
	}
	regResp, regBody, err := makeRequest(http.MethodPost, "/register", registerBody)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. body: %s", regResp.StatusCode, string(regBody))
	}

	var registerResult struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		CPF   string `json:"cpf"`
		Phone string `json:"phone"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(regBody, &registerResult); err != nil {
		t.Fatalf("failed to unmarshal register response: %v", err)
	}

	if registerResult.ID == "" {
		t.Fatal("expected non-empty id in register response")
	}
	if registerResult.Name != "Test User" {
		t.Fatalf("expected name 'Test User', got %s", registerResult.Name)
	}
	if registerResult.Email != "test@example.com" {
		t.Fatalf("expected email 'test@example.com', got %s", registerResult.Email)
	}
	if registerResult.Phone != "+5511999999999" {
		t.Fatalf("expected phone '+5511999999999', got %s", registerResult.Phone)
	}
	if registerResult.Role != "ELDERLY" {
		t.Fatalf("expected role 'ELDERLY', got %s", registerResult.Role)
	}

	// Query DB directly to verify
	dbRow := db.QueryRowContext(ctx, "SELECT email, cpf, firebase_uid FROM users WHERE id = $1", registerResult.ID)
	var dbEmail, dbCPF, dbFirebaseUID string
	if err := dbRow.Scan(&dbEmail, &dbCPF, &dbFirebaseUID); err != nil {
		t.Fatalf("failed to query DB: %v", err)
	}
	if dbEmail != "test@example.com" {
		t.Fatalf("expected DB email 'test@example.com', got %s", dbEmail)
	}
	if dbFirebaseUID == "" {
		t.Fatal("expected non-empty firebase_uid in DB")
	}

	// 6. LOGIN: POST /api/v1/auth/login
	loginReqBody := map[string]interface{}{
		"email":    "test@example.com",
		"password": "password123",
	}
	loginResp, loginRespBody, err := makeRequest(http.MethodPost, "/login", loginReqBody)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d. body: %s", loginResp.StatusCode, string(loginRespBody))
	}

	var loginResult struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginRespBody, &loginResult); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}
	if loginResult.ID != registerResult.ID {
		t.Fatalf("expected login id '%s', got '%s'", registerResult.ID, loginResult.ID)
	}

	// 7. LOGOUT: POST /api/v1/auth/logout
	logoutResp, logoutBody, err := makeRequest(http.MethodPost, "/logout", nil)
	if err != nil {
		t.Fatalf("logout request failed: %v", err)
	}
	if logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", logoutResp.StatusCode)
	}
	var logoutResult struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(logoutBody, &logoutResult); err != nil {
		t.Fatalf("failed to unmarshal logout response: %v", err)
	}
	if logoutResult.Message != "logged out successfully" {
		t.Fatalf("expected message 'logged out successfully', got '%s'", logoutResult.Message)
	}

	// 8. DELETE FIREBASE USER: Use Firebase Admin SDK to delete the user by firebase_uid
	firebaseUID := dbFirebaseUID

	if firebaseUID != "" {
		app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(firebaseCreds))
		if err != nil {
			t.Logf("WARNING: failed to create firebase app: %v", err)
		} else {
			firebaseAuth, err := app.Auth(ctx)
			if err != nil {
				t.Logf("WARNING: failed to get firebase auth: %v", err)
			} else {
				if err := firebaseAuth.DeleteUser(ctx, firebaseUID); err != nil {
					if !firebaseadminauth.IsUserNotFound(err) {
						t.Logf("WARNING: failed to delete firebase user: %v", err)
					}
				}
			}
		}
	}

	// 9. CLEANUP: Kill API subprocess, close DB connection
	apiExited = true
	_ = apiCmd.Process.Kill()
	_ = apiCmd.Wait()
}

func decodeJSONResp(body io.ReadCloser) ([]byte, map[string]interface{}, error) {
	respBody, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(respBody, &m); err != nil {
		return nil, nil, err
	}
	return respBody, m, nil
}
