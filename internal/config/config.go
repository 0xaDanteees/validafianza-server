package config

import (
	"fmt"
	"os"
	"strconv"
)


type Config struct {
	DatabaseURL string
	Port		string
	CacheTTl	int
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port="3000"
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

	CacheTTl, _ := strconv.Atoi(getEnv("CACHE_TTL_HOURS", "24"))

	return &Config {
		DatabaseURL: dbURL,
		Port: port,
		CacheTTl: CacheTTl,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}