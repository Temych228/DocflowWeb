package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                    string
	AuthServiceAddr         string
	UserServiceAddr         string
	DocumentServiceAddr     string
	TaskServiceAddr         string
	CalendarServiceAddr     string
	NotificationServiceAddr string
	MailServiceAddr         string
	RedisHost               string
	RedisPort               string
	RedisPassword           string
	RedisDB                 int
	JWTSecret               string
	RateLimitRPM            int
}

func Load() *Config {
	cfg := &Config{
		Port:                    getEnv("PORT", "8080"),
		AuthServiceAddr:         getEnv("AUTH_SERVICE_ADDR", "localhost:9001"),
		UserServiceAddr:         getEnv("USER_SERVICE_ADDR", "localhost:9002"),
		DocumentServiceAddr:     getEnv("DOCUMENT_SERVICE_ADDR", "localhost:9003"),
		TaskServiceAddr:         getEnv("TASK_SERVICE_ADDR", "localhost:9004"),
		CalendarServiceAddr:     getEnv("CALENDAR_SERVICE_ADDR", "localhost:9005"),
		NotificationServiceAddr: getEnv("NOTIFICATION_SERVICE_ADDR", "localhost:9006"),
		MailServiceAddr:         getEnv("MAIL_SERVICE_ADDR", "localhost:9007"),
		RedisHost:               getEnv("REDIS_HOST", "localhost"),
		RedisPort:               getEnv("REDIS_PORT", "6379"),
		RedisPassword:           getEnv("REDIS_PASSWORD", ""),
		RedisDB:                 getEnvInt("REDIS_DB", 7),
		JWTSecret:               getEnv("JWT_SECRET", "your-secret-key"),
		RateLimitRPM:            getEnvInt("RATE_LIMIT_RPM", 100),
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}
