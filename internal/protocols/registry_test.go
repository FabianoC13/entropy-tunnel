package protocols

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	vless := NewVLESS()
	trojan := NewTrojan("/ws")

	// Register
	if err := r.Register(vless); err != nil {
		t.Fatalf("Register(vless) error = %v", err)
	}
	if err := r.Register(trojan); err != nil {
		t.Fatalf("Register(trojan) error = %v", err)
	}

	// Duplicate registration
	if err := r.Register(vless); err == nil {
		t.Error("expected error for duplicate registration")
	}

	// Get
	got, err := r.Get("vless")
	if err != nil {
		t.Fatalf("Get(vless) error = %v", err)
	}
	if got.Name() != "vless" {
		t.Errorf("Get(vless).Name() = %q, want 'vless'", got.Name())
	}

	// Get unknown
	_, err = r.Get("wireguard")
	if err == nil {
		t.Error("expected error for unknown protocol")
	}

	// List
	names := r.List()
	if len(names) != 2 {
		t.Errorf("List() returned %d items, want 2", len(names))
	}
}

func TestSelectBest(t *testing.T) {
	r := NewRegistry()

	trojan := NewTrojan("/ws")   // Priority 2
	vless := NewVLESS()          // Priority 1

	_ = r.Register(trojan) // Register lower-priority first
	_ = r.Register(vless)

	best, err := r.SelectBest()
	if err != nil {
		t.Fatalf("SelectBest() error = %v", err)
	}
	if best.Name() != "vless" {
		t.Errorf("SelectBest() = %q, want 'vless' (priority 1)", best.Name())
	}
}

func TestFallbackChain(t *testing.T) {
	r := NewRegistry()

	trojan := NewTrojan("/ws")
	vless := NewVLESS()

	_ = r.Register(trojan)
	_ = r.Register(vless)

	chain := r.FallbackChain()
	if len(chain) != 2 {
		t.Fatalf("FallbackChain() returned %d items, want 2", len(chain))
	}
	if chain[0].Name() != "vless" {
		t.Errorf("FallbackChain()[0] = %q, want 'vless'", chain[0].Name())
	}
	if chain[1].Name() != "trojan" {
		t.Errorf("FallbackChain()[1] = %q, want 'trojan'", chain[1].Name())
	}
}

func TestEmptyRegistrySelectBest(t *testing.T) {
	r := NewRegistry()
	_, err := r.SelectBest()
	if err == nil {
		t.Error("expected error for empty registry")
	}
}
