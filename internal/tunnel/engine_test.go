package tunnel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ---- Config Validation Tests ----

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Listen:   ":443",
				Protocol: "vless",
				UUID:     "test-uuid-1234",
				Reality: RealityConfig{
					SNI:        "www.google.com",
					PrivateKey: "test-private-key",
				},
				Fingerprint: "chrome",
			},
			wantErr: false,
		},
		{name: "missing listen", config: Config{UUID: "x", Reality: RealityConfig{SNI: "g.com", PrivateKey: "k"}}, wantErr: true},
		{name: "missing UUID", config: Config{Listen: ":443", Reality: RealityConfig{SNI: "g.com", PrivateKey: "k"}}, wantErr: true},
		{name: "missing SNI", config: Config{Listen: ":443", UUID: "x", Reality: RealityConfig{PrivateKey: "k"}}, wantErr: true},
		{name: "missing private key", config: Config{Listen: ":443", UUID: "x", Reality: RealityConfig{SNI: "g.com"}}, wantErr: true},
		{name: "invalid protocol", config: Config{Listen: ":443", Protocol: "invalid", UUID: "x", Reality: RealityConfig{SNI: "g.com", PrivateKey: "k"}}, wantErr: true},
		{
			name: "defaults applied",
			config: Config{
				Listen:  ":443",
				UUID:    "x",
				Reality: RealityConfig{SNI: "g.com", PrivateKey: "k"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name:    "valid",
			config:  ClientConfig{Server: "1.2.3.4:443", UUID: "u", SNI: "g.com", PublicKey: "pk"},
			wantErr: false,
		},
		{name: "missing server", config: ClientConfig{UUID: "u", SNI: "g.com", PublicKey: "pk"}, wantErr: true},
		{name: "missing uuid", config: ClientConfig{Server: "x", SNI: "g.com", PublicKey: "pk"}, wantErr: true},
		{name: "missing sni", config: ClientConfig{Server: "x", UUID: "u", PublicKey: "pk"}, wantErr: true},
		{name: "missing pubkey", config: ClientConfig{Server: "x", UUID: "u", SNI: "g.com"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ---- Config Builder Tests ----

func TestBuildServerJSON(t *testing.T) {
	cfg := &Config{
		Listen:   ":443",
		Protocol: "vless",
		UUID:     "test-uuid-1234",
		Reality: RealityConfig{
			SNI:        "www.google.com",
			PrivateKey: "test-private-key",
			ShortIDs:   []string{"abcd1234"},
		},
		Fingerprint: "chrome",
		Fallbacks: []FallbackConfig{
			{Protocol: "trojan", Listen: ":8443", Transport: "ws", Path: "/ws"},
		},
	}

	jsonBytes, err := BuildServerJSON(cfg)
	if err != nil {
		t.Fatalf("BuildServerJSON() error = %v", err)
	}

	var parsed xrayFullConfig
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(parsed.Inbounds) != 2 {
		t.Errorf("expected 2 inbounds, got %d", len(parsed.Inbounds))
	}
	if len(parsed.Outbounds) != 2 {
		t.Errorf("expected 2 outbounds, got %d", len(parsed.Outbounds))
	}
	if parsed.Inbounds[0].Protocol != "vless" {
		t.Errorf("expected protocol 'vless', got %q", parsed.Inbounds[0].Protocol)
	}
	if parsed.Inbounds[0].Port != 443 {
		t.Errorf("expected port 443, got %d", parsed.Inbounds[0].Port)
	}
	if parsed.Inbounds[0].Stream == nil || parsed.Inbounds[0].Stream.Security != "reality" {
		t.Error("expected reality security on primary inbound")
	}
}

func TestBuildClientJSON(t *testing.T) {
	cfg := &ClientConfig{
		Server:      "1.2.3.4:443",
		UUID:        "test-uuid",
		SNI:         "www.google.com",
		Fingerprint: "chrome",
		PublicKey:   "test-pubkey",
		ShortID:     "abcd",
		LocalListen: "127.0.0.1:1080",
	}

	jsonBytes, err := BuildClientJSON(cfg)
	if err != nil {
		t.Fatalf("BuildClientJSON() error = %v", err)
	}

	var parsed xrayFullConfig
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(parsed.Inbounds) != 1 {
		t.Errorf("expected 1 inbound (socks), got %d", len(parsed.Inbounds))
	}
	if parsed.Inbounds[0].Protocol != "socks" {
		t.Errorf("expected socks protocol, got %q", parsed.Inbounds[0].Protocol)
	}
	if parsed.Outbounds[0].Protocol != "vless" {
		t.Errorf("expected vless outbound, got %q", parsed.Outbounds[0].Protocol)
	}
	if parsed.Outbounds[0].Stream.Reality.Fingerprint != "chrome" {
		t.Errorf("expected chrome fingerprint, got %q", parsed.Outbounds[0].Stream.Reality.Fingerprint)
	}
}

func TestBuildClientJSON_WithHTTPListen(t *testing.T) {
	cfg := &ClientConfig{
		Server:      "1.2.3.4:443",
		UUID:        "u",
		SNI:         "g.com",
		PublicKey:   "pk",
		LocalListen: "127.0.0.1:1080",
		HTTPListen:  "127.0.0.1:8080",
	}

	jsonBytes, err := BuildClientJSON(cfg)
	if err != nil {
		t.Fatalf("BuildClientJSON() error = %v", err)
	}

	var parsed xrayFullConfig
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(parsed.Inbounds) != 2 {
		t.Errorf("expected 2 inbounds (socks+http), got %d", len(parsed.Inbounds))
	}
}

func TestBuildServerJSON_NilConfig(t *testing.T) {
	_, err := BuildServerJSON(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestBuildClientJSON_NilConfig(t *testing.T) {
	_, err := BuildClientJSON(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

// ---- Engine Tests ----

func TestNewEngine(t *testing.T) {
	cfg := &Config{
		Listen: ":443", Protocol: "vless", UUID: "test-uuid",
		Reality: RealityConfig{SNI: "www.google.com", PrivateKey: "key"},
	}

	engine, err := NewEngine(cfg, nil)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}
	if engine.Status() != StatusStopped {
		t.Errorf("expected status 'stopped', got %q", engine.Status())
	}
}

func TestNewClientEngine(t *testing.T) {
	cfg := &ClientConfig{
		Server: "1.2.3.4:443", UUID: "u", SNI: "g.com", PublicKey: "pk",
	}

	engine, err := NewClientEngine(cfg, nil)
	if err != nil {
		t.Fatalf("NewClientEngine() error = %v", err)
	}
	if engine.Status() != StatusStopped {
		t.Errorf("expected status 'stopped', got %q", engine.Status())
	}
	if engine.mode != ModeClient {
		t.Errorf("expected mode 'client', got %q", engine.mode)
	}
}

func TestNewEngineNilConfig(t *testing.T) {
	_, err := NewEngine(nil, nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestNewClientEngineNilConfig(t *testing.T) {
	_, err := NewClientEngine(nil, nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestEngineStartStop(t *testing.T) {
	cfg := &Config{
		Listen: ":443", Protocol: "vless", UUID: "test-uuid",
		Reality: RealityConfig{SNI: "www.google.com", PrivateKey: "key"},
	}

	engine, err := NewEngine(cfg, nil)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if engine.Status() != StatusRunning {
		t.Errorf("expected status 'running', got %q", engine.Status())
	}

	// Double start should fail
	if err := engine.Start(); err == nil {
		t.Error("expected error on double start")
	}

	if err := engine.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if engine.Status() != StatusStopped {
		t.Errorf("expected status 'stopped', got %q", engine.Status())
	}

	// Double stop should fail
	if err := engine.Stop(); err == nil {
		t.Error("expected error on double stop")
	}
}

func TestClientEngineStartStop(t *testing.T) {
	cfg := &ClientConfig{
		Server: "1.2.3.4:443", UUID: "u", SNI: "g.com", PublicKey: "pk",
		LocalListen: "127.0.0.1:1080",
	}

	engine, err := NewClientEngine(cfg, nil)
	if err != nil {
		t.Fatalf("NewClientEngine() error = %v", err)
	}

	if err := engine.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if engine.Status() != StatusRunning {
		t.Errorf("expected running, got %q", engine.Status())
	}

	if err := engine.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestEngineJSONConfig(t *testing.T) {
	cfg := &Config{
		Listen: ":443", Protocol: "vless", UUID: "test-uuid",
		Reality: RealityConfig{SNI: "www.google.com", PrivateKey: "key"},
	}

	engine, _ := NewEngine(cfg, nil)
	_ = engine.Start()
	defer engine.Stop()

	jsonCfg := engine.JSONConfig()
	if jsonCfg == nil {
		t.Error("expected JSON config after start")
	}

	pretty := engine.ConfigPretty()
	if pretty == "{}" {
		t.Error("expected non-empty pretty config")
	}
}

// ---- Config Loader Tests ----

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	yaml := `
listen: ":443"
protocol: vless
uuid: "test-uuid"
reality:
  sni: "www.google.com"
  private_key: "test-key"
  short_ids: ["abc"]
fingerprint: "chrome"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.UUID != "test-uuid" {
		t.Errorf("expected UUID 'test-uuid', got %q", cfg.UUID)
	}
	if cfg.Reality.SNI != "www.google.com" {
		t.Errorf("expected SNI 'www.google.com', got %q", cfg.Reality.SNI)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// ---- splitHostPort Tests ----

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		input    string
		wantHost string
		wantPort int
		wantErr  bool
	}{
		{":443", "0.0.0.0", 443, false},
		{":8080", "0.0.0.0", 8080, false},
		{"127.0.0.1:1080", "127.0.0.1", 1080, false},
		{"", "", 0, true},
		{"noport", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			host, port, err := splitHostPort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitHostPort(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if host != tt.wantHost {
					t.Errorf("host = %q, want %q", host, tt.wantHost)
				}
				if port != tt.wantPort {
					t.Errorf("port = %d, want %d", port, tt.wantPort)
				}
			}
		})
	}
}
