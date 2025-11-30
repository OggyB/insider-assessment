package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App struct {
		Name string
		Env  string
	}

	API struct {
		Host string
		Port string
	}

	DB struct {
		Host     string
		Port     int
		User     string
		Password string
		Name     string
		SSLMode  string
	}

	Redis struct {
		Addr     string
		Password string
		DB       int
	}

	SMS struct {
		ProviderURL string
		ProviderKey string
	}

	Scheduler struct {
		Interval     time.Duration
		BatchTimeout time.Duration
	}

	Worker struct {
		BatchSize         int
		MaxWorkers        int
		PerMessageTimeout time.Duration
	}
}

func New() *Config {
	_ = godotenv.Load()

	cfg := &Config{}

	// App
	cfg.App.Name = getEnv("APP_NAME", "kitabist")
	cfg.App.Env = getEnv("APP_ENV", "development")

	// API
	cfg.API.Host = getEnv("API_HOST", "0.0.0.0")
	cfg.API.Port = getEnv("API_PORT", "8080")

	// DB
	cfg.DB.Host = getEnv("DB_HOST", "db")
	cfg.DB.Port = getInt("DB_PORT", 5432)
	cfg.DB.User = getEnv("DB_USER", "root")
	cfg.DB.Password = getEnv("DB_PASSWORD", "123456")
	cfg.DB.Name = getEnv("DB_NAME", "db_ins_message")
	cfg.DB.SSLMode = getEnv("DB_SSLMODE", "disable")

	// Redis
	cfg.Redis.Addr = getEnv("REDIS_ADDR", "redis:6379")
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getInt("REDIS_DB", 0)

	// SMS Service
	cfg.SMS.ProviderURL = getEnv("SMS_PROVIDER_URL", "")
	cfg.SMS.ProviderKey = getEnv("SMS_PROVIDER_KEY", "")

	// Worker
	cfg.Scheduler.Interval = getDuration("SCHEDULER_INTERVAL", 5*time.Second)
	cfg.Scheduler.BatchTimeout = getDuration("SCHEDULER_BATCH_TIMEOUT", 30*time.Second)

	// Worker / message processing
	cfg.Worker.BatchSize = getInt("MESSAGE_BATCH_SIZE", 100)
	cfg.Worker.MaxWorkers = getInt("MESSAGE_MAX_WORKERS", 4)
	cfg.Worker.PerMessageTimeout = getDuration("MESSAGE_PER_MESSAGE_TIMEOUT", 5*time.Second)

	return cfg
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func getInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func getDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host,
		c.DB.Port,
		c.DB.User,
		c.DB.Password,
		c.DB.Name,
		c.DB.SSLMode,
	)
}
