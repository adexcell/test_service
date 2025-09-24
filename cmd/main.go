package main

import (
	"context"
	"l0/cmd/app"
	_ "l0/docs"
	"l0/internal/config"
	"log"

	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Println(err.Error())
		return
	}

	logger := app.SetupLogger("local")

	var wg sync.WaitGroup

	sigQuit := make(chan os.Signal, 1)
	signal.Notify(sigQuit, os.Interrupt, syscall.SIGTERM)

	comp, err := app.InitComponents(ctx, cfg, logger)
	if err != nil {
		logger.Error("Bad configuration", slog.String("error", err.Error()))
		return
	}

	// Запускаем HTTP сервер
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := comp.HttpServer.Run(ctx); err != nil {
			logger.Error("failed to run HttpServer", slog.String("error", err.Error()))
		}
	}()

	// Запускаем Kafka consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := comp.KafkaConsumer.Consume(ctx); err != nil {
			logger.Error("Kafka consumer failed", slog.String("error", err.Error()))
		}
	}()

	// Ждём сигнал завершения
	<-sigQuit
	logger.Info("Received shutdown signal, stopping...")

	// Отменяем контекст, чтобы все горутины начали завершение
	cancel()

	// Вызываем shutdown для компонентов (если требуется)
	if err := comp.Shutdown(); err != nil {
		logger.Error("Error during shutdown", slog.String("error", err.Error()))
	}

	// Ждем завершения всех горутин
	wg.Wait()

	logger.Info("The program has exited")
}
