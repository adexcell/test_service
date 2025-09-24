package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"l0/internal/config"
	"l0/internal/domain"
	"l0/pkg/e"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client redis.UniversalClient
	logger *slog.Logger
}

func NewRedis(config *config.RedisConfig, logger *slog.Logger) (*Redis, error) {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    config.Addrs,
		Password: "",
		DB:       0,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("redis.NewRedis failed: %w", err)
	}

	return &Redis{
		client: client,
		logger: logger,
	}, nil
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении в кэш: %v", err)
	}
	return r.client.Set(ctx, key, jsonValue, exp).Err()
}

func (r *Redis) Get(ctx context.Context, key string, value *domain.Order) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("the requested key is not found: %v", err)
	} else if err != nil {
		return "", fmt.Errorf("just errror %v", err)
	}

	if err := json.Unmarshal([]byte(result), value); err != nil {
		return "", fmt.Errorf("could not unmarshal(cache): %v", err)
	}

	return result, nil
}

func (r *Redis) Close() error {
	err := r.client.Close()
	if err != nil {
		r.logger.Error("storage.redis.Close", slog.String("error", err.Error()))
		return e.Wrap("failed to close redis", err)
	}
	return nil
}
