package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName         string
	AppPort             string
	GRPCPort            string
	MetricsPort         string
	DatabaseURL         string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	DBSSLMode           string
	RedisHost           string
	RedisPort           string
	RedisPassword       string
	RedisDB             int
	CacheTTL            time.Duration
	DedupTTL            time.Duration
	NATSURL             string
	LogSubject          string
	MailJobSubject      string
	NotificationSubject string
	UserServiceAddr     string
	CronEnabled         bool
}

func Load() (*Config, error) {
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, err
	}

	cacheTTL, err := time.ParseDuration(getEnv("CACHE_TTL", "5m"))
	if err != nil {
		return nil, err
	}

	dedupTTL, err := time.ParseDuration(getEnv("DEDUP_TTL", "24h"))
	if err != nil {
		return nil, err
	}

	return &Config{
		ServiceName:         "notification-service",
		AppPort:             getEnv("APP_PORT", "8086"),
		GRPCPort:            getEnv("GRPC_PORT", "9006"),
		MetricsPort:         getEnv("METRICS_PORT", "9206"),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", ""),
		DBName:              getEnv("DB_NAME", "postgres"),
		DBSSLMode:           getEnv("DB_SSLMODE", "require"),
		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnv("REDIS_PORT", "6379"),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		RedisDB:             redisDB,
		CacheTTL:            cacheTTL,
		DedupTTL:            dedupTTL,
		NATSURL:             getEnv("NATS_URL", ""),
		LogSubject:          getEnv("LOG_SUBJECT", "queue.logs"),
		MailJobSubject:      getEnv("MAIL_JOB_SUBJECT", "queue.mail.jobs"),
		NotificationSubject: getEnv("NOTIFICATION_SUBJECT", "queue.notification.events"),
		UserServiceAddr:     getEnv("USER_SERVICE_ADDR", ""),
		CronEnabled:         getEnv("CRON_ENABLED", "true") == "true",
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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

func (c *Config) Address() string {
	return fmt.Sprintf(":%s", c.AppPort)
}

func (c *Config) GRPCAddress() string {
	return fmt.Sprintf(":%s", c.GRPCPort)
}

func (c *Config) MetricsAddress() string {
	return fmt.Sprintf(":%s", c.MetricsPort)
}
