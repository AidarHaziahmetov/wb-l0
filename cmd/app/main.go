package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	gin "github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"wb-l0-go/internal/cache"
	"wb-l0-go/internal/config"
	"wb-l0-go/internal/db"
	"wb-l0-go/internal/logger"
	"wb-l0-go/internal/repository"
	"wb-l0-go/internal/service"
	httpHandler "wb-l0-go/internal/transport/http"
	kafkaTransport "wb-l0-go/internal/transport/kafka"
)

// @title           WB L0 Go API
// @version         1.0
// @description     API for WB L0 Go project
// @host            localhost:8080
// @BasePath        /
func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Инициализируем логгер
	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	// Подключаемся к БД
	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Panic("failed to connect db", zap.Error(err))
	}
	defer pool.Close()

	// Инициализируем репозиторий и сервис
	repo := repository.NewPostgresOrderRepository(pool)
	memCache := cache.NewMemoryCache(cfg.CacheMaxItems)
	svc := service.NewOrderService(repo, memCache, log, pool)
	memCache.Load(ctx, svc)

	// Инициализируем Kafka producer и consumer
	producer := kafkaTransport.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic, log)
	consumer := kafkaTransport.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, svc, log)

	// Контекст graceful shutdown по сигналам
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Обработка второго сигнала — принудительный выход
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info("shutdown signal received")
		stop()
		<-sigCh
		log.Warn("second signal received, forcing exit")
		_ = log.Sync()
		os.Exit(1)
	}()

	// Запускаем consumer в отдельной горутине
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := consumer.Run(shutdownCtx); err != nil && err != context.Canceled {
			log.Error("consumer stopped with error", zap.Error(err))
			// Инициируем общий shutdown, чтобы корректно закрыть остальные компоненты
			stop()
		} else {
			log.Info("consumer stopped")
		}
	})

	// Инициализируем HTTP сервер
	r := gin.Default()
	h := httpHandler.NewHandler(svc, producer, log)
	h.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: r,
	}

	// Запуск HTTP сервера
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", zap.Error(err))
			// Инициируем общий shutdown при ошибке сервера
			stop()
		}
	}()

	// Ожидаем сигнал завершения
	<-shutdownCtx.Done()
	log.Info("shutting down")

	// Корректно останавливаем HTTP сервер (перестаёт принимать новые запросы и ждёт текущие)
	httpShutdownCtx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer httpCancel()
	if err := srv.Shutdown(httpShutdownCtx); err != nil {
		log.Error("http shutdown error", zap.Error(err))
	}

	// Дожидаемся завершения consumer
	wg.Wait()

	// Закрываем consumer
	if err := consumer.Close(); err != nil {
		log.Error("consumer close error", zap.Error(err))
	}

	// Закрываем producer (после остановки HTTP, чтобы не ломать обработчики)
	if err := producer.Close(); err != nil {
		log.Error("producer close error", zap.Error(err))
	}
}
