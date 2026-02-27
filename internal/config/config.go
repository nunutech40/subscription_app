package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application.
type Config struct {
	// Server
	Port    string
	GinMode string

	// Database
	DatabaseURL string

	// Auth
	JWTSecret              string
	JWTExpiry              string
	RefreshTokenExpiryDays int

	// Midtrans
	MidtransServerKey string
	MidtransClientKey string
	MidtransBaseURL   string

	// Email
	ResendAPIKey string
	FromEmail    string
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPass     string

	// CORS
	CORSOrigins string

	// Frontend
	FrontendURL string

	// Admin
	AdminSecretKey string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:    getEnv("PORT", "8080"),
		GinMode: getEnv("GIN_MODE", "debug"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		JWTSecret:              getEnv("JWT_SECRET", ""),
		JWTExpiry:              getEnv("JWT_EXPIRY", "1h"),
		RefreshTokenExpiryDays: getEnvInt("REFRESH_TOKEN_EXPIRY_DAYS", 30),

		MidtransServerKey: getEnv("MIDTRANS_SERVER_KEY", ""),
		MidtransClientKey: getEnv("MIDTRANS_CLIENT_KEY", ""),
		MidtransBaseURL:   getEnv("MIDTRANS_BASE_URL", "https://app.sandbox.midtrans.com"),

		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		FromEmail:    getEnv("FROM_EMAIL", "noreply@sains.id"),
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPass:     getEnv("SMTP_PASS", ""),

		CORSOrigins: getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"),

		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),

		AdminSecretKey: getEnv("ADMIN_SECRET_KEY", ""),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
