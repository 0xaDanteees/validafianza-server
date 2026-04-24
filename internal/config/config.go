package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL        string
	Port               string
	Env                string
	CacheTTL           int
	RateLimitMax       int // max requests per window
	RateLimitWindowSec int // sliding window size in seconds
	AllowedOrigins     string
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s",
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "postgres"),
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_NAME", "validations"),
		)
	}

	cacheTTL, _ := strconv.Atoi(getEnv("CACHE_TTL_HOURS", "24"))
	rateLimitMax, _ := strconv.Atoi(getEnv("RATE_LIMIT_MAX", "60"))
	rateLimitWindow, _ := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW_SECONDS", "60"))

	return &Config{
		DatabaseURL:        dbURL,
		Port:               port,
		Env:                getEnv("ENV", "development"),
		CacheTTL:           cacheTTL,
		RateLimitMax:       rateLimitMax,
		RateLimitWindowSec: rateLimitWindow,
		AllowedOrigins:     getEnv("ALLOWED_ORIGINS", "*"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
