package tunnel

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ---- Xray-core compatible JSON structures ----

type xrayFullConfig struct {
	Log       *xrayLog       `json:"log,omitempty"`
	Inbounds  []xrayInbound  `json:"inbounds"`
	Outbounds []xrayOutbound `json:"outbounds"`
}

type xrayLog struct {
	LogLevel string `json:"loglevel"`
}

type xrayInbound struct {
	Tag      string          `json:"tag"`
	Listen   string          `json:"listen,omitempty"`
	Port     int             `json:"port"`
	Protocol string          `json:"protocol"`
	Settings json.RawMessage `json:"settings"`
	Stream   *xrayStream     `json:"streamSettings,omitempty"`
}

type xrayOutbound struct {
	Tag      string          `json:"tag"`
	Protocol string          `json:"protocol"`
	Settings json.RawMessage `json:"settings,omitempty"`
	Stream   *xrayStream     `json:"streamSettings,omitempty"`
}

type xrayStream struct {
	Network  string              `json:"network"`
	Security string              `json:"security"`
	Reality  *xrayRealityStream  `json:"realitySettings,omitempty"`
	TLS      *xrayTLSStream      `json:"tlsSettings,omitempty"`
	WS       *xrayWSStream       `json:"wsSettings,omitempty"`
}

type xrayRealityStream struct {
	Show        bool     `json:"show"`
	Dest        string   `json:"dest,omitempty"`
	Xver        int      `json:"xver,omitempty"`
	ServerNames []string `json:"serverNames,omitempty"`
	PrivateKey  string   `json:"privateKey,omitempty"`
	ShortIDs    []string `json:"shortIds,omitempty"`
	// Client-side fields
	ServerName  string `json:"serverName,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortID     string `json:"shortId,omitempty"`
}

type xrayTLSStream struct {
	ServerName string `json:"serverName,omitempty"`
}

type xrayWSStream struct {
	Path string `json:"path"`
}

// ---- Server JSON builder ----

// BuildServerJSON produces xray-core compatible JSON for server mode.
func BuildServerJSON(cfg *Config) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	host, port, err := splitHostPort(cfg.Listen)
	if err != nil {
		return nil, fmt.Errorf("invalid listen address: %w", err)
	}

	shortIDs := cfg.Reality.ShortIDs
	if len(shortIDs) == 0 {
		shortIDs = []string{""}
	}

	// VLESS client settings
	vlessSettings, _ := json.Marshal(map[string]any{
		"clients": []map[string]any{
			{
				"id":   cfg.UUID,
				"flow": "xtls-rprx-vision",
			},
		},
		"decryption": "none",
	})

	xc := &xrayFullConfig{
		Log: &xrayLog{LogLevel: coalesce(cfg.LogLevel, "info")},
		Inbounds: []xrayInbound{
			{
				Tag:      "vless-reality",
				Listen:   host,
				Port:     port,
				Protocol: "vless",
				Settings: vlessSettings,
				Stream: &xrayStream{
					Network:  "tcp",
					Security: "reality",
					Reality: &xrayRealityStream{
						Show:        false,
						Dest:        fmt.Sprintf("%s:443", cfg.Reality.SNI),
						Xver:        0,
						ServerNames: []string{cfg.Reality.SNI},
						PrivateKey:  cfg.Reality.PrivateKey,
						ShortIDs:    shortIDs,
					},
				},
			},
		},
		Outbounds: []xrayOutbound{
			{Tag: "direct", Protocol: "freedom"},
			{Tag: "block", Protocol: "blackhole"},
		},
	}

	// Add fallback inbounds
	for i, fb := range cfg.Fallbacks {
		fbHost, fbPort, err := splitHostPort(fb.Listen)
		if err != nil {
			return nil, fmt.Errorf("invalid fallback listen address: %w", err)
		}

		fbSettings, _ := json.Marshal(map[string]any{})
		inbound := xrayInbound{
			Tag:      fmt.Sprintf("fallback-%s-%d", fb.Protocol, i),
			Listen:   fbHost,
			Port:     fbPort,
			Protocol: fb.Protocol,
			Settings: fbSettings,
		}

		if fb.Transport == "ws" {
			inbound.Stream = &xrayStream{
				Network:  "ws",
				Security: "tls",
				WS:       &xrayWSStream{Path: fb.Path},
			}
		}

		xc.Inbounds = append(xc.Inbounds, inbound)
	}

	return json.Marshal(xc)
}

// ---- Client JSON builder ----

// BuildClientJSON produces xray-core compatible JSON for client mode.
func BuildClientJSON(cfg *ClientConfig) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("client config is nil")
	}

	localHost, localPort, err := splitHostPort(cfg.LocalListen)
	if err != nil {
		return nil, fmt.Errorf("invalid local_listen: %w", err)
	}

	serverHost, serverPort := parseServerAddr(cfg.Server)

	// SOCKS5 inbound
	socksSettings, _ := json.Marshal(map[string]any{"udp": true})

	// VLESS outbound
	vlessOutSettings, _ := json.Marshal(map[string]any{
		"vnext": []map[string]any{
			{
				"address": serverHost,
				"port":    serverPort,
				"users": []map[string]any{
					{
						"id":         cfg.UUID,
						"encryption": "none",
						"flow":       "xtls-rprx-vision",
					},
				},
			},
		},
	})

	fingerprint := cfg.Fingerprint
	if fingerprint == "" {
		fingerprint = "chrome"
	}

	xc := &xrayFullConfig{
		Log: &xrayLog{LogLevel: coalesce(cfg.LogLevel, "info")},
		Inbounds: []xrayInbound{
			{
				Tag:      "socks-in",
				Listen:   localHost,
				Port:     localPort,
				Protocol: "socks",
				Settings: socksSettings,
			},
		},
		Outbounds: []xrayOutbound{
			{
				Tag:      "proxy",
				Protocol: "vless",
				Settings: vlessOutSettings,
				Stream: &xrayStream{
					Network:  "tcp",
					Security: "reality",
					Reality: &xrayRealityStream{
						Show:        false,
						ServerName:  cfg.SNI,
						Fingerprint: fingerprint,
						PublicKey:   cfg.PublicKey,
						ShortID:     cfg.ShortID,
					},
				},
			},
			{Tag: "direct", Protocol: "freedom"},
		},
	}

	// Add HTTP inbound alongside SOCKS
	if cfg.HTTPListen != "" {
		hHost, hPort, err := splitHostPort(cfg.HTTPListen)
		if err == nil {
			httpSettings, _ := json.Marshal(map[string]any{
				"allowTransparent": false,
			})
			xc.Inbounds = append(xc.Inbounds, xrayInbound{
				Tag:      "http-in",
				Listen:   hHost,
				Port:     hPort,
				Protocol: "http",
				Settings: httpSettings,
			})
		}
	}

	return json.Marshal(xc)
}

// ---- Helpers ----

func splitHostPort(addr string) (string, int, error) {
	if addr == "" {
		return "", 0, fmt.Errorf("empty address")
	}

	// Handle ":port"
	if addr[0] == ':' {
		port, err := strconv.Atoi(addr[1:])
		if err != nil {
			return "", 0, fmt.Errorf("invalid port in %q: %w", addr, err)
		}
		return "0.0.0.0", port, nil
	}

	// Handle "host:port"
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return "", 0, fmt.Errorf("no port in %q", addr)
	}

	host := addr[:idx]
	port, err := strconv.Atoi(addr[idx+1:])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in %q: %w", addr, err)
	}

	return host, port, nil
}

func parseServerAddr(addr string) (string, int) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, 443
	}
	host := addr[:idx]
	port, err := strconv.Atoi(addr[idx+1:])
	if err != nil {
		return addr, 443
	}
	return host, port
}

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
