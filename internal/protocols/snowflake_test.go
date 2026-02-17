package protocols

import (
	"context"
	"testing"
)

func TestSnowflakeProtocol_Name(t *testing.T) {
	sf := NewSnowflake(nil, nil)
	if sf.Name() != "snowflake" {
		t.Errorf("Name() = %q, want 'snowflake'", sf.Name())
	}
}

func TestSnowflakeProtocol_Priority(t *testing.T) {
	sf := NewSnowflake(nil, nil)
	if sf.Priority() != 99 {
		t.Errorf("Priority() = %d, want 99 (emergency only)", sf.Priority())
	}
}

func TestSnowflakeProtocol_Available(t *testing.T) {
	sf := NewSnowflake(nil, nil)
	if !sf.Available() {
		t.Error("Snowflake should always be available as emergency fallback")
	}
}

func TestSnowflakeProtocol_ListenNotSupported(t *testing.T) {
	sf := NewSnowflake(nil, nil)
	_, err := sf.Listen(":1234")
	if err == nil {
		t.Error("expected error: Snowflake doesn't support Listen")
	}
}

func TestSnowflakeProtocol_Stats(t *testing.T) {
	cfg := DefaultSnowflakeConfig()
	sf := NewSnowflake(cfg, nil)

	stats := sf.Stats()
	if stats["broker"] != cfg.BrokerURL {
		t.Errorf("stats broker = %v, want %q", stats["broker"], cfg.BrokerURL)
	}
	if stats["total_connections"] != 0 {
		t.Errorf("stats total_connections = %v, want 0", stats["total_connections"])
	}
}

func TestSnowflakeProtocol_DefaultConfig(t *testing.T) {
	cfg := DefaultSnowflakeConfig()
	if cfg.BrokerURL == "" {
		t.Error("default BrokerURL should not be empty")
	}
	if len(cfg.STUNURLs) == 0 {
		t.Error("default STUNURLs should not be empty")
	}
	if cfg.MaxPeers <= 0 {
		t.Error("default MaxPeers should be positive")
	}
}

func TestSnowflakeProtocol_InRegistry(t *testing.T) {
	r := NewRegistry()
	sf := NewSnowflake(nil, nil)
	vless := NewVLESS()

	_ = r.Register(vless)
	_ = r.Register(sf)

	chain := r.FallbackChain()
	if len(chain) != 2 {
		t.Fatalf("expected 2 protocols in chain, got %d", len(chain))
	}
	// VLESS (priority 1) should come before Snowflake (priority 99)
	if chain[0].Name() != "vless" {
		t.Errorf("chain[0] = %q, want 'vless'", chain[0].Name())
	}
	if chain[1].Name() != "snowflake" {
		t.Errorf("chain[1] = %q, want 'snowflake'", chain[1].Name())
	}
}

func TestSnowflakeProtocol_DialLocal(t *testing.T) {
	cfg := &SnowflakeConfig{KeepLocal: true}
	sf := NewSnowflake(cfg, nil)

	// DialContext in local mode should try direct TCP
	ctx := context.Background()
	_, err := sf.DialContext(ctx, "127.0.0.1:0") // Invalid port, should fail
	if err == nil {
		t.Error("expected error dialing invalid address")
	}
}
