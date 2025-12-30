package middleware

import (
	"context"
	"net/http"
	"time"

	"payment-api/internal/infrastructure/ratelimiter"

	"go.uber.org/zap"
)

// RateLimitMiddleware applies rate limiting to HTTP requests
type RateLimitMiddleware struct {
	limiter ratelimiter.RateLimiter
	logger  *zap.Logger
	keyFunc func(*http.Request) string // Function to extract rate limit key
}

// NewRateLimitMiddleware creates rate limit middleware
func NewRateLimitMiddleware(limiter ratelimiter.RateLimiter, logger *zap.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		logger:  logger,
		keyFunc: defaultKeyFunc,
	}
}

// defaultKeyFunc extracts user ID or IP address as key
func defaultKeyFunc(r *http.Request) string {
	// Try to get user ID from context (set by auth middleware)
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	// Fall back to IP address
	return r.RemoteAddr
}

// Handler returns HTTP middleware
func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := m.keyFunc(r)

		// Check rate limit
		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
		defer cancel()

		allowed, err := m.limiter.Allow(ctx, key)
		if err != nil {
			m.logger.Error("Rate limiter error", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			m.logger.Warn("Rate limit exceeded",
				zap.String("key", key),
				zap.String("path", r.URL.Path),
			)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// BackpressureMiddleware monitors system health and applies backpressure
type BackpressureMiddleware struct {
	healthChecker func() bool
	logger        *zap.Logger
}

// NewBackpressureMiddleware creates backpressure middleware
func NewBackpressureMiddleware(healthChecker func() bool, logger *zap.Logger) *BackpressureMiddleware {
	return &BackpressureMiddleware{
		healthChecker: healthChecker,
		logger:        logger,
	}
}

// Handler returns HTTP middleware
func (m *BackpressureMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check system health
		if !m.healthChecker() {
			m.logger.Warn("System unhealthy, applying backpressure",
				zap.String("path", r.URL.Path),
			)
			w.Header().Set("Retry-After", "30")
			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// TimeoutMiddleware adds timeout to requests
type TimeoutMiddleware struct {
	timeout time.Duration
	logger  *zap.Logger
}

// NewTimeoutMiddleware creates timeout middleware
func NewTimeoutMiddleware(timeout time.Duration, logger *zap.Logger) *TimeoutMiddleware {
	return &TimeoutMiddleware{
		timeout: timeout,
		logger:  logger,
	}
}

// Handler returns HTTP middleware
func (m *TimeoutMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), m.timeout)
		defer cancel()

		// Create channel to signal completion
		done := make(chan struct{})

		go func() {
			next.ServeHTTP(w, r.WithContext(ctx))
			close(done)
		}()

		select {
		case <-done:
			// Request completed
			return
		case <-ctx.Done():
			// Timeout occurred
			m.logger.Warn("Request timeout",
				zap.String("path", r.URL.Path),
				zap.Duration("timeout", m.timeout),
			)
			http.Error(w, "Request timeout", http.StatusGatewayTimeout)
			return
		}
	})
}

// RecoveryMiddleware recovers from panics
type RecoveryMiddleware struct {
	logger *zap.Logger
}

// NewRecoveryMiddleware creates recovery middleware
func NewRecoveryMiddleware(logger *zap.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: logger,
	}
}

// Handler returns HTTP middleware
func (m *RecoveryMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", r.URL.Path),
				)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests
type LoggingMiddleware struct {
	logger *zap.Logger
}

// NewLoggingMiddleware creates logging middleware
func NewLoggingMiddleware(logger *zap.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Handler returns HTTP middleware
func (m *LoggingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		m.logger.Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrapper.statusCode),
			zap.Duration("duration", duration),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
