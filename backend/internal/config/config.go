package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	AppEnv              string
	HTTP                HTTPConfig
	DB                  DatabaseConfig
	Redis               RedisConfig
	Kafka               KafkaConfig
	Worker              WorkerConfig
	Delivery            DeliveryConfig
	RateLimit           RateLimitConfig
	Log                 LogConfig
	Metrics             MetricsConfig
	ShutdownTimeout     time.Duration
	BootstrapAPIKey     string
	AllowPrivateTargets bool
}

type HTTPConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

type WorkerConfig struct {
	Count int
}

type DeliveryConfig struct {
	HTTPTimeout         time.Duration
	MaxPayloadSize      int64
	MaxResponseBodySize int64
	MaxRetries          int
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

type LogConfig struct {
	Level  string
	Format string
}

type MetricsConfig struct {
	Enabled bool
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv: getEnv("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Port:         getEnv("HTTP_PORT", "8080"),
			ReadTimeout:  getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		DB: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://hookforge:hookforge@localhost:5432/hookforge?sslmode=disable"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
		},
		Kafka: KafkaConfig{
			Brokers: splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
			Topic:   getEnv("KAFKA_TOPIC", "webhook.events"),
			GroupID: getEnv("KAFKA_GROUP_ID", "hookforge-workers"),
		},
		Worker: WorkerConfig{
			Count: getInt("WORKER_COUNT", 10),
		},
		Delivery: DeliveryConfig{
			HTTPTimeout:         getDuration("HTTP_TIMEOUT", 10*time.Second),
			MaxPayloadSize:      getInt64("MAX_PAYLOAD_SIZE", 1<<20),
			MaxResponseBodySize: getInt64("MAX_RESPONSE_BODY_SIZE", 64<<10),
			MaxRetries:          getInt("MAX_RETRIES", 5),
		},
		RateLimit: RateLimitConfig{
			Requests: getInt("RATE_LIMIT", 100),
			Window:   getDuration("RATE_LIMIT_WINDOW", time.Minute),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Metrics: MetricsConfig{
			Enabled: getBool("METRICS_ENABLED", true),
		},
		ShutdownTimeout:     getDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		BootstrapAPIKey:     os.Getenv("BOOTSTRAP_API_KEY"),
		AllowPrivateTargets: getBool("ALLOW_PRIVATE_TARGETS", false),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks that required configuration values are present and sane.
func (c *Config) Validate() error {
	if c.HTTP.Port == "" {
		return fmt.Errorf("HTTP_PORT is required")
	}
	if c.DB.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	if c.Kafka.Topic == "" {
		return fmt.Errorf("KAFKA_TOPIC is required")
	}
	if c.Worker.Count < 1 {
		return fmt.Errorf("WORKER_COUNT must be >= 1")
	}
	if c.Delivery.MaxRetries < 0 {
		return fmt.Errorf("MAX_RETRIES must be >= 0")
	}
	return nil
}

// IsDevelopment reports whether the app runs in development mode.
func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.AppEnv, "development") || strings.EqualFold(c.AppEnv, "dev")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func getBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
