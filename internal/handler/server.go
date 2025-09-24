package handler

import (
	"context"
	"fmt"
	"l0/internal/config"

	"l0/internal/service"
	"l0/pkg/e"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	logger *slog.Logger
	server *http.Server
	cfg    *config.Config
}

func NewServer(ctx context.Context, config *config.Config, logger *slog.Logger, orderService service.OrderRepository, cacheService service.Cache, serviceRender Renderer) *Server {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.Http.Port),
		Handler: InitRouter(ctx, logger, orderService, cacheService, serviceRender),
	}

	return &Server{
		logger: logger,
		server: server,
		cfg:    config,
	}
}

func InitRouter(ctx context.Context, logger *slog.Logger, orderService service.OrderRepository, cacheService service.Cache, serviceRender Renderer) *gin.Engine {
	r := gin.Default()

	h := NewHandler(logger, orderService, cacheService, serviceRender)
	docsURL := ginSwagger.URL("http://localhost:8080/swagger/doc.json")
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:8080"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowCredentials = true

	r.Use(cors.New(config))

	r.GET("/", h.ShowHomepage)
	r.GET("/orders/:id", h.GetOrderByID)
	r.POST("/order", h.CreateOrder)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, docsURL))

	return r
}

func (s *Server) Run(ctx context.Context) error {
	errResult := make(chan error, 1)
	go func() {
		s.logger.Info("starting listinening", slog.String("address", s.server.Addr))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errResult <- fmt.Errorf("http server failed: %w", err)
		} else if err == http.ErrServerClosed {
			s.logger.Info("HTTP server stopped gracefully")
			errResult <- nil
		}

	}()

	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down server due to context cancellation")
		if err := s.Stop(); err != nil {
			return e.Wrap("failed to stop HttpServer gracefully", err)
		}
		return ctx.Err()
	case err := <-errResult:
		return err
	}
}

func (s *Server) Stop() error {
	shutDownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err := s.server.Shutdown(shutDownCtx)
	s.logger.Info("Shutting down HTTP server")
	if err != nil {
		s.logger.Error("failed to shutdown HTTP Server", slog.String("error", err.Error()))
		return err
	}
	s.logger.Info("HTTP server shut down successfully")
	return nil
}
