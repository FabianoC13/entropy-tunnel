package camouflage

import (
	"encoding/base64"
	"testing"
)

func TestGenerateGreaseECH(t *testing.T) {
	payload, err := GenerateGreaseECH()
	if err != nil {
		t.Fatalf("GenerateGreaseECH() error = %v", err)
	}

	// Should start with 0xfe0d version
	if len(payload) < 2 {
		t.Fatal("payload too short")
	}
	if payload[0] != 0xfe || payload[1] != 0x0d {
		t.Errorf("expected version 0xfe0d, got 0x%02x%02x", payload[0], payload[1])
	}

	// Should be between 130 and 194 bytes (2 header + 128-192 random)
	if len(payload) < 130 || len(payload) > 194 {
		t.Errorf("unexpected payload length %d (expected 130-194)", len(payload))
	}
}

func TestGenerateGreaseECH_Randomness(t *testing.T) {
	p1, _ := GenerateGreaseECH()
	p2, _ := GenerateGreaseECH()

	// Very unlikely to be identical (random content)
	if len(p1) == len(p2) {
		same := true
		for i := range p1 {
			if p1[i] != p2[i] {
				same = false
				break
			}
		}
		if same {
			t.Error("two GREASE payloads are identical (should be random)")
		}
	}
}

func TestEncodeECHConfigList(t *testing.T) {
	result := EncodeECHConfigList("cloudflare.com", nil)
	if result == "" {
		t.Error("EncodeECHConfigList returned empty string")
	}

	// Should be valid base64
	_, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Errorf("result is not valid base64: %v", err)
	}
}

func TestEncodeECHConfigList_WithPublicKey(t *testing.T) {
	pubKey := []byte{0x04, 0x01, 0x02, 0x03}
	result := EncodeECHConfigList("example.com", pubKey)
	if result == "" {
		t.Error("expected non-empty result")
	}

	decoded, _ := base64.StdEncoding.DecodeString(result)
	if len(decoded) < 6 {
		t.Error("decoded result too short")
	}
}

func TestValidateECHConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ECHConfig
		wantErr bool
	}{
		{"nil config", nil, true},
		{"disabled", &ECHConfig{Enabled: false}, false},
		{"enabled without name", &ECHConfig{Enabled: true}, true},
		{"valid", &ECHConfig{Enabled: true, PublicName: "cf.com"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateECHConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateECHConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
