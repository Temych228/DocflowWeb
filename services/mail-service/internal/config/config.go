package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName     string
	AppPort         string
	GRPCPort        string
	MetricsPort     string
	DatabaseURL     string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	RedisHost       string
	RedisPort       string
	RedisPassword   string
	RedisDB         int
	CacheTTL        time.Duration
	DedupTTL        time.Duration
	NATSURL         string
	LogSubject      string
	MailJobsSubject string
	SMTPHost        string
	SMTPPort        string
	SMTPUsername    string
	SMTPPassword    string
	SMTPFrom        string
	SMTPUseTLS      bool
	SMTPUseStartTLS bool
	SMTPSkipVerify  bool
	SMTPTimeout     time.Duration
	CronEnabled     bool
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

	smtpPort := getEnv("SMTP_PORT", "587")
	smtpTLS := getEnv("SMTP_USE_TLS", "false") == "true"
	smtpStartTLS := getEnv("SMTP_USE_STARTTLS", "true") == "true"
	smtpSkipVerify := getEnv("SMTP_SKIP_VERIFY", "false") == "true"
	smtpTimeout, err := time.ParseDuration(getEnv("SMTP_TIMEOUT", "15s"))
	if err != nil {
		return nil, err
	}

	return &Config{
		ServiceName:     "mail-service",
		AppPort:         getEnv("APP_PORT", "8087"),
		GRPCPort:        getEnv("GRPC_PORT", "9007"),
		MetricsPort:     getEnv("METRICS_PORT", "9207"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", ""),
		DBName:          getEnv("DB_NAME", "postgres"),
		DBSSLMode:       getEnv("DB_SSLMODE", "require"),
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         redisDB,
		CacheTTL:        cacheTTL,
		DedupTTL:        dedupTTL,
		NATSURL:         getEnv("NATS_URL", ""),
		LogSubject:      getEnv("LOG_SUBJECT", "queue.logs"),
		MailJobsSubject: getEnv("MAIL_JOB_SUBJECT", "queue.mail.jobs"),
		SMTPHost:        getEnv("SMTP_HOST", ""),
		SMTPPort:        smtpPort,
		SMTPUsername:    getEnv("SMTP_USERNAME", ""),
		SMTPPassword:    getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:        getEnv("SMTP_FROM", ""),
		SMTPUseTLS:      smtpTLS,
		SMTPUseStartTLS: smtpStartTLS,
		SMTPSkipVerify:  smtpSkipVerify,
		SMTPTimeout:     smtpTimeout,
		CronEnabled:     getEnv("CRON_ENABLED", "false") == "true",
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
	return ":" + c.AppPort
}

func (c *Config) GRPCAddress() string {
	return ":" + c.GRPCPort
}

func (c *Config) MetricsAddress() string {
	return ":" + c.MetricsPort
}
