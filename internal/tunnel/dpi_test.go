package tunnel

import (
	"encoding/json"
	"testing"
)

// TestDPIResistance_RealityStreamConfig verifies the generated JSON
// contains the correct Reality stream settings that defeat DPI.
// A real DPI engine inspects the TLS ClientHello for known proxy fingerprints.
func TestDPIResistance_RealityStreamConfig(t *testing.T) {
	cfg := &Config{
		Listen: ":443", Protocol: "vless", UUID: "uuid",
		Reality: RealityConfig{
			SNI:        "www.microsoft.com",
			PrivateKey: "test-key",
			ShortIDs:   []string{"deadbeef"},
		},
	}

	jsonBytes, err := BuildServerJSON(cfg)
	if err != nil {
		t.Fatalf("BuildServerJSON: %v", err)
	}

	var parsed map[string]any
	json.Unmarshal(jsonBytes, &parsed)

	inbounds := parsed["inbounds"].([]any)
	primary := inbounds[0].(map[string]any)
	stream := primary["streamSettings"].(map[string]any)

	// DPI check 1: Security must be "reality" (not "tls" which is fingerprintable)
	if stream["security"] != "reality" {
		t.Errorf("DPI: security = %q, want 'reality'", stream["security"])
	}

	reality := stream["realitySettings"].(map[string]any)

	// DPI check 2: Dest must point to a real HTTPS site
	if reality["dest"] != "www.microsoft.com:443" {
		t.Errorf("DPI: dest = %q, want 'www.microsoft.com:443'", reality["dest"])
	}

	// DPI check 3: ServerNames must include the SNI target
	serverNames := reality["serverNames"].([]any)
	found := false
	for _, sn := range serverNames {
		if sn == "www.microsoft.com" {
			found = true
		}
	}
	if !found {
		t.Error("DPI: serverNames missing 'www.microsoft.com'")
	}

	// DPI check 4: show must be false (don't expose Reality in errors)
	if reality["show"] != false {
		t.Error("DPI: show should be false to avoid detection")
	}
}

// TestDPIResistance_ClientFingerprint verifies the client config uses
// a browser fingerprint that matches real browser TLS ClientHello.
func TestDPIResistance_ClientFingerprint(t *testing.T) {
	fingerprints := []string{"chrome", "firefox", "safari", "edge"}

	for _, fp := range fingerprints {
		t.Run(fp, func(t *testing.T) {
			cfg := &ClientConfig{
				Server: "1.2.3.4:443", UUID: "u", SNI: "g.com",
				PublicKey: "pk", Fingerprint: fp, LocalListen: "127.0.0.1:1080",
			}

			jsonBytes, err := BuildClientJSON(cfg)
			if err != nil {
				t.Fatalf("BuildClientJSON: %v", err)
			}

			var parsed map[string]any
			json.Unmarshal(jsonBytes, &parsed)

			outbounds := parsed["outbounds"].([]any)
			proxy := outbounds[0].(map[string]any)
			stream := proxy["streamSettings"].(map[string]any)
			reality := stream["realitySettings"].(map[string]any)

			if reality["fingerprint"] != fp {
				t.Errorf("DPI: fingerprint = %q, want %q", reality["fingerprint"], fp)
			}
		})
	}
}

// TestDPIResistance_NoLeakedFields verifies the JSON config doesn't
// contain metadata that would identify it as a proxy config.
func TestDPIResistance_NoLeakedFields(t *testing.T) {
	cfg := &Config{
		Listen: ":443", UUID: "u",
		Reality: RealityConfig{SNI: "g.com", PrivateKey: "k"},
	}

	jsonBytes, _ := BuildServerJSON(cfg)
	raw := string(jsonBytes)

	// Should NOT contain identifying strings
	leakyStrings := []string{
		"entropy", "tunnel", "vpn", "proxy",
		"shadowsocks", "v2ray", "xray",
	}
	for _, s := range leakyStrings {
		if containsCI(raw, s) {
			t.Errorf("DPI: config JSON contains identifying string %q", s)
		}
	}
}

// TestDPIResistance_FallbackDecoy verifies that fallback inbounds
// are configured to return believable content to non-VPN traffic.
func TestDPIResistance_FallbackDecoy(t *testing.T) {
	cfg := &Config{
		Listen: ":443", UUID: "u",
		Reality: RealityConfig{SNI: "www.google.com", PrivateKey: "k"},
		Fallbacks: []FallbackConfig{
			{Protocol: "trojan", Listen: ":8443", Transport: "ws", Path: "/ws"},
		},
	}

	jsonBytes, err := BuildServerJSON(cfg)
	if err != nil {
		t.Fatalf("BuildServerJSON: %v", err)
	}

	var parsed xrayFullConfig
	json.Unmarshal(jsonBytes, &parsed)

	if len(parsed.Inbounds) < 2 {
		t.Fatal("expected fallback inbound")
	}

	fb := parsed.Inbounds[1]
	if fb.Protocol != "trojan" {
		t.Errorf("fallback protocol = %q, want 'trojan'", fb.Protocol)
	}
	if fb.Stream == nil || fb.Stream.WS == nil {
		t.Error("fallback should have WebSocket stream settings")
	}
}

func containsCI(haystack, needle string) bool {
	// Case-insensitive contains
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			h := haystack[i+j]
			n := needle[j]
			if h >= 'A' && h <= 'Z' {
				h += 32
			}
			if n >= 'A' && n <= 'Z' {
				n += 32
			}
			if h != n {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
