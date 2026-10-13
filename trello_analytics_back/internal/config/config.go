package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds process configuration from environment.
type Config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string

	// FrontendOrigin is the allowed CORS origin (cookie auth requires explicit origin + credentials).
	FrontendOrigin string

	// SessionTTL is absolute session lifetime (default 7d).
	SessionTTL time.Duration

	// Bootstrap Admin (one-shot when app_users is empty).
	BootstrapAdminEmail    string
	BootstrapAdminPassword string

	// Trello credentials — env only; never persist in app_settings / DB.
	// Used by cmd/trello-sync and HTTP sync (dual entrypoint).
	TrelloAPIKey  string
	TrelloToken   string
	TrelloBoardID string
	// TrelloActionsPageSize defaults to 100; clamped to 1..1000.
	TrelloActionsPageSize int
}

// Load reads configuration from environment with local-dev defaults.
func Load() Config {
	return Config{
		AppEnv:                 getenv("APP_ENV", "development"),
		HTTPAddr:               getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:            getenv("DATABASE_URL", "postgres://trello:trello_dev_password@localhost:5432/trello_analytics?sslmode=disable"),
		FrontendOrigin:         getenv("FRONTEND_ORIGIN", "http://localhost:5173"),
		SessionTTL:             getenvDuration("SESSION_TTL", 7*24*time.Hour),
		BootstrapAdminEmail:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		TrelloAPIKey:           os.Getenv("TRELLO_API_KEY"),
		TrelloToken:            os.Getenv("TRELLO_TOKEN"),
		TrelloBoardID:          os.Getenv("TRELLO_BOARD_ID"),
		TrelloActionsPageSize:  getenvIntClamped("TRELLO_ACTIONS_PAGE_SIZE", 100, 1, 1000),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvIntClamped(key string, fallback, min, max int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
