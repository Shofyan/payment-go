package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"payment-api/internal/infrastructure/circuitbreaker"
	"payment-api/internal/infrastructure/database"
	"payment-api/internal/infrastructure/workerpool"
)

// Config holds all application configuration
type Config struct {
	Server         ServerConfig
	Database       database.Config
	Redis          RedisConfig
	WorkerPool     workerpool.Config
	RateLimit      RateLimitConfig
	CircuitBreaker circuitbreaker.Config
	Logging        LoggingConfig
	Monitoring     MonitoringConfig
	Environment    string
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	PrometheusEnabled bool
	MetricsPath       string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Address:      getEnv("SERVER_ADDRESS", ":8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: database.Config{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getIntEnv("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Database:        getEnv("DB_NAME", "payment_db"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getDurationEnv("DB_CONN_MAX_IDLE_TIME", 10*time.Minute),
			ConnectTimeout:  getDurationEnv("DB_CONNECT_TIMEOUT", 10*time.Second),
			QueryTimeout:    getDurationEnv("DB_QUERY_TIMEOUT", 30*time.Second),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getIntEnv("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		WorkerPool: workerpool.Config{
			WorkerCount:     getIntEnv("WORKER_COUNT", 50),
			QueueSize:       getIntEnv("QUEUE_SIZE", 1000),
			ShutdownTimeout: getDurationEnv("WORKER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: getFloatEnv("RATE_LIMIT_REQUESTS_PER_SECOND", 100.0),
			BurstSize:         getIntEnv("RATE_LIMIT_BURST_SIZE", 200),
		},
		CircuitBreaker: circuitbreaker.Config{
			MaxFailures:     getIntEnv("CIRCUIT_BREAKER_MAX_FAILURES", 5),
			Timeout:         getDurationEnv("CIRCUIT_BREAKER_TIMEOUT", 60*time.Second),
			HalfOpenSuccess: getIntEnv("CIRCUIT_BREAKER_HALF_OPEN_SUCCESS", 2),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Monitoring: MonitoringConfig{
			PrometheusEnabled: getBoolEnv("PROMETHEUS_ENABLED", true),
			MetricsPath:       getEnv("METRICS_PATH", "/metrics"),
		},
		Environment: getEnv("ENV", "development"),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Address == "" {
		return fmt.Errorf("SERVER_ADDRESS is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.WorkerPool.WorkerCount <= 0 {
		return fmt.Errorf("WORKER_COUNT must be positive")
	}
	if c.WorkerPool.QueueSize <= 0 {
		return fmt.Errorf("QUEUE_SIZE must be positive")
	}
	if c.RateLimit.RequestsPerSecond <= 0 {
		return fmt.Errorf("RATE_LIMIT_REQUESTS_PER_SECOND must be positive")
	}
	return nil
}

// IsDevelopment checks if environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction checks if environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Helper functions to get environment variables with defaults

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
