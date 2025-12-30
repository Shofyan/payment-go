package router

import (
	"payment-api/internal/interface/http/handler"
	"payment-api/internal/interface/http/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds router configuration
type Config struct {
	PaymentHandler         *handler.PaymentHandler
	RateLimitMiddleware    *middleware.RateLimitMiddleware
	BackpressureMiddleware *middleware.BackpressureMiddleware
	TimeoutMiddleware      *middleware.TimeoutMiddleware
	RecoveryMiddleware     *middleware.RecoveryMiddleware
	LoggingMiddleware      *middleware.LoggingMiddleware
}

// NewRouter creates a new HTTP router with all routes
func NewRouter(cfg Config) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware (order matters!)
	r.Use(cfg.RecoveryMiddleware.Handler)     // Recover from panics
	r.Use(cfg.LoggingMiddleware.Handler)      // Log requests
	r.Use(chimiddleware.RequestID)            // Add request ID
	r.Use(chimiddleware.RealIP)               // Get real IP
	r.Use(cfg.TimeoutMiddleware.Handler)      // Request timeout
	r.Use(cfg.BackpressureMiddleware.Handler) // System health check
	r.Use(cfg.RateLimitMiddleware.Handler)    // Rate limiting

	// Health endpoints (no rate limiting)
	r.Group(func(r chi.Router) {
		r.Get("/health", cfg.PaymentHandler.HealthCheck)
		r.Get("/ready", cfg.PaymentHandler.ReadinessCheck)
	})

	// Metrics endpoint (for Prometheus)
	r.Handle("/metrics", promhttp.Handler())

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Payment endpoints
		r.Route("/payments", func(r chi.Router) {
			r.Post("/", cfg.PaymentHandler.CreatePayment)
			r.Get("/{id}", cfg.PaymentHandler.GetPayment)
		})
	})

	return r
}
