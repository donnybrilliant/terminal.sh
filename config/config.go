package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Host        string
	Port        int
	WebHost     string
	WebPort     int
	HostKeyPath string
	DatabasePath string
	DatabaseURL  string // For PostgreSQL support
	JWTSecret    string
}

// Load loads configuration from environment variables and .env file
// Environment variables take precedence over .env file values
// .env file is loaded for local development convenience
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

