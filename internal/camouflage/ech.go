package camouflage

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// ECHConfig represents Encrypted Client Hello configuration.
// ECH hides the true SNI inside an encrypted extension,
// preventing ISPs from seeing which domain you connect to.
type ECHConfig struct {
	// Enabled controls whether ECH is used.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// PublicName is the outer SNI visible to the network (e.g., "cloudflare-ech.com").
	PublicName string `json:"public_name" yaml:"public_name"`

	// ConfigList is the base64-encoded ECHConfigList from DNS HTTPS records.
	ConfigList string `json:"config_list" yaml:"config_list"`
}

// ECHMode type for selecting ECH behavior.
type ECHMode string

const (
	ECHModeDisabled ECHMode = "disabled"
	ECHModeGrease   ECHMode = "grease"   // Send GREASE ECH extension (camouflage only)
	ECHModeFull     ECHMode = "full"     // Full ECH with real config
)

// GenerateGreaseECH creates a GREASE (fake) ECH extension payload.
// GREASE ECH looks like real ECH to passive observers but doesn't
// actually hide the SNI. Useful for blending in with browsers that
// send ECH extensions.
func GenerateGreaseECH() ([]byte, error) {
	// GREASE ECH payload: random bytes that look like an ECH extension.
	// Format: 2-byte version (0xfe0d) + random payload (128-192 bytes)
	payloadLen := 128 + randInt(64)
	payload := make([]byte, payloadLen+2)

	// ECH version 0xfe0d (draft-ietf-tls-esni)
	payload[0] = 0xfe
	payload[1] = 0x0d

	// Random payload
	if _, err := rand.Read(payload[2:]); err != nil {
		return nil, fmt.Errorf("failed to generate GREASE ECH: %w", err)
	}

	return payload, nil
}

// EncodeECHConfigList encodes an ECH config for use in TLS ClientHello.
func EncodeECHConfigList(publicName string, publicKey []byte) string {
	// Simplified ECHConfigList encoding for the outer config.
	// In production, this would parse real DNS HTTPS records.
	raw := make([]byte, 0, 64)

	// Version: 0xfe0d
	raw = append(raw, 0xfe, 0x0d)

	// Length placeholder (will fill later)
	raw = append(raw, 0x00, 0x00)

	// Contents (simplified)
	nameBytes := []byte(publicName)
	raw = append(raw, byte(len(nameBytes)))
	raw = append(raw, nameBytes...)

	if len(publicKey) > 0 {
		raw = append(raw, publicKey...)
	}

	// Fill in length
	contentLen := len(raw) - 4
	raw[2] = byte(contentLen >> 8)
	raw[3] = byte(contentLen)

	return base64.StdEncoding.EncodeToString(raw)
}

// ValidateECHConfig checks if an ECH configuration is valid.
func ValidateECHConfig(cfg *ECHConfig) error {
	if cfg == nil {
		return fmt.Errorf("ECH config is nil")
	}
	if !cfg.Enabled {
		return nil // Nothing to validate if disabled
	}
	if cfg.PublicName == "" {
		return fmt.Errorf("ECH public_name is required when enabled")
	}
	return nil
}

func randInt(max int) int {
	if max <= 0 {
		return 0
	}
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return int(b[0]) % max
}
