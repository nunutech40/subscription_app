package service

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MidtransService handles Midtrans Snap API calls.
type MidtransService struct {
	serverKey  string
	baseURL    string
	httpClient *http.Client
}

// NewMidtransService creates a new MidtransService.
// For sandbox: baseURL = "https://app.sandbox.midtrans.com"
// For production: baseURL = "https://app.midtrans.com"
func NewMidtransService(serverKey, baseURL string) *MidtransService {
	if baseURL == "" {
		baseURL = "https://app.sandbox.midtrans.com"
	}
	return &MidtransService{
		serverKey: serverKey,
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ── Snap Transaction Types ──────────────────────────────────────────

// CreateTransactionInput holds data needed to create a Midtrans Snap transaction.
type CreateTransactionInput struct {
	OrderID     string // our subscription ID
	Amount      int    // in IDR (gross_amount)
	PayerEmail  string
	PayerName   string
	Description string
	FinishURL   string // redirect URL after payment
}

// SnapResponse holds the Midtrans Snap API response.
type SnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

// CreateTransaction creates a new Midtrans Snap transaction.
func (s *MidtransService) CreateTransaction(ctx context.Context, input CreateTransactionInput) (*SnapResponse, error) {
	body := map[string]interface{}{
		"transaction_details": map[string]interface{}{
			"order_id":     input.OrderID,
			"gross_amount": input.Amount,
		},
		"customer_details": map[string]interface{}{
			"email":      input.PayerEmail,
			"first_name": input.PayerName,
		},
		"item_details": []map[string]interface{}{
			{
				"id":       "sub-" + input.OrderID[:8],
				"name":     input.Description,
				"price":    input.Amount,
				"quantity": 1,
			},
		},
		"callbacks": map[string]interface{}{
			"finish": input.FinishURL,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/snap/v1/transactions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Midtrans uses Basic Auth: ServerKey as username, empty password
	req.SetBasicAuth(s.serverKey, "")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Midtrans: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("midtrans API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var snapResp SnapResponse
	if err := json.Unmarshal(respBody, &snapResp); err != nil {
		return nil, fmt.Errorf("failed to parse Midtrans response: %w", err)
	}

	return &snapResp, nil
}

// ── Webhook/Notification Types ──────────────────────────────────────

// MidtransNotification is the webhook payload from Midtrans.
type MidtransNotification struct {
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	TransactionStatus string `json:"transaction_status"` // settlement, pending, expire, cancel, deny
	FraudStatus       string `json:"fraud_status"`       // accept, deny, challenge
	PaymentType       string `json:"payment_type"`
	GrossAmount       string `json:"gross_amount"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	TransactionTime   string `json:"transaction_time"`
	StatusMessage     string `json:"status_message"`
}

// VerifySignature verifies the Midtrans notification signature.
// Formula: SHA512(order_id + status_code + gross_amount + server_key)
func (s *MidtransService) VerifySignature(notif MidtransNotification) bool {
	raw := notif.OrderID + notif.StatusCode + notif.GrossAmount + s.serverKey
	hash := sha512.Sum512([]byte(raw))
	computed := fmt.Sprintf("%x", hash)
	return computed == notif.SignatureKey
}

// IsPaymentSuccess checks if the notification indicates a successful payment.
func IsPaymentSuccess(notif MidtransNotification) bool {
	return notif.TransactionStatus == "settlement" ||
		(notif.TransactionStatus == "capture" && notif.FraudStatus == "accept")
}

// IsPaymentExpired checks if the notification indicates an expired/cancelled payment.
func IsPaymentExpired(notif MidtransNotification) bool {
	return notif.TransactionStatus == "expire" ||
		notif.TransactionStatus == "cancel" ||
		notif.TransactionStatus == "deny"
}
