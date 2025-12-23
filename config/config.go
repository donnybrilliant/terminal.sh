package config

import (
	"os"
	"strconv"
)

type Config struct {
	Host        string
	Port        int
	HostKeyPath string
	DatabasePath string
	JWTSecret    string
}

func Load() *Config {
	host := getEnv("HOST", "0.0.0.0")
	port := getEnvInt("PORT", 2222)
	hostKeyPath := getEnv("HOSTKEY_PATH", "")
	databasePath := getEnv("DATABASE_PATH", "ssh4xx.db")
	jwtSecret := getEnv("JWT_SECRET", "change-this-secret-key-in-production")

	return &Config{
		Host:        host,
		Port:        port,
		HostKeyPath: hostKeyPath,
		DatabasePath: databasePath,
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

