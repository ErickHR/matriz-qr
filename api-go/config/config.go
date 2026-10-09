package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	StatisticsURL string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("cargar .env: %w", err)
	}

	return Config{
		Port:          getEnv("PORT", "3000"),
		StatisticsURL: getEnv("NODE_STATISTICS_URL", "http://localhost:3001/api/statistics"),
	}, nil
}

func getEnv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
