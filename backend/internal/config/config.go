package config

import (
	"os"
	"strings"
)

// Load reads Open Context server configuration from environment variables.
func Load() Config {
	return Config{
		HTTPAddr:              getEnv("OPEN_CONTEXT_HTTP_ADDR", ":8000"),
		APIKey:                getEnv("OPEN_CONTEXT_API_KEY", "changeme"),
		PostgresDSN:           getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/opencontext?sslmode=disable"),
		GraphitiURL:           strings.TrimRight(getEnv("GRAPHITI_SERVICE_URL", "http://localhost:8003"), "/"),
		ProjectUUID:           getEnv("OPEN_CONTEXT_PROJECT_UUID", "00000000-0000-4000-8000-000000000001"),
		OpenContextName:       getEnv("OPEN_CONTEXT_PROJECT_NAME", "open-context"),
		OpenContextVersion:    getEnv("OPEN_CONTEXT_VERSION", "0.1.0"),
	}
}

type Config struct {
	HTTPAddr           string
	APIKey             string
	PostgresDSN        string
	GraphitiURL        string
	ProjectUUID        string
	OpenContextName    string
	OpenContextVersion string
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

