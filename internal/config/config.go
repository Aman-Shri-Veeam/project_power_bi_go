package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Settings holds all configuration for the application
type Settings struct {
	// Authentication
	PowerBIClientID     string
	PowerBIClientSecret string
	PowerBITenantID     string

	// API Configuration
	APIBaseURL   string
	Resource     string
	AuthorityURL string

	// Storage
	BackupPath string

	// Server
	Debug bool
}

var AppSettings *Settings

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() (*Settings, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	settings := &Settings{
		PowerBIClientID:     getEnv("POWERBI_CLIENT_ID", ""),
		PowerBIClientSecret: getEnv("POWERBI_CLIENT_SECRET", ""),
		PowerBITenantID:     getEnv("POWERBI_TENANT_ID", ""),
		APIBaseURL:          getEnv("API_BASE_URL", "https://api.powerbi.com/v1.0/myorg"),
		Resource:            "https://analysis.windows.net/powerbi/api",
		AuthorityURL:        "https://login.microsoftonline.com",
		BackupPath:          getEnv("BACKUP_PATH", "./backups"),
		Debug:               getEnv("DEBUG", "false") == "true",
	}

	AppSettings = settings
	return settings, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
