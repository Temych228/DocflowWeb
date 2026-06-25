package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort     string
	GRPCPort    string
	MetricsPort string

	DatabaseURL string

	NotificationServiceAddr string
	CalendarServiceAddr     string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppPort:                 getEnv("APP_PORT", "8084"),
		GRPCPort:                getEnv("GRPC_PORT", "9004"),
		MetricsPort:             getEnv("METRICS_PORT", "9204"),
		DatabaseURL:             getEnv("DATABASE_URL", ""),
		NotificationServiceAddr: getEnv("NOTIFICATION_SERVICE_ADDR", ""),
		CalendarServiceAddr:     getEnv("CALENDAR_SERVICE_ADDR", ""),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func (c *Config) Address() string {
	return fmt.Sprintf(":%s", c.AppPort)
}

func (c *Config) MetricsAddress() string {
	return fmt.Sprintf(":%s", c.MetricsPort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
