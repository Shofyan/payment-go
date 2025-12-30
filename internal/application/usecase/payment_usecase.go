package usecase

import (
	"context"
	"fmt"
	"time"

	"payment-api/internal/domain/payment"
	"payment-api/internal/infrastructure/circuitbreaker"
	"payment-api/internal/infrastructure/workerpool"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProcessPaymentUseCase handles payment processing business logic
type ProcessPaymentUseCase struct {
	repo           payment.Repository
	gateway        payment.PaymentGateway
	workerPool     *workerpool.WorkerPool
	circuitBreaker *circuitbreaker.CircuitBreaker
	logger         *zap.Logger
}

// ProcessPaymentRequest represents payment processing request
type ProcessPaymentRequest struct {
	UserID     string
	MerchantID string
	Amount     int64
	Currency   string
	Method     payment.PaymentMethod
}

// ProcessPaymentResponse represents payment processing response
type ProcessPaymentResponse struct {
	PaymentID     uuid.UUID
	Status        payment.PaymentStatus
	TransactionID string
	CreatedAt     time.Time
}

// NewProcessPaymentUseCase creates a new use case
func NewProcessPaymentUseCase(
	repo payment.Repository,
	gateway payment.PaymentGateway,
	workerPool *workerpool.WorkerPool,
	circuitBreaker *circuitbreaker.CircuitBreaker,
	logger *zap.Logger,
) *ProcessPaymentUseCase {
	return &ProcessPaymentUseCase{
		repo:           repo,
		gateway:        gateway,
		workerPool:     workerPool,
		circuitBreaker: circuitBreaker,
		logger:         logger,
	}
}

// Execute processes a payment request
func (uc *ProcessPaymentUseCase) Execute(ctx context.Context, req ProcessPaymentRequest) (*ProcessPaymentResponse, error) {
	// Create payment aggregate
	pmt, err := payment.NewPayment(req.UserID, req.MerchantID, req.Amount, req.Currency, req.Method)
	if err != nil {
		uc.logger.Error("Failed to create payment", zap.Error(err))
		return nil, fmt.Errorf("invalid payment: %w", err)
	}

	// Save payment (pending state)
	if err := uc.repo.Save(ctx, pmt); err != nil {
		uc.logger.Error("Failed to save payment", zap.Error(err))
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	// Process payment asynchronously using worker pool
	// This prevents blocking the HTTP handler and provides backpressure
	err = uc.workerPool.Submit(ctx, func(taskCtx context.Context) error {
		return uc.processPaymentAsync(taskCtx, pmt)
	})

	if err != nil {
		// Worker pool is full - apply backpressure
		uc.logger.Warn("Worker pool full, rejecting request",
			zap.String("payment_id", pmt.ID().String()),
		)
		return nil, fmt.Errorf("system overloaded, please try again: %w", err)
	}

	return &ProcessPaymentResponse{
		PaymentID: pmt.ID(),
		Status:    pmt.Status(),
		CreatedAt: pmt.CreatedAt(),
	}, nil
}

// processPaymentAsync handles actual payment processing
func (uc *ProcessPaymentUseCase) processPaymentAsync(ctx context.Context, pmt *payment.Payment) error {
	// Add timeout to prevent hanging
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()

	uc.logger.Info("Processing payment",
		zap.String("payment_id", pmt.ID().String()),
		zap.String("user_id", pmt.UserID()),
	)

	// Use circuit breaker to prevent cascading failures
	var transactionID string
	err := uc.circuitBreaker.Execute(ctx, func() error {
		var gatewayErr error
		transactionID, gatewayErr = uc.gateway.ProcessPayment(ctx, pmt)
		return gatewayErr
	})

	if err != nil {
		// Check if circuit breaker is open
		if err == circuitbreaker.ErrCircuitOpen {
			uc.logger.Warn("Circuit breaker open, payment gateway unavailable",
				zap.String("payment_id", pmt.ID().String()),
			)
			// Implement fallback: queue for retry, return cached response, etc.
			pmt.Fail("payment gateway unavailable")
		} else {
			uc.logger.Error("Payment gateway failed",
				zap.String("payment_id", pmt.ID().String()),
				zap.Error(err),
			)
			pmt.Fail(err.Error())
		}

		// Update payment status
		if updateErr := uc.repo.Update(ctx, pmt); updateErr != nil {
			uc.logger.Error("Failed to update payment", zap.Error(updateErr))
		}

		// Record metrics
		duration := time.Since(start).Seconds()
		RecordPaymentMetric("failed", duration)

		return err
	}

	// Payment successful
	if err := pmt.Process(transactionID); err != nil {
		uc.logger.Error("Failed to process payment", zap.Error(err))
		return err
	}

	if err := pmt.Complete(); err != nil {
		uc.logger.Error("Failed to complete payment", zap.Error(err))
		return err
	}

	// Update payment status
	if err := uc.repo.Update(ctx, pmt); err != nil {
		uc.logger.Error("Failed to update payment", zap.Error(err))
		return err
	}

	uc.logger.Info("Payment processed successfully",
		zap.String("payment_id", pmt.ID().String()),
		zap.String("transaction_id", transactionID),
	)

	// Record metrics
	duration := time.Since(start).Seconds()
	RecordPaymentMetric("completed", duration)

	return nil
}

// GetPaymentUseCase retrieves payment information
type GetPaymentUseCase struct {
	repo   payment.Repository
	logger *zap.Logger
}

// NewGetPaymentUseCase creates a new use case
func NewGetPaymentUseCase(repo payment.Repository, logger *zap.Logger) *GetPaymentUseCase {
	return &GetPaymentUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute retrieves a payment by ID
func (uc *GetPaymentUseCase) Execute(ctx context.Context, id uuid.UUID) (*payment.Payment, error) {
	pmt, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error("Failed to find payment", zap.Error(err))
		return nil, err
	}
	return pmt, nil
}
