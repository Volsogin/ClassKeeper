package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Port           string
	Environment    string
	AllowedOrigins []string
}

type DatabaseConfig struct {
	Type     string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SQLPath  string
}

type JWTConfig struct {
	Secret              string
	AccessTokenExpiry   time.Duration
	RefreshTokenExpiry  time.Duration
}

func Load() *Config {
	// Загружаем .env файл (если существует)
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	return &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Environment: getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Type:     getEnv("DB_TYPE", "sqlite"),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "classkeeper"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "classkeeper_db"),
			SQLPath:  getEnv("SQLITE_PATH", "./classkeeper.db"),
		},
		JWT: JWTConfig{
			Secret:              getEnv("JWT_SECRET", "default-secret-key-change-me"),
			AccessTokenExpiry:   parseDuration(getEnv("JWT_EXPIRY", "15m")),
			RefreshTokenExpiry:  parseDuration(getEnv("REFRESH_TOKEN_EXPIRY", "168h")),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	duration, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("Invalid duration format for %s, using default", s)
		return 15 * time.Minute
	}
	return duration
}
