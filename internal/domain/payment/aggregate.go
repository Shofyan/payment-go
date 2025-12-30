package payment

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Payment Aggregate - DDD Aggregate Root
type Payment struct {
	id            uuid.UUID
	userID        string
	amount        Money
	status        PaymentStatus
	method        PaymentMethod
	merchantID    string
	transactionID string
	metadata      map[string]string
	createdAt     time.Time
	updatedAt     time.Time
	version       int // For optimistic locking
}

// Value Objects
type Money struct {
	Amount   int64  // Amount in smallest currency unit (e.g., cents)
	Currency string // ISO 4217
}

type PaymentStatus string

const (
	StatusPending    PaymentStatus = "PENDING"
	StatusProcessing PaymentStatus = "PROCESSING"
	StatusCompleted  PaymentStatus = "COMPLETED"
	StatusFailed     PaymentStatus = "FAILED"
	StatusCancelled  PaymentStatus = "CANCELLED"
)

type PaymentMethod string

const (
	MethodCreditCard   PaymentMethod = "CREDIT_CARD"
	MethodDebitCard    PaymentMethod = "DEBIT_CARD"
	MethodWallet       PaymentMethod = "WALLET"
	MethodBankTransfer PaymentMethod = "BANK_TRANSFER"
)

// Domain Errors
var (
	ErrInvalidAmount           = errors.New("invalid payment amount")
	ErrInvalidCurrency         = errors.New("invalid currency")
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrPaymentAlreadyProcessed = errors.New("payment already processed")
	ErrInsufficientFunds       = errors.New("insufficient funds")
	ErrPaymentTimeout          = errors.New("payment timeout")
)

// Factory Method
func NewPayment(userID, merchantID string, amount int64, currency string, method PaymentMethod) (*Payment, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if currency == "" {
		return nil, ErrInvalidCurrency
	}

	return &Payment{
		id:         uuid.New(),
		userID:     userID,
		merchantID: merchantID,
		amount: Money{
			Amount:   amount,
			Currency: currency,
		},
		method:    method,
		status:    StatusPending,
		metadata:  make(map[string]string),
		createdAt: time.Now(),
		updatedAt: time.Now(),
		version:   1,
	}, nil
}

// Business Logic Methods
func (p *Payment) Process(transactionID string) error {
	if p.status != StatusPending {
		return ErrPaymentAlreadyProcessed
	}
	p.status = StatusProcessing
	p.transactionID = transactionID
	p.updatedAt = time.Now()
	p.version++
	return nil
}

func (p *Payment) Complete() error {
	if p.status != StatusProcessing {
		return errors.New("payment must be in processing state")
	}
	p.status = StatusCompleted
	p.updatedAt = time.Now()
	p.version++
	return nil
}

func (p *Payment) Fail(reason string) error {
	p.status = StatusFailed
	p.metadata["failure_reason"] = reason
	p.updatedAt = time.Now()
	p.version++
	return nil
}

func (p *Payment) Cancel() error {
	if p.status == StatusCompleted {
		return errors.New("cannot cancel completed payment")
	}
	p.status = StatusCancelled
	p.updatedAt = time.Now()
	p.version++
	return nil
}

// Getters (following encapsulation principle)
func (p *Payment) ID() uuid.UUID         { return p.id }
func (p *Payment) UserID() string        { return p.userID }
func (p *Payment) Amount() Money         { return p.amount }
func (p *Payment) Status() PaymentStatus { return p.status }
func (p *Payment) Method() PaymentMethod { return p.method }
func (p *Payment) MerchantID() string    { return p.merchantID }
func (p *Payment) TransactionID() string { return p.transactionID }
func (p *Payment) CreatedAt() time.Time  { return p.createdAt }
func (p *Payment) Version() int          { return p.version }
