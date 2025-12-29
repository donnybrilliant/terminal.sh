// Package config provides configuration management for the terminal.sh server.
// It loads settings from environment variables and optional .env files.
package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the application configuration including server ports, database settings, and JWT secret.
type Config struct {
	Host         string // SSH server host (default: "0.0.0.0")
	Port         int    // SSH server port (default: 2222)
	WebHost      string // Web server host (default: same as Host)
	WebPort      int    // Web server port (default: 8080)
	HostKeyPath  string // Path to SSH host key file (optional)
	DatabasePath string // Path to SQLite database file (default: "data/terminal.db")
	DatabaseURL  string // PostgreSQL connection URL (optional, takes precedence over DatabasePath)
	JWTSecret    string // Secret key for JWT token signing
}

// Load loads configuration from environment variables and .env file.
// Environment variables take precedence over .env file values.
// The .env file is loaded for local development convenience and is optional.
// Returns a Config instance with defaults applied for any missing values.
func Load() *Config {
	// Load .env file if it exists (ignore errors - it's optional)
	// This is useful for local development
	if err := godotenv.Load(); err != nil {
		// .env file doesn't exist or can't be read - that's fine
		// We'll use environment variables or defaults
	}

	host := getEnv("HOST", "0.0.0.0")
	port := getEnvInt("PORT", 2222)
	webHost := getEnv("WEB_HOST", host) // Default to same as SSH host
	webPort := getEnvInt("WEB_PORT", 8080)
	hostKeyPath := getEnv("HOSTKEY_PATH", "")
	databasePath := getEnv("DATABASE_PATH", "data/terminal.db") // Default: data/terminal.db
	databaseURL := getEnv("DATABASE_URL", "") // For PostgreSQL support
	jwtSecret := getEnv("JWT_SECRET", "change-this-secret-key-in-production")

	return &Config{
		Host:        host,
		Port:        port,
		WebHost:     webHost,
		WebPort:     webPort,
		HostKeyPath: hostKeyPath,
		DatabasePath: databasePath,
		DatabaseURL:  databaseURL,
		JWTSecret:    jwtSecret,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

