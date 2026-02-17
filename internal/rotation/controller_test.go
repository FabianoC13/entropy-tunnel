package rotation

import (
	"context"
	"testing"
	"time"
)

func TestNoOpController_Rotate(t *testing.T) {
	ctrl := NewNoOpController(nil)

	ep, err := ctrl.Rotate(context.Background())
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if ep.ID == "" {
		t.Error("expected non-empty endpoint ID")
	}
	if ep.Provider != "noop" {
		t.Errorf("expected provider 'noop', got %q", ep.Provider)
	}
}

func TestNoOpController_Retire(t *testing.T) {
	ctrl := NewNoOpController(nil)

	ep, _ := ctrl.Rotate(context.Background())
	err := ctrl.Retire(context.Background(), ep)
	if err != nil {
		t.Fatalf("Retire() error = %v", err)
	}

	// Should have 0 active endpoints
	active := ctrl.ActiveEndpoints()
	if len(active) != 0 {
		t.Errorf("expected 0 active endpoints, got %d", len(active))
	}
}

func TestNoOpController_RetireNotFound(t *testing.T) {
	ctrl := NewNoOpController(nil)

	err := ctrl.Retire(context.Background(), &Endpoint{ID: "nonexistent"})
	if err == nil {
		t.Error("expected error retiring nonexistent endpoint")
	}
}

func TestNoOpController_ActiveEndpoints(t *testing.T) {
	ctrl := NewNoOpController(nil)

	// Create 3 endpoints
	for i := 0; i < 3; i++ {
		_, _ = ctrl.Rotate(context.Background())
	}

	active := ctrl.ActiveEndpoints()
	if len(active) != 3 {
		t.Errorf("expected 3 active endpoints, got %d", len(active))
	}
}

func TestEndpoint_IsExpired(t *testing.T) {
	ep := &Endpoint{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if !ep.IsExpired() {
		t.Error("endpoint should be expired")
	}

	ep.ExpiresAt = time.Now().Add(1 * time.Hour)
	if ep.IsExpired() {
		t.Error("endpoint should not be expired")
	}
}

func TestCloudflareController_Create(t *testing.T) {
	ctrl := NewCloudflareController("token", "account", "zone", nil)
	if ctrl == nil {
		t.Fatal("NewCloudflareController returned nil")
	}
	if ctrl.apiToken != "token" {
		t.Errorf("expected apiToken 'token', got %q", ctrl.apiToken)
	}
}

func TestAWSController_Create(t *testing.T) {
	ctrl := NewAWSController("us-east-1", "key", "secret", nil)
	if ctrl == nil {
		t.Fatal("NewAWSController returned nil")
	}
	if ctrl.region != "us-east-1" {
		t.Errorf("expected region 'us-east-1', got %q", ctrl.region)
	}
}

func TestHealthChecker_Create(t *testing.T) {
	ctrl := NewNoOpController(nil)
	hc := NewHealthChecker(ctrl, 30*time.Second, 5*time.Second, nil)
	if hc == nil {
		t.Fatal("NewHealthChecker returned nil")
	}
}

func TestHealthChecker_Results(t *testing.T) {
	ctrl := NewNoOpController(nil)
	hc := NewHealthChecker(ctrl, 30*time.Second, 5*time.Second, nil)

	results := hc.Results()
	if len(results) != 0 {
		t.Errorf("expected 0 results initially, got %d", len(results))
	}
}
