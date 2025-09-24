package service

import (
	"context"
	"encoding/json"
	"l0/internal/domain"
	"l0/pkg/e"
	"log/slog"
	"time"
)

//go:generate mockgen -source=service.go -destination=mocks/mock.go
type OrderRepository interface {
	GetByID(ctx context.Context, id int) (domain.Order, error)
	Create(ctx context.Context, order domain.Order) (int, error)
}

// Cache интерфейс кеша
type Cache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest *domain.Order) (string, error)
}

// Service бизнес-логика для заказов
type Service struct {
	repo   OrderRepository
	cache  Cache
	logger *slog.Logger
}

// NewService создаёт новый сервисный слой
func NewService(logger *slog.Logger, repo OrderRepository, cache Cache) *Service {
	return &Service{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

// GetOrderByID получает заказ по его ID из репозитория
func (s *Service) GetOrderByID(ctx context.Context, id int) (domain.Order, error) {
	return s.repo.GetByID(ctx, id)
}

// CreateOrder создаёт новый заказ через репозиторий
func (s *Service) CreateOrder(ctx context.Context, order domain.Order) (int, error) {
	id, err := s.repo.Create(ctx, order)
	if err != nil {
		s.logger.Error("Failed to create order", slog.String("error", err.Error()))
		return 0, e.Wrap("service.CreateOrder", err)
	}
	return id, nil
}

// MarshalOrderJSON преобразует заказ в JSON строку
func (s *Service) MarshalOrderJSON(order domain.Order) (string, error) {
	b, err := json.Marshal(order)
	if err != nil {
		return "", e.Wrap("service.MarshalOrderJSON", err)
	}
	return string(b), nil
}

// CacheSet сохраняет значение в кеш
func (s *Service) CacheSet(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return s.cache.Set(ctx, key, value, expiration)
}

// CacheGet получает значение из кеша
func (s *Service) CacheGet(ctx context.Context, key string, dest *domain.Order) (string, error) {
	return s.cache.Get(ctx, key, dest)
}
