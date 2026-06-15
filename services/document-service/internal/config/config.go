package config

import (
	"fmt"
	"os"
)

type Config struct {
	GRPCPort string

	DatabaseURL string

	NotificationServiceAddr string
	CalendarServiceAddr     string
}

func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:                getEnv("GRPC_PORT", "9003"),
		DatabaseURL:             getEnv("DATABASE_URL", ""),
		NotificationServiceAddr: getEnv("NOTIFICATION_SERVICE_ADDR", ""),
		CalendarServiceAddr:     getEnv("CALENDAR_SERVICE_ADDR", ""),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
