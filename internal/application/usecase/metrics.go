package usecase

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Payment metrics
	paymentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_total",
			Help: "Total number of payment transactions",
		},
		[]string{"status"},
	)

	paymentDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_duration_seconds",
			Help:    "Payment processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)
)

// RecordPaymentMetric records payment metrics
func RecordPaymentMetric(status string, duration float64) {
	paymentTotal.WithLabelValues(status).Inc()
	paymentDuration.WithLabelValues(status).Observe(duration)
}
