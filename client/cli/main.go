package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com.br/lucas-mezencio/pdsi1/client"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	tokenFile    = ".cc_token"
	envAPIDomain = "CC_API_DOMAIN"
	defaultDomain = "localhost:8080"
)

var (
	storedToken string
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: cc <command> [args]\nCommands: register, login, logout, delete")
	}

	loadToken()

	switch os.Args[1] {
	case "register":
		return register(os.Args[2:])
	case "login":
		return login(os.Args[2:])
	case "logout":
		return logout()
	case "delete":
		return delete(os.Args[2:])
	default:
		return fmt.Errorf("unknown command: %s\nUsage: cc <command>\nCommands: register, login, logout, delete", os.Args[1])
	}
}

func getAPIURL() string {
	if domain := os.Getenv(envAPIDomain); domain != "" {
		if strings.HasPrefix(domain, "http") {
			return domain
		}
		if domain == "localhost:8080" {
			return "http://" + domain + "/api/v1"
		}
		return "https://" + domain + "/api/v1"
	}
	return "http://" + defaultDomain + "/api/v1"
}

func getClient() (*client.Client, error) {
	apiURL := getAPIURL()
	c, err := client.NewClient(apiURL,
		client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			if storedToken != "" {
				req.Header.Set("Authorization", "Bearer "+storedToken)
			}
			return nil
		}),
	)
	return c, err
}

func loadToken() {
	usr, _ := user.Current()
	if usr == nil {
		return
	}
	path := filepath.Join(usr.HomeDir, tokenFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	storedToken = strings.TrimSpace(string(data))
}

func saveToken(token string) error {
	usr, err := user.Current()
	if err != nil {
		return err
	}
	path := filepath.Join(usr.HomeDir, tokenFile)
	return os.WriteFile(path, []byte(token), 0600)
}

func clearToken() {
	usr, _ := user.Current()
	if usr == nil {
		return
	}
	path := filepath.Join(usr.HomeDir, tokenFile)
	_ = os.Remove(path)
}


// register creates a new user with default test values
func register(args []string) error {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	name := fs.String("name", "Test User", "user name")
	email := fs.String("email", "test@example.com", "user email")
	phone := fs.String("phone", "+1234567890", "user phone")
	password := fs.String("password", "password123", "user password")
	_ = fs.Parse(args)

	c, err := getClient()
	if err != nil {
		return fmt.Errorf("create client failed: %w", err)
	}
	resp, err := c.Register(context.Background(), client.RegisterRequest{
		Name:     *name,
		Email:    openapi_types.Email(*email),
		Phone:    *phone,
		Password: *password,
	})
	if err != nil {
		return fmt.Errorf("register failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("register failed with status: %d", resp.StatusCode)
	}

	var authResp client.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}

	storedToken = authResp.Token
	_ = saveToken(authResp.Token)

	fmt.Printf("User registered successfully!\n")
	fmt.Printf("Name: %s\n", authResp.User.Name)
	fmt.Printf("Email: %s\n", authResp.User.Email)
	fmt.Printf("ID: %s\n", authResp.User.Id)
	fmt.Printf("Token: %s\n", authResp.Token)
	return nil
}

// login authenticates a user and stores the token
func login(args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	email := fs.String("email", "", "user email (required)")
	password := fs.String("password", "", "user password (required)")
	_ = fs.Parse(args)

	if *email == "" || *password == "" {
		return fmt.Errorf("email and password are required")
	}

	c, err := getClient()
	if err != nil {
		return fmt.Errorf("create client failed: %w", err)
	}
	resp, err := c.Login(context.Background(), client.LoginRequest{
		Email:    openapi_types.Email(*email),
		Password: *password,
	})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid credentials")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var authResp client.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}

	storedToken = authResp.Token
	_ = saveToken(authResp.Token)

	fmt.Printf("Login successful!\n")
	fmt.Printf("User: %s (%s)\n", authResp.User.Name, authResp.User.Email)
	fmt.Printf("Token: %s\n", authResp.Token)
	return nil
}

// logout invalidates the stored token
func logout() error {
	if storedToken == "" {
		return fmt.Errorf("not logged in (no token stored)")
	}

	c, err := getClient()
	if err != nil {
		return fmt.Errorf("create client failed: %w", err)
	}
	resp, err := c.Logout(context.Background())
	if err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}
	defer resp.Body.Close()

	clearToken()
	storedToken = ""

	fmt.Println("Logged out successfully!")
	return nil
}

// delete removes a user by ID
func delete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	_ = fs.Parse(args)

	if storedToken == "" {
		return fmt.Errorf("not logged in (required to delete user)")
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("user ID is required")
	}
	userID := fs.Arg(0)

	c, err := getClient()
	if err != nil {
		return fmt.Errorf("create client failed: %w", err)
	}
	ctx := context.Background()
	uid, _ := uuid.Parse(userID)
	resp, err := c.DeleteUser(ctx, client.UserId(uid))
	if err != nil {
		return fmt.Errorf("delete user failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("user not found: %s", userID)
	}
	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized")
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("delete failed with status: %d", resp.StatusCode)
	}

	clearToken()
	storedToken = ""

	fmt.Printf("User %s deleted successfully!\n", userID)
	return nil
}