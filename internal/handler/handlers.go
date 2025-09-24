package handler

import (
	"context"
	"errors"
	"l0/internal/domain"
	"l0/internal/service"
	"l0/pkg/e"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Обертка для swagger ответа по заказу
type OrderResponse struct {
	Order domain.Order `json:"order"`
}

// Обертка для swagger ошибки
type ErrorResponse struct {
	Error string `json:"error"`
}

// @title OrderService App Api
// @version 1

type OrderRepository interface {
	GetByID(ctx context.Context, id int) (domain.Order, error)
	Create(ctx context.Context, order domain.Order) (int, error)
}

type Renderer interface {
	RenderHome(http.ResponseWriter)
}

type Handler struct {
	orderRepo OrderRepository
	cacheRepo service.Cache
	renderer  Renderer
	logger    *slog.Logger
}

func NewHandler(logger *slog.Logger, orderService service.OrderRepository, cacheService service.Cache, serviceRender Renderer) *Handler {
	return &Handler{
		orderRepo: orderService,
		cacheRepo: cacheService,
		logger:    logger,
		renderer:  serviceRender,
	}
}

// GetOrderByID godoc
// @Summary Получить заказ по ID
// @Description Возвращает заказ по уникальному идентификатору
// @Param id path int true "ID заказа"
// @Success 200 {object} handler.OrderResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Router /orders/{id} [get]
func (h *Handler) GetOrderByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Error("Invalid order id", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid order ID"})
		return
	}

	order, err := h.orderRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, e.ErrNotFound) {
			h.logger.Error("Order not found", slog.Int("id", id), slog.String("error", err.Error()))
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Order not found"})
			return
		}
		h.logger.Error("Failed to fetch order", slog.Int("id", id), slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, OrderResponse{Order: order})
}

// ShowHomepage отображает домашнюю страницу
func (h *Handler) ShowHomepage(c *gin.Context) {
	h.renderer.RenderHome(c.Writer)
}

// CreateOrder godoc
// @Summary Создать новый заказ
// @Description Создаёт заказ с переданными данными
// @Accept json
// @Produce json
// @Param order body domain.Order true "Данные заказа"
// @Success 201 {object} map[string]int "ID созданного заказа"
// @Failure 400 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Router /order [post]
func (h *Handler) CreateOrder(c *gin.Context) {
	var order domain.Order
	if err := c.BindJSON(&order); err != nil {
		h.logger.Error("Failed to bind order json", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid input"})
		return
	}

	id, err := h.orderRepo.Create(c.Request.Context(), order)
	if err != nil {
		h.logger.Error("Failed to create order", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create order"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"order_id": id})
}
