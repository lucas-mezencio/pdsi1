package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds application configuration values.
type Config struct {
	DatabaseURL              string
	HTTPAddr                 string
	RedisAddr                string
	NotifierMode             string
	FirebaseCredentialsFile  string
	FirebaseCredentialsJSON  string
	FirebaseWebAPIKey        string
	NotificationLookback     time.Duration
	DemoPrescriptionSecret   string
	LogFormat                string
	LogLevel                 string
}

const (
	defaultAddr                 = ":8080"
	defaultDSN                  = "postgres://mednotify:mednotify@localhost:5432/mednotify?sslmode=disable"
	defaultRedisAddr            = "localhost:6379"
	defaultNotifierMode         = "dev"
	defaultNotificationLookback = 2 * time.Hour
	defaultLogFormat            = "text"
	defaultLogLevel             = "info"
)

// Load loads configuration from .env (if present) and environment variables.
func Load() (*Config, error) {
	_ = loadDotEnv(".env")
	return &Config{
		DatabaseURL:              envString("DATABASE_URL", defaultDSN),
		HTTPAddr:                 envString("HTTP_ADDR", defaultAddr),
		RedisAddr:                envString("REDIS_ADDR", defaultRedisAddr),
		NotifierMode:             envString("NOTIFIER_MODE", defaultNotifierMode),
		FirebaseCredentialsFile:  envString("FIREBASE_CREDENTIALS_FILE", ""),
		FirebaseCredentialsJSON:  envString("FIREBASE_CREDENTIALS_JSON", ""),
		FirebaseWebAPIKey:        envString("FIREBASE_WEB_API_KEY", ""),
		NotificationLookback:     defaultNotificationLookback,
		DemoPrescriptionSecret:   envString("DEMO_PRESCRIPTION_SECRET", ""),
		LogFormat:                envString("LOG_FORMAT", defaultLogFormat),
		LogLevel:                 envString("LOG_LEVEL", defaultLogLevel),
	}, nil
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func loadDotEnv(path string) error {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}