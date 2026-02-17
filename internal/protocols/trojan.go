package protocols

import (
	"context"
	"fmt"
	"net"
)

// TrojanProtocol implements the Protocol interface for Trojan connections.
// Uses WebSocket transport for CDN compatibility.
type TrojanProtocol struct {
	available bool
	wsPath    string
}

// NewTrojan creates a new Trojan protocol adapter.
func NewTrojan(wsPath string) *TrojanProtocol {
	if wsPath == "" {
		wsPath = "/ws"
	}
	return &TrojanProtocol{
		available: true,
		wsPath:    wsPath,
	}
}

func (t *TrojanProtocol) Name() string     { return "trojan" }
func (t *TrojanProtocol) Priority() int     { return 2 } // Fallback after VLESS
func (t *TrojanProtocol) Available() bool   { return t.available }

func (t *TrojanProtocol) DialContext(ctx context.Context, addr string) (net.Conn, error) {
	// TODO: Implement Trojan-GO WebSocket dial
	return nil, fmt.Errorf("trojan dial not yet implemented")
}

func (t *TrojanProtocol) Listen(addr string) (net.Listener, error) {
	// TODO: Implement Trojan-GO WebSocket listener
	return nil, fmt.Errorf("trojan listen not yet implemented")
}

// WSPath returns the WebSocket path for this Trojan instance.
func (t *TrojanProtocol) WSPath() string {
	return t.wsPath
}
