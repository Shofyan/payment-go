package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status", "instance"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "instance"},
	)
)

// MetricsMiddleware collects Prometheus metrics for HTTP requests
type MetricsMiddleware struct {
	instance string
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(instance string) *MetricsMiddleware {
	return &MetricsMiddleware{
		instance: instance,
	}
}

// Handler returns HTTP middleware that collects metrics
func (m *MetricsMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start).Seconds()

		// Get route pattern from chi context
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = r.URL.Path
		}

		// Record metrics
		httpRequestsTotal.WithLabelValues(
			r.Method,
			routePattern,
			strconv.Itoa(wrapper.statusCode),
			m.instance,
		).Inc()

		httpRequestDuration.WithLabelValues(
			r.Method,
			routePattern,
			m.instance,
		).Observe(duration)
	})
}
