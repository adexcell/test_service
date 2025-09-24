package handler

import (
	"l0/internal/domain"
	mock_handler "l0/internal/handler/mocks"
	mock_service "l0/internal/service/mocks"
	"l0/pkg/e"

	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func setupRouter(logger *slog.Logger, mockRepo *mock_handler.MockOrderRepository, mockCache *mock_service.MockCache, mockRenderer *mock_handler.MockRenderer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(logger, mockRepo, mockCache, mockRenderer)
	r := gin.New()
	r.GET("/orders/:id", h.GetOrderByID)
	r.POST("/orders", h.CreateOrder)
	r.GET("/", h.ShowHomepage)
	return r
}

func TestHandler_GetOrderByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_handler.NewMockOrderRepository(ctrl)
	mockCache := mock_service.NewMockCache(ctrl)
	mockRenderer := mock_handler.NewMockRenderer(ctrl)
	logger := slog.Default()

	order := domain.Order{OrderUID: "abc123"}

	mockRepo.EXPECT().GetByID(gomock.Any(), 1).Return(order, nil)

	r := setupRouter(logger, mockRepo, mockCache, mockRenderer)

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"order_uid\":\"abc123\"")
}

func TestHandler_GetOrderByID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_handler.NewMockOrderRepository(ctrl)
	mockCache := mock_service.NewMockCache(ctrl)
	mockRenderer := mock_handler.NewMockRenderer(ctrl)
	logger := slog.Default()

	mockRepo.EXPECT().GetByID(gomock.Any(), 1).Return(domain.Order{}, e.ErrNotFound) // change ErrNotFound as per your package.

	r := setupRouter(logger, mockRepo, mockCache, mockRenderer)

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Order not found")
}

func TestHandler_CreateOrder_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_handler.NewMockOrderRepository(ctrl)
	mockCache := mock_service.NewMockCache(ctrl)
	mockRenderer := mock_handler.NewMockRenderer(ctrl)
	logger := slog.Default()

	orderJSON := `{"order_uid": "abc123"}`

	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(1, nil)

	r := setupRouter(logger, mockRepo, mockCache, mockRenderer)

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(orderJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "order_id")
}

func TestHandler_CreateOrder_BindError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_handler.NewMockOrderRepository(ctrl)
	mockCache := mock_service.NewMockCache(ctrl)
	mockRenderer := mock_handler.NewMockRenderer(ctrl)
	logger := slog.Default()

	r := setupRouter(logger, mockRepo, mockCache, mockRenderer)

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader("invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid input")
}

func TestHandler_ShowHomepage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_handler.NewMockOrderRepository(ctrl)
	mockCache := mock_service.NewMockCache(ctrl)
	mockRenderer := mock_handler.NewMockRenderer(ctrl)
	logger := slog.Default()

	mockRenderer.EXPECT().RenderHome(gomock.Any()).DoAndReturn(func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("homepage"))
	})

	r := setupRouter(logger, mockRepo, mockCache, mockRenderer)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "homepage", w.Body.String())
}
