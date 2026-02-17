package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BTCPayClient provides integration with BTCPay Server for crypto-only payments.
type BTCPayClient struct {
	baseURL  string
	apiKey   string
	storeID  string
	client   *http.Client
}

// Invoice represents a BTCPay Server invoice.
type Invoice struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"` // New, Processing, Settled, Expired, Invalid
	Amount      string    `json:"amount"`
	Currency    string    `json:"currency"`
	CheckoutURL string    `json:"checkoutLink"`
	CreatedAt   time.Time `json:"createdTime"`
	ExpiresAt   time.Time `json:"expirationTime"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Plan represents a subscription plan.
type Plan struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	Duration string  `json:"duration"` // "monthly", "yearly"
	Features []string `json:"features"`
}

// AvailablePlans returns the subscription plans.
func AvailablePlans() []Plan {
	return []Plan{
		{
			ID:       "entropy-monthly",
			Name:     "EntropyTunnel Monthly",
			Price:    9.99,
			Currency: "USD",
			Duration: "monthly",
			Features: []string{
				"Unlimited bandwidth",
				"VLESS + XTLS-Reality",
				"Auto endpoint rotation",
				"Sports Mode",
				"5 devices",
			},
		},
		{
			ID:       "entropy-yearly",
			Name:     "EntropyTunnel Yearly",
			Price:    79.99,
			Currency: "USD",
			Duration: "yearly",
			Features: []string{
				"Everything in Monthly",
				"10 devices",
				"Priority rotation",
				"Snowflake fallback",
				"2 months free",
			},
		},
	}
}

// NewBTCPayClient creates a BTCPay Server client.
func NewBTCPayClient(baseURL, apiKey, storeID string) *BTCPayClient {
	return &BTCPayClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		storeID: storeID,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateInvoice creates a new payment invoice.
func (b *BTCPayClient) CreateInvoice(ctx context.Context, plan Plan, email string) (*Invoice, error) {
	payload, _ := json.Marshal(map[string]any{
		"amount":   plan.Price,
		"currency": plan.Currency,
		"metadata": map[string]any{
			"plan_id": plan.ID,
			"email":   email,
		},
		"checkout": map[string]any{
			"defaultPaymentMethod": "BTC",
			"expirationMinutes":    30,
			"redirectURL":          fmt.Sprintf("%s/payment/success", b.baseURL),
		},
	})

	url := fmt.Sprintf("%s/api/v1/stores/%s/invoices", b.baseURL, b.storeID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+b.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BTCPay API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("BTCPay error %d: %s", resp.StatusCode, string(body))
	}

	var invoice Invoice
	if err := json.NewDecoder(resp.Body).Decode(&invoice); err != nil {
		return nil, fmt.Errorf("failed to decode invoice: %w", err)
	}

	return &invoice, nil
}

// GetInvoice retrieves an existing invoice.
func (b *BTCPayClient) GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error) {
	url := fmt.Sprintf("%s/api/v1/stores/%s/invoices/%s", b.baseURL, b.storeID, invoiceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var invoice Invoice
	if err := json.NewDecoder(resp.Body).Decode(&invoice); err != nil {
		return nil, err
	}

	return &invoice, nil
}

// IsActive checks if a user has an active subscription.
func (b *BTCPayClient) IsActive(ctx context.Context, email string) (bool, error) {
	// In production, this would query a database of paid subscriptions.
	// For the MVP, we check recent invoices.
	url := fmt.Sprintf("%s/api/v1/stores/%s/invoices?status=Settled", b.baseURL, b.storeID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "token "+b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var invoices []Invoice
	if err := json.NewDecoder(resp.Body).Decode(&invoices); err != nil {
		return false, err
	}

	for _, inv := range invoices {
		if meta, ok := inv.Metadata["email"].(string); ok && meta == email {
			return true, nil
		}
	}

	return false, nil
}
