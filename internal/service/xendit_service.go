package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// XenditService handles Xendit payment API calls.
type XenditService struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewXenditService creates a new XenditService.
func NewXenditService(apiKey, baseURL string) *XenditService {
	if baseURL == "" {
		baseURL = "https://api.xendit.co"
	}
	return &XenditService{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ── Invoice Types ───────────────────────────────────────────────────

// CreateInvoiceInput holds data needed to create a Xendit invoice.
type CreateInvoiceInput struct {
	ExternalID  string // our subscription ID
	Amount      int    // in IDR
	PayerEmail  string
	Description string
	SuccessURL  string
	FailureURL  string
}

// InvoiceResponse holds the Xendit API response.
type InvoiceResponse struct {
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
	InvoiceURL string `json:"invoice_url"`
	Status     string `json:"status"`
	Amount     int    `json:"amount"`
	ExpiryDate string `json:"expiry_date"`
}

// CreateInvoice creates a new Xendit invoice for payment.
func (s *XenditService) CreateInvoice(ctx context.Context, input CreateInvoiceInput) (*InvoiceResponse, error) {
	body := map[string]interface{}{
		"external_id":          input.ExternalID,
		"amount":               input.Amount,
		"payer_email":          input.PayerEmail,
		"description":          input.Description,
		"invoice_duration":     86400, // 24 hours
		"currency":             "IDR",
		"should_send_email":    true,
		"success_redirect_url": input.SuccessURL,
		"failure_redirect_url": input.FailureURL,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/v2/invoices", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(s.apiKey, "")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Xendit: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("xendit API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var invoice InvoiceResponse
	if err := json.Unmarshal(respBody, &invoice); err != nil {
		return nil, fmt.Errorf("failed to parse Xendit response: %w", err)
	}

	return &invoice, nil
}

// ── Webhook Types ───────────────────────────────────────────────────

// WebhookPayload is the Xendit callback payload.
type WebhookPayload struct {
	ID             string `json:"id"`
	ExternalID     string `json:"external_id"`
	Status         string `json:"status"` // PAID, EXPIRED
	PayerEmail     string `json:"payer_email"`
	Amount         int    `json:"amount"`
	PaidAmount     int    `json:"paid_amount"`
	PaymentMethod  string `json:"payment_method"`
	PaymentChannel string `json:"payment_channel"`
	PaymentID      string `json:"payment_id"`
	Currency       string `json:"currency"`
	PaidAt         string `json:"paid_at"`
}
