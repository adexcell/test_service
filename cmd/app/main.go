package app

import (
	"context"
	"fmt"
	"l0/internal/config"
	"l0/internal/handler"
	"l0/internal/kafka"
	"l0/internal/service"
	pg "l0/internal/storage/postgres"
	"l0/internal/storage/redis"
	"l0/pkg/logger"
	"log"
	"log/slog"
	"os"

	"github.com/IBM/sarama"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

type Components struct {
	HttpServer    *handler.Server
	Postgres      *pg.Postgres
	Redis         *redis.Redis
	KafkaConsumer *kafka.KafkaConsumer
}

func InitComponents(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Components, error) {

	redis, err := redis.NewRedis(&cfg.Redis, logger)
	if err != nil {
		logger.Error("redis error", "error", err.Error())
		return nil, fmt.Errorf("components.init.InitComponents.redis failed: %v", err)
	}

	postgres, err := pg.NewPostgres(ctx, cfg, logger, redis)
	if err != nil {
		logger.Error("postgres error", "error", err.Error())
		return nil, fmt.Errorf("components.init.InitComponents.postgres failed: %w", err)
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConfig.Consumer.Return.Errors = true

	orderService := service.NewService(logger, postgres, redis)

	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("failed to get current directory: %v", err)
		return nil, err
	}
	fmt.Println("Current work directory:", cwd)

	render := service.New(cwd+"/templates", logger)

	consumer, err := sarama.NewConsumer(cfg.Kafka.BrokerList, saramaConfig)
	if err != nil {
		logger.Error("components.init.InitComponents.consumer: failed to create consumer client", "error", err.Error())
		return nil, fmt.Errorf("components.init.InitComponent: consumer client failed to init: %w", err)
	}
	kafkaConsumer := kafka.NewKafkaConsumer(*cfg, logger, consumer, orderService)

	httpServer := handler.NewServer(ctx, cfg, logger, postgres, redis, render)

	return &Components{
		Postgres:      postgres,
		Redis:         redis,
		KafkaConsumer: kafkaConsumer,
		HttpServer:    httpServer,
	}, nil
}

func (c *Components) Shutdown() error {
	var errs []error
	c.Postgres.CloseConnection()
	if err := c.Redis.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close redis client: %w", err))
	}
	if err := c.KafkaConsumer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close kafka client: %w", err))
	}

	if err := c.HttpServer.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close Http Server: %v", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

func SetupLogger(s string) *slog.Logger {
	log := &slog.Logger{}

	switch "local" {
	case envLocal:
		log = logger.SetupPrettySlog()
	case "dev":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "prod":

		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
