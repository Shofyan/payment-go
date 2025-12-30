package payment

import (
	"context"

	"github.com/google/uuid"
)

// Repository Interface - DDD Repository Pattern
type Repository interface {
	Save(ctx context.Context, payment *Payment) error
	FindByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*Payment, error)
	Update(ctx context.Context, payment *Payment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Payment Gateway Interface - External Service
type PaymentGateway interface {
	ProcessPayment(ctx context.Context, payment *Payment) (transactionID string, err error)
	VerifyPayment(ctx context.Context, transactionID string) (bool, error)
	RefundPayment(ctx context.Context, transactionID string, amount int64) error
}
