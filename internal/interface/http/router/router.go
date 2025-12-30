package router

import (
	"net/http"
	"payment-api/internal/interface/http/handler"
	"payment-api/internal/interface/http/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds router configuration
type Config struct {
	PaymentHandler         *handler.PaymentHandler
	WebHandler             *handler.WebHandler
	RateLimitMiddleware    *middleware.RateLimitMiddleware
	BackpressureMiddleware *middleware.BackpressureMiddleware
	TimeoutMiddleware      *middleware.TimeoutMiddleware
	RecoveryMiddleware     *middleware.RecoveryMiddleware
	LoggingMiddleware      *middleware.LoggingMiddleware
	MetricsMiddleware      *middleware.MetricsMiddleware
}

// NewRouter creates a new HTTP router with all routes
func NewRouter(cfg Config) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware (order matters!)
	r.Use(cfg.RecoveryMiddleware.Handler)     // Recover from panics
	r.Use(cfg.LoggingMiddleware.Handler)      // Log requests
	r.Use(cfg.MetricsMiddleware.Handler)      // Collect metrics
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

	// Web interface routes
	r.Route("/web", func(r chi.Router) {
		// Serve static files
		r.Handle("/static/*", http.StripPrefix("/web/static/", http.FileServer(http.Dir("web/static"))))

		// Web pages
		r.Get("/", cfg.WebHandler.IndexPage)
		r.Get("/create", cfg.WebHandler.CreatePaymentPage)
		r.Get("/get", cfg.WebHandler.GetPaymentPage)

		// HTMX endpoints
		r.Post("/payments/create", cfg.WebHandler.CreatePaymentForm)
		r.Post("/payments/get", cfg.WebHandler.GetPaymentForm)
		r.Get("/payments/{id}", cfg.WebHandler.GetPaymentByID)
	})

	// Root redirects to web interface
	r.Get("/", cfg.WebHandler.IndexPage)

	// Static files at root level
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	return r
}
