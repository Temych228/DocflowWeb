package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppPort     string
	GRPCPort    string
	MetricsPort string

	DatabaseURL string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	CacheTTL      time.Duration

	NotificationServiceAddr string
	CalendarServiceAddr     string
}

func Load() (*Config, error) {
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	cacheTTL, err := time.ParseDuration(getEnv("CACHE_TTL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid CACHE_TTL: %w", err)
	}

	cfg := &Config{
		AppPort:                 getEnv("APP_PORT", "8083"),
		GRPCPort:                getEnv("GRPC_PORT", "9003"),
		MetricsPort:             getEnv("METRICS_PORT", "9203"),
		DatabaseURL:             getEnv("DATABASE_URL", ""),
		DBHost:                  getEnv("DB_HOST", "localhost"),
		DBPort:                  getEnv("DB_PORT", "5432"),
		DBUser:                  getEnv("DB_USER", "postgres"),
		DBPassword:              getEnv("DB_PASSWORD", ""),
		DBName:                  getEnv("DB_NAME", "postgres"),
		DBSSLMode:               getEnv("DB_SSLMODE", "require"),
		RedisHost:               getEnv("REDIS_HOST", "localhost"),
		RedisPort:               getEnv("REDIS_PORT", "6379"),
		RedisPassword:           getEnv("REDIS_PASSWORD", ""),
		RedisDB:                 redisDB,
		CacheTTL:                cacheTTL,
		NotificationServiceAddr: getEnv("NOTIFICATION_SERVICE_ADDR", ""),
		CalendarServiceAddr:     getEnv("CALENDAR_SERVICE_ADDR", ""),
	}

	return cfg, nil
}

func (c *Config) Address() string {
	return fmt.Sprintf(":%s", c.AppPort)
}

func (c *Config) GRPCAddress() string {
	return fmt.Sprintf(":%s", c.GRPCPort)
}

func (c *Config) MetricsAddress() string {
	return fmt.Sprintf(":%s", c.MetricsPort)
}

func (c *Config) PostgresDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
