package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env      string `env:"ENV"`
	Http     HTTPConfig
	Redis    RedisConfig
	Postgres PostgresConfig
	Kafka    KafkaConfig
}

type HTTPConfig struct {
	Port string `env:"HTTP_PORT"`
}

type RedisConfig struct {
	Addrs    []string `env:"REDIS_ADDRS"`
	Password string   `env:"REDIS_PASSWORD"`
	DBRedis  int      `env:"REDIS_DB"`
}

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST"`
	Port     string `env:"POSTGRES_PORT"`
	Database string `env:"POSTGRES_DATABASE"`
	User     string `env:"POSTGRES_USER"`
	Password string `env:"POSTGRES_PASSWORD"`
	SSLMode  string `env:"POSTGRES_SSL_MODE"`
}

type KafkaConfig struct {
	BrokerList     []string      `env:"KAFKA_BROKER_LIST"`
	Topic          string        `env:"KAFKA_TOPIC"`
	InitialBackoff time.Duration `env:"KAFKA_INITIAL_BACKOFF"`
	MaxRetries     int           `env:"KAFKA_MAX_RETRIES"`
	ConsumerGroup  string        `env:"KAFKA_CONSUMER_GROUP"`
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		// .env может отсутствовать в некоторых окружениях, не обязательно ошибку делать
	}

	cfg := &Config{}

	cfg.Env = os.Getenv("ENV")

	cfg.Http.Port = os.Getenv("HTTP_PORT")

	redisAddrs := os.Getenv("REDIS_ADDRS")
	if redisAddrs != "" {
		cfg.Redis.Addrs = splitAndTrim(redisAddrs, ",")
	}
	cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")
	dbRedisStr := os.Getenv("REDIS_DB")
	if dbRedisStr != "" {
		if dbIndex, err := strconv.Atoi(dbRedisStr); err == nil {
			cfg.Redis.DBRedis = dbIndex
		}
	}

	cfg.Postgres.Host = os.Getenv("POSTGRES_HOST")
	cfg.Postgres.Port = os.Getenv("POSTGRES_PORT")
	cfg.Postgres.Database = os.Getenv("POSTGRES_DATABASE")
	cfg.Postgres.User = os.Getenv("POSTGRES_USER")
	cfg.Postgres.Password = os.Getenv("POSTGRES_PASSWORD")
	cfg.Postgres.SSLMode = os.Getenv("POSTGRES_SSL_MODE")

	kafkaBrokers := os.Getenv("KAFKA_BROKER_LIST")
	if kafkaBrokers != "" {
		cfg.Kafka.BrokerList = splitAndTrim(kafkaBrokers, ",")
	}
	cfg.Kafka.Topic = os.Getenv("KAFKA_TOPIC")

	if backoffStr := os.Getenv("KAFKA_INITIAL_BACKOFF"); backoffStr != "" {
		if d, err := time.ParseDuration(backoffStr); err == nil {
			cfg.Kafka.InitialBackoff = d
		}
	}
	if maxRetriesStr := os.Getenv("KAFKA_MAX_RETRIES"); maxRetriesStr != "" {
		if retries, err := strconv.Atoi(maxRetriesStr); err == nil {
			cfg.Kafka.MaxRetries = retries
		}
	}
	cfg.Kafka.ConsumerGroup = os.Getenv("KAFKA_CONSUMER_GROUP")

	return cfg, nil
}

func splitAndTrim(str, sep string) []string {
	parts := strings.Split(str, sep)
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
