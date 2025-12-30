package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-api/internal/application/usecase"
	"payment-api/internal/config"
	"payment-api/internal/infrastructure/circuitbreaker"
	"payment-api/internal/infrastructure/database"
	"payment-api/internal/infrastructure/gateway"
	"payment-api/internal/infrastructure/persistence/postgres"
	"payment-api/internal/infrastructure/ratelimiter"
	"payment-api/internal/infrastructure/workerpool"
	"payment-api/internal/interface/http/handler"
	"payment-api/internal/interface/http/middleware"
	"payment-api/internal/interface/http/router"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting Payment API server")

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	logger.Info("Configuration loaded",
		zap.String("environment", cfg.Environment),
		zap.String("server_address", cfg.Server.Address),
		zap.Int("worker_count", cfg.WorkerPool.WorkerCount),
	)

	// Initialize database connection pool
	dbPool, err := database.NewConnectionPool(cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer dbPool.Close()

	// Initialize infrastructure components
	workerPool := workerpool.NewWorkerPool(cfg.WorkerPool, logger)
	defer workerPool.Shutdown(30 * time.Second)

	rateLimiter := ratelimiter.NewTokenBucketLimiter(
		cfg.RateLimit.RequestsPerSecond,
		cfg.RateLimit.BurstSize,
	)

	circuitBreaker := circuitbreaker.NewCircuitBreaker(cfg.CircuitBreaker)
	circuitBreaker.OnStateChange(func(from, to circuitbreaker.State) {
		logger.Warn("Circuit breaker state changed",
			zap.String("from", from.String()),
			zap.String("to", to.String()),
		)
	})

	// Initialize repositories and gateways
	paymentRepo := postgres.NewPaymentRepository(dbPool, logger)
	paymentGateway := gateway.NewMockPaymentGateway(logger)

	// Initialize use cases
	processPaymentUC := usecase.NewProcessPaymentUseCase(
		paymentRepo,
		paymentGateway,
		workerPool,
		circuitBreaker,
		logger,
	)
	getPaymentUC := usecase.NewGetPaymentUseCase(paymentRepo, logger)

	// Initialize HTTP handlers
	paymentHandler := handler.NewPaymentHandler(processPaymentUC, getPaymentUC, logger)

	// Initialize middleware
	rateLimitMW := middleware.NewRateLimitMiddleware(rateLimiter, logger)

	backpressureMW := middleware.NewBackpressureMiddleware(func() bool {
		// Check if worker pool is healthy
		return workerPool.IsHealthy() && dbPool.IsHealthy(context.Background())
	}, logger)

	timeoutMW := middleware.NewTimeoutMiddleware(30*time.Second, logger)
	recoveryMW := middleware.NewRecoveryMiddleware(logger)
	loggingMW := middleware.NewLoggingMiddleware(logger)

	// Create router
	r := router.NewRouter(router.Config{
		PaymentHandler:         paymentHandler,
		RateLimitMiddleware:    rateLimitMW,
		BackpressureMiddleware: backpressureMW,
		TimeoutMiddleware:      timeoutMW,
		RecoveryMiddleware:     recoveryMW,
		LoggingMiddleware:      loggingMW,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server listening", zap.String("address", cfg.Server.Address))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}

// initLogger initializes the logger based on environment
func initLogger() (*zap.Logger, error) {
	logLevel := os.Getenv("LOG_LEVEL")
	logFormat := os.Getenv("LOG_FORMAT")

	// Default to production logger
	if logFormat == "console" || os.Getenv("ENV") == "development" {
		cfg := zap.NewDevelopmentConfig()
		if logLevel != "" {
			var level zapcore.Level
			if err := level.UnmarshalText([]byte(logLevel)); err == nil {
				cfg.Level = zap.NewAtomicLevelAt(level)
			}
		}
		return cfg.Build()
	}

	cfg := zap.NewProductionConfig()
	if logLevel != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(logLevel)); err == nil {
			cfg.Level = zap.NewAtomicLevelAt(level)
		}
	}
	return cfg.Build()
}
