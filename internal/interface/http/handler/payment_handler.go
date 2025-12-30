package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"payment-api/internal/application/usecase"
	"payment-api/internal/domain/payment"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PaymentHandler handles payment HTTP requests
type PaymentHandler struct {
	processUseCase *usecase.ProcessPaymentUseCase
	getUseCase     *usecase.GetPaymentUseCase
	logger         *zap.Logger
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(
	processUseCase *usecase.ProcessPaymentUseCase,
	getUseCase *usecase.GetPaymentUseCase,
	logger *zap.Logger,
) *PaymentHandler {
	return &PaymentHandler{
		processUseCase: processUseCase,
		getUseCase:     getUseCase,
		logger:         logger,
	}
}

// CreatePaymentRequest represents HTTP request body
type CreatePaymentRequest struct {
	UserID     string `json:"user_id"`
	MerchantID string `json:"merchant_id"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	Method     string `json:"method"`
}

// PaymentResponse represents HTTP response
type PaymentResponse struct {
	PaymentID     string    `json:"payment_id"`
	Status        string    `json:"status"`
	TransactionID string    `json:"transaction_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// CreatePayment handles POST /api/v1/payments
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if err := h.validateCreateRequest(req); err != nil {
		h.respondError(w, http.StatusBadRequest, "validation failed", err)
		return
	}

	// Convert to use case request
	useCaseReq := usecase.ProcessPaymentRequest{
		UserID:     req.UserID,
		MerchantID: req.MerchantID,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Method:     payment.PaymentMethod(req.Method),
	}

	// Execute use case
	resp, err := h.processUseCase.Execute(r.Context(), useCaseReq)
	if err != nil {
		h.logger.Error("Failed to process payment", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to process payment", err)
		return
	}

	// Return response
	h.respondJSON(w, http.StatusAccepted, PaymentResponse{
		PaymentID: resp.PaymentID.String(),
		Status:    string(resp.Status),
		CreatedAt: resp.CreatedAt,
	})
}

// GetPayment handles GET /api/v1/payments/{id}
func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid payment ID", err)
		return
	}

	// Execute use case
	pmt, err := h.getUseCase.Execute(r.Context(), id)
	if err != nil {
		if err == payment.ErrPaymentNotFound {
			h.respondError(w, http.StatusNotFound, "payment not found", err)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get payment", err)
		return
	}

	// Return response
	h.respondJSON(w, http.StatusOK, PaymentResponse{
		PaymentID:     pmt.ID().String(),
		Status:        string(pmt.Status()),
		TransactionID: pmt.TransactionID(),
		CreatedAt:     pmt.CreatedAt(),
	})
}

// HealthCheck handles GET /health
func (h *PaymentHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// ReadinessCheck handles GET /ready
func (h *PaymentHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// Check dependencies (database, redis, etc.)
	// For simplicity, return ready
	h.respondJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func (h *PaymentHandler) validateCreateRequest(req CreatePaymentRequest) error {
	if req.UserID == "" {
		return &ValidationError{"user_id is required"}
	}
	if req.MerchantID == "" {
		return &ValidationError{"merchant_id is required"}
	}
	if req.Amount <= 0 {
		return &ValidationError{"amount must be positive"}
	}
	if req.Currency == "" {
		return &ValidationError{"currency is required"}
	}
	if req.Method == "" {
		return &ValidationError{"payment method is required"}
	}
	return nil
}

func (h *PaymentHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (h *PaymentHandler) respondError(w http.ResponseWriter, status int, message string, err error) {
	h.logger.Error(message, zap.Error(err))
	h.respondJSON(w, status, ErrorResponse{
		Error:   err.Error(),
		Message: message,
		Code:    status,
	})
}

type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}
