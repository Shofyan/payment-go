package gateway

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"payment-api/internal/domain/payment"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MockPaymentGateway simulates external payment gateway
// In production, this would integrate with Stripe, PayPal, etc.
type MockPaymentGateway struct {
	logger      *zap.Logger
	failureRate float64 // Simulate failures
	latency     time.Duration
}

// NewMockPaymentGateway creates a mock gateway
func NewMockPaymentGateway(logger *zap.Logger) *MockPaymentGateway {
	return &MockPaymentGateway{
		logger:      logger,
		failureRate: 0.1, // 10% failure rate
		latency:     100 * time.Millisecond,
	}
}

// ProcessPayment simulates payment processing
func (g *MockPaymentGateway) ProcessPayment(ctx context.Context, pmt *payment.Payment) (string, error) {
	// Simulate network latency
	select {
	case <-time.After(g.latency):
	case <-ctx.Done():
		return "", ctx.Err()
	}

	// Simulate random failures
	if rand.Float64() < g.failureRate {
		g.logger.Warn("Payment gateway failure simulated")
		return "", errors.New("payment gateway error")
	}

	// Generate transaction ID
	transactionID := fmt.Sprintf("TXN_%s", uuid.New().String()[:8])

	g.logger.Info("Payment processed by gateway",
		zap.String("payment_id", pmt.ID().String()),
		zap.String("transaction_id", transactionID),
	)

	return transactionID, nil
}

// VerifyPayment verifies a payment
func (g *MockPaymentGateway) VerifyPayment(ctx context.Context, transactionID string) (bool, error) {
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		return false, ctx.Err()
	}

	// Simulate verification
	return true, nil
}

// RefundPayment refunds a payment
func (g *MockPaymentGateway) RefundPayment(ctx context.Context, transactionID string, amount int64) error {
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	g.logger.Info("Payment refunded",
		zap.String("transaction_id", transactionID),
		zap.Int64("amount", amount),
	)

	return nil
}
