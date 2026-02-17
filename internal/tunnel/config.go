package tunnel

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the server-side tunnel configuration.
type Config struct {
	Listen      string           `yaml:"listen"`
	Protocol    string           `yaml:"protocol"`
	UUID        string           `yaml:"uuid"`
	Reality     RealityConfig    `yaml:"reality"`
	Fingerprint string           `yaml:"fingerprint"`
	Fallbacks   []FallbackConfig `yaml:"fallbacks"`
	LogLevel    string           `yaml:"log_level"`

	// SportsMode enables low-latency + extra noise for streaming.
	SportsMode bool `yaml:"sports_mode"`

	// Rotation settings (Phase 3).
	Rotation RotationConfig `yaml:"rotation"`

	// Payment settings.
	Payment PaymentConfig `yaml:"payment"`
}

// RealityConfig holds XTLS-Reality settings.
type RealityConfig struct {
	SNI        string   `yaml:"sni"`
	PrivateKey string   `yaml:"private_key"`
	PublicKey  string   `yaml:"public_key"`
	ShortIDs   []string `yaml:"short_ids"`
}

// FallbackConfig defines a protocol fallback.
type FallbackConfig struct {
	Protocol  string `yaml:"protocol"`
	Listen    string `yaml:"listen"`
	Transport string `yaml:"transport"`
	Path      string `yaml:"path"`
}

// RotationConfig holds dynamic endpoint rotation settings.
type RotationConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Provider   string `yaml:"provider"`   // "cloudflare", "aws", "noop"
	Interval   string `yaml:"interval"`   // e.g. "30m"
	CFAPIToken string `yaml:"cf_api_token"`
	CFAccountID string `yaml:"cf_account_id"`
	CFZoneID   string `yaml:"cf_zone_id"`
	AWSRegion  string `yaml:"aws_region"`
	AWSKey     string `yaml:"aws_access_key"`
	AWSSecret  string `yaml:"aws_secret_key"`
}

// PaymentConfig holds BTCPay Server settings.
type PaymentConfig struct {
	Enabled    bool   `yaml:"enabled"`
	BTCPayURL  string `yaml:"btcpay_url"`
	BTCPayKey  string `yaml:"btcpay_api_key"`
	StoreID    string `yaml:"btcpay_store_id"`
}

// Validate checks the configuration for required fields.
func (c *Config) Validate() error {
	if c.Listen == "" {
		return fmt.Errorf("listen address is required")
	}
	if c.Protocol == "" {
		c.Protocol = "vless"
	}
	if c.UUID == "" {
		return fmt.Errorf("UUID is required")
	}
	if c.Reality.SNI == "" {
		return fmt.Errorf("reality.sni is required")
	}
	if c.Reality.PrivateKey == "" {
		return fmt.Errorf("reality.private_key is required")
	}
	if c.Fingerprint == "" {
		c.Fingerprint = "chrome"
	}

	validProtocols := map[string]bool{"vless": true, "trojan": true}
	if !validProtocols[c.Protocol] {
		return fmt.Errorf("unsupported protocol: %s (supported: vless, trojan)", c.Protocol)
	}

	return nil
}

// LoadConfig reads and parses a YAML configuration file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}
	return &cfg, nil
}

// ClientConfig holds the client-side configuration.
type ClientConfig struct {
	Server      string `yaml:"server"`
	UUID        string `yaml:"uuid"`
	SNI         string `yaml:"sni"`
	Fingerprint string `yaml:"fingerprint"`
	PublicKey   string `yaml:"public_key"`
	ShortID     string `yaml:"short_id"`
	LocalListen string `yaml:"local_listen"`
	HTTPListen  string `yaml:"http_listen"`
	LogLevel    string `yaml:"log_level"`

	// SportsMode for low-latency + extra noise.
	SportsMode bool `yaml:"sports_mode"`

	// APIListen is the local HTTP API address for GUI integration.
	APIListen string `yaml:"api_listen"`
}

// LoadClientConfig reads and parses a client YAML configuration file.
func LoadClientConfig(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read client config file %s: %w", path, err)
	}
	var cfg ClientConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse client config file %s: %w", path, err)
	}
	return &cfg, nil
}

// Validate checks the client configuration.
func (c *ClientConfig) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("server address is required")
	}
	if c.UUID == "" {
		return fmt.Errorf("UUID is required")
	}
	if c.SNI == "" {
		return fmt.Errorf("SNI is required")
	}
	if c.PublicKey == "" {
		return fmt.Errorf("public_key is required")
	}
	if c.LocalListen == "" {
		c.LocalListen = "127.0.0.1:1080"
	}
	if c.Fingerprint == "" {
		c.Fingerprint = "chrome"
	}
	if c.APIListen == "" {
		c.APIListen = "127.0.0.1:9876"
	}
	return nil
}
