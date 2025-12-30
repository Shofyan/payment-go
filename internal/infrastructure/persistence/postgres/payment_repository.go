package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payment-api/internal/domain/payment"
	"payment-api/internal/infrastructure/database"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PaymentRepository implements payment.Repository using PostgreSQL
type PaymentRepository struct {
	pool   *database.ConnectionPool
	logger *zap.Logger
}

// NewPaymentRepository creates a new PostgreSQL repository
func NewPaymentRepository(pool *database.ConnectionPool, logger *zap.Logger) *PaymentRepository {
	return &PaymentRepository{
		pool:   pool,
		logger: logger,
	}
}

// Save saves a new payment
func (r *PaymentRepository) Save(ctx context.Context, pmt *payment.Payment) error {
	query := `
		INSERT INTO payments (
			id, user_id, merchant_id, amount, currency, method, status, 
			transaction_id, created_at, updated_at, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.pool.ExecContext(ctx, query,
		pmt.ID(),
		pmt.UserID(),
		pmt.MerchantID(),
		pmt.Amount().Amount,
		pmt.Amount().Currency,
		pmt.Method(),
		pmt.Status(),
		pmt.TransactionID(),
		pmt.CreatedAt(),
		time.Now(),
		pmt.Version(),
	)

	if err != nil {
		return fmt.Errorf("failed to save payment: %w", err)
	}

	return nil
}

// FindByID finds payment by ID
func (r *PaymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*payment.Payment, error) {
	query := `
		SELECT id, user_id, merchant_id, amount, currency, method, status, 
			   transaction_id, created_at, updated_at, version
		FROM payments
		WHERE id = $1
	`

	row := r.pool.QueryRowContext(ctx, query, id)

	var (
		paymentID     uuid.UUID
		userID        string
		merchantID    string
		amount        int64
		currency      string
		method        payment.PaymentMethod
		status        payment.PaymentStatus
		transactionID string
		createdAt     time.Time
		updatedAt     time.Time
		version       int
	)

	err := row.Scan(
		&paymentID, &userID, &merchantID, &amount, &currency, &method,
		&status, &transactionID, &createdAt, &updatedAt, &version,
	)

	if err == sql.ErrNoRows {
		return nil, payment.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find payment: %w", err)
	}

	// Reconstruct payment aggregate (simplified)
	pmt, _ := payment.NewPayment(userID, merchantID, amount, currency, method)

	return pmt, nil
}

// FindByUserID finds payments by user ID
func (r *PaymentRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*payment.Payment, error) {
	query := `
		SELECT id, user_id, merchant_id, amount, currency, method, status, 
			   transaction_id, created_at, updated_at, version
		FROM payments
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find payments: %w", err)
	}
	defer rows.Close()

	var payments []*payment.Payment
	for rows.Next() {
		var (
			paymentID     uuid.UUID
			userID        string
			merchantID    string
			amount        int64
			currency      string
			method        payment.PaymentMethod
			status        payment.PaymentStatus
			transactionID string
			createdAt     time.Time
			updatedAt     time.Time
			version       int
		)

		err := rows.Scan(
			&paymentID, &userID, &merchantID, &amount, &currency, &method,
			&status, &transactionID, &createdAt, &updatedAt, &version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}

		pmt, _ := payment.NewPayment(userID, merchantID, amount, currency, method)
		payments = append(payments, pmt)
	}

	return payments, nil
}

// Update updates an existing payment
func (r *PaymentRepository) Update(ctx context.Context, pmt *payment.Payment) error {
	query := `
		UPDATE payments
		SET status = $2, transaction_id = $3, updated_at = $4, version = $5
		WHERE id = $1 AND version = $6
	`

	result, err := r.pool.ExecContext(ctx, query,
		pmt.ID(),
		pmt.Status(),
		pmt.TransactionID(),
		time.Now(),
		pmt.Version(),
		pmt.Version()-1, // Optimistic locking
	)

	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("payment was modified by another transaction")
	}

	return nil
}

// Delete deletes a payment
func (r *PaymentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM payments WHERE id = $1`

	_, err := r.pool.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete payment: %w", err)
	}

	return nil
}
