package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"l0/internal/config"
	"l0/internal/domain"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-playground/validator/v10"
)

type DB interface {
	CreateOrder(ctx context.Context, order domain.Order) (int, error)
}

type KafkaConsumer struct {
	cfg          config.Config
	logger       *slog.Logger
	consumer     sarama.Consumer
	orderService DB
	validator    *validator.Validate
	errChan      chan error
	wg           sync.WaitGroup
	topic        string
}

func NewKafkaConsumer(cfg config.Config, logger *slog.Logger, consumer sarama.Consumer, orderService DB) *KafkaConsumer {
	return &KafkaConsumer{
		cfg:          cfg,
		logger:       logger,
		consumer:     consumer,
		orderService: orderService,
		validator:    validator.New(),
		errChan:      make(chan error, 10),
		topic:        cfg.Kafka.Topic,
	}
}

func (kc *KafkaConsumer) Consume(ctx context.Context) error {
	partitions, err := kc.consumer.Partitions(kc.topic)
	if err != nil {
		return fmt.Errorf("get partitions: %w", err)
	}

	var mu sync.Mutex
	var errs []error

	for _, partition := range partitions {
		pc, err := kc.consumer.ConsumePartition(kc.topic, partition, sarama.OffsetNewest)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("consume partition %d: %w", partition, err))
			mu.Unlock()
			kc.logger.Error("failed to consume partition", "partition", partition, "error", err)
			continue
		}

		kc.wg.Add(1)
		go kc.consumePartition(ctx, pc, partition, &mu, &errs)
	}

	kc.wg.Wait()
	close(kc.errChan)

	select {
	case e := <-kc.errChan:
		return e
	default:
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		if ctx.Err() == context.Canceled {
			kc.logger.Info("context canceled, consumer finished")
			return ctx.Err()
		}
		return nil
	}
}

func (kc *KafkaConsumer) consumePartition(
	ctx context.Context,
	pc sarama.PartitionConsumer,
	partition int32,
	mu *sync.Mutex,
	errs *[]error) {

	defer kc.wg.Done()
	defer func() {
		if err := pc.Close(); err != nil {
			kc.logger.Error("failed to close partition consumer", "partition", partition, "error", err)
			mu.Lock()
			*errs = append(*errs, err)
			mu.Unlock()
		}
	}()

	for {
		select {
		case msg, ok := <-pc.Messages():
			if !ok {
				kc.logger.Info("message channel closed", "partition", partition)
				return
			}

			var order domain.Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				kc.logger.Error("failed to unmarshal message", "error", err)
				continue
			}

			if err := kc.validator.Struct(order); err != nil {
				kc.logger.Error("validation failed", "error", err.Error())
				continue
			}

			var procErr error
			for attempt := 0; attempt <= kc.cfg.Kafka.MaxRetries; attempt++ {
				_, procErr = kc.orderService.CreateOrder(ctx, order)
				if procErr == nil {
					break
				}
				if attempt < kc.cfg.Kafka.MaxRetries {
					kc.logger.Warn("processing attempt failed",
						"attempt", attempt,
						"partition", partition,
						"error", procErr.Error())
					backoff := kc.cfg.Kafka.InitialBackoff * time.Duration(1<<attempt)
					select {
					case <-time.After(backoff):
					case <-ctx.Done():
						kc.logger.Info("context canceled during backoff", "partition", partition)
						return
					}
				}
			}

			if procErr != nil {
				mu.Lock()
				*errs = append(*errs, procErr)
				mu.Unlock()
			}

		case err, ok := <-pc.Errors():
			if !ok {
				kc.logger.Info("error channel closed", "partition", partition)
				return
			}
			kc.logger.Error("partition consumer error", "error", err.Err)
			mu.Lock()
			*errs = append(*errs, err.Err)
			mu.Unlock()
			select {
			case kc.errChan <- fmt.Errorf("partition consumer error: %w", err.Err):
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			kc.logger.Info("context canceled, shutting down partition consumer", "partition", partition)
			return
		}
	}
}

func (kc *KafkaConsumer) Close() error {
	return kc.consumer.Close()
}

func (kc *KafkaConsumer) GetError() error {
	select {
	case err := <-kc.errChan:
		return err
	default:
		return nil
	}
}
