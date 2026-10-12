package config

import "os"

// Config holds process configuration from environment.
type Config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string
}

// Load reads configuration from environment with local-dev defaults.
func Load() Config {
	return Config{
		AppEnv:      getenv("APP_ENV", "development"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://trello:trello_dev_password@localhost:5432/trello_analytics?sslmode=disable"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
