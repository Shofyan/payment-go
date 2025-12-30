package payment

import (
	"time"

	"github.com/google/uuid"
)

// Domain Events - Event-Driven Architecture
type DomainEvent interface {
	EventID() uuid.UUID
	EventType() string
	OccurredAt() time.Time
	AggregateID() uuid.UUID
}

type BaseEvent struct {
	eventID     uuid.UUID
	eventType   string
	occurredAt  time.Time
	aggregateID uuid.UUID
}

func (e BaseEvent) EventID() uuid.UUID     { return e.eventID }
func (e BaseEvent) EventType() string      { return e.eventType }
func (e BaseEvent) OccurredAt() time.Time  { return e.occurredAt }
func (e BaseEvent) AggregateID() uuid.UUID { return e.aggregateID }

type PaymentCreatedEvent struct {
	BaseEvent
	UserID     string
	Amount     Money
	Method     PaymentMethod
	MerchantID string
}

type PaymentProcessedEvent struct {
	BaseEvent
	TransactionID string
}

type PaymentCompletedEvent struct {
	BaseEvent
	TransactionID string
	CompletedAt   time.Time
}

type PaymentFailedEvent struct {
	BaseEvent
	Reason string
}

func NewPaymentCreatedEvent(payment *Payment) *PaymentCreatedEvent {
	return &PaymentCreatedEvent{
		BaseEvent: BaseEvent{
			eventID:     uuid.New(),
			eventType:   "payment.created",
			occurredAt:  time.Now(),
			aggregateID: payment.id,
		},
		UserID:     payment.userID,
		Amount:     payment.amount,
		Method:     payment.method,
		MerchantID: payment.merchantID,
	}
}
