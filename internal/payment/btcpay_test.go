package payment

import (
	"testing"
)

func TestAvailablePlans(t *testing.T) {
	plans := AvailablePlans()
	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}

	monthly := plans[0]
	if monthly.ID != "entropy-monthly" {
		t.Errorf("expected monthly plan ID 'entropy-monthly', got %q", monthly.ID)
	}
	if monthly.Price <= 0 {
		t.Error("monthly price should be positive")
	}
	if monthly.Currency != "USD" {
		t.Errorf("expected currency 'USD', got %q", monthly.Currency)
	}
	if len(monthly.Features) == 0 {
		t.Error("monthly plan should have features")
	}

	yearly := plans[1]
	if yearly.Price >= monthly.Price*12 {
		t.Error("yearly plan should be cheaper than 12x monthly")
	}
}

func TestNewBTCPayClient(t *testing.T) {
	client := NewBTCPayClient("https://btcpay.example.com", "api-key", "store-123")
	if client == nil {
		t.Fatal("NewBTCPayClient returned nil")
	}
	if client.baseURL != "https://btcpay.example.com" {
		t.Errorf("expected baseURL, got %q", client.baseURL)
	}
	if client.storeID != "store-123" {
		t.Errorf("expected storeID 'store-123', got %q", client.storeID)
	}
}
