package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"payment-api/internal/application/usecase"
	"payment-api/internal/domain/payment"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WebHandler handles web interface requests
type WebHandler struct {
	processUseCase *usecase.ProcessPaymentUseCase
	getUseCase     *usecase.GetPaymentUseCase
	logger         *zap.Logger
	templates      *template.Template
}

// NewWebHandler creates a new web handler
func NewWebHandler(
	processUseCase *usecase.ProcessPaymentUseCase,
	getUseCase *usecase.GetPaymentUseCase,
	logger *zap.Logger,
) *WebHandler {
	templates := template.Must(template.ParseGlob("web/templates/*.html"))
	return &WebHandler{
		processUseCase: processUseCase,
		getUseCase:     getUseCase,
		logger:         logger,
		templates:      templates,
	}
}

// IndexPage renders the main page
func (h *WebHandler) IndexPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// CreatePaymentPage renders the create payment form
func (h *WebHandler) CreatePaymentPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "create-payment.html", nil); err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// GetPaymentPage renders the get payment form
func (h *WebHandler) GetPaymentPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "get-payment.html", nil); err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// CreatePaymentForm handles HTMX form submission for creating payment
func (h *WebHandler) CreatePaymentForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, "Invalid form data")
		return
	}

	// Parse amount
	var amount int64
	if _, err := fmt.Sscanf(r.FormValue("amount"), "%d", &amount); err != nil {
		h.renderError(w, "Invalid amount")
		return
	}

	// Create payment request
	useCaseReq := usecase.ProcessPaymentRequest{
		UserID:     r.FormValue("user_id"),
		MerchantID: r.FormValue("merchant_id"),
		Amount:     amount,
		Currency:   r.FormValue("currency"),
		Method:     payment.PaymentMethod(r.FormValue("method")),
	}

	// Validate
	if useCaseReq.UserID == "" || useCaseReq.MerchantID == "" || useCaseReq.Amount <= 0 || useCaseReq.Currency == "" {
		h.renderError(w, "All fields are required and amount must be positive")
		return
	}

	// Execute use case
	resp, err := h.processUseCase.Execute(r.Context(), useCaseReq)
	if err != nil {
		h.logger.Error("Failed to process payment", zap.Error(err))
		h.renderError(w, "Failed to process payment: "+err.Error())
		return
	}

	// Render success response
	data := map[string]interface{}{
		"PaymentID": resp.PaymentID.String(),
		"Status":    string(resp.Status),
		"CreatedAt": resp.CreatedAt.Format(time.RFC3339),
	}

	if err := h.templates.ExecuteTemplate(w, "payment-result.html", data); err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
	}
}

// GetPaymentForm handles HTMX form submission for getting payment
func (h *WebHandler) GetPaymentForm(w http.ResponseWriter, r *http.Request) {
	paymentID := r.FormValue("payment_id")
	if paymentID == "" {
		h.renderError(w, "Payment ID is required")
		return
	}

	id, err := uuid.Parse(paymentID)
	if err != nil {
		h.renderError(w, "Invalid payment ID format")
		return
	}

	// Execute use case
	pmt, err := h.getUseCase.Execute(r.Context(), id)
	if err != nil {
		if err == payment.ErrPaymentNotFound {
			h.renderError(w, "Payment not found")
			return
		}
		h.logger.Error("Failed to get payment", zap.Error(err))
		h.renderError(w, "Failed to get payment: "+err.Error())
		return
	}

	// Render payment details
	data := map[string]interface{}{
		"PaymentID":     pmt.ID().String(),
		"UserID":        pmt.UserID(),
		"MerchantID":    pmt.MerchantID(),
		"Amount":        pmt.Amount().Amount,
		"Currency":      pmt.Amount().Currency,
		"Status":        string(pmt.Status()),
		"Method":        string(pmt.Method()),
		"TransactionID": pmt.TransactionID(),
		"CreatedAt":     pmt.CreatedAt().Format(time.RFC3339),
	}

	if err := h.templates.ExecuteTemplate(w, "payment-details.html", data); err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
	}
}

// GetPaymentByID handles direct URL access to payment details
func (h *WebHandler) GetPaymentByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.renderError(w, "Invalid payment ID format")
		return
	}

	// Execute use case
	pmt, err := h.getUseCase.Execute(r.Context(), id)
	if err != nil {
		if err == payment.ErrPaymentNotFound {
			h.renderError(w, "Payment not found")
			return
		}
		h.logger.Error("Failed to get payment", zap.Error(err))
		h.renderError(w, "Failed to get payment: "+err.Error())
		return
	}

	// Render full page with payment details
	data := map[string]interface{}{
		"PaymentID":     pmt.ID().String(),
		"UserID":        pmt.UserID(),
		"MerchantID":    pmt.MerchantID(),
		"Amount":        pmt.Amount().Amount,
		"Currency":      pmt.Amount().Currency,
		"Status":        string(pmt.Status()),
		"Method":        string(pmt.Method()),
		"TransactionID": pmt.TransactionID(),
		"CreatedAt":     pmt.CreatedAt().Format(time.RFC3339),
	}

	if err := h.templates.ExecuteTemplate(w, "payment-page.html", data); err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
	}
}

func (h *WebHandler) renderError(w http.ResponseWriter, message string) {
	data := map[string]string{"Error": message}
	w.WriteHeader(http.StatusBadRequest)
	if err := h.templates.ExecuteTemplate(w, "error.html", data); err != nil {
		h.logger.Error("Failed to render error template", zap.Error(err))
		http.Error(w, message, http.StatusBadRequest)
	}
}
