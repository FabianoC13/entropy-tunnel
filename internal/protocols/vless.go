package protocols

import (
	"context"
	"fmt"
	"net"
)

// VLESSProtocol implements the Protocol interface for VLESS connections.
// This wraps the Xray-core VLESS transport.
type VLESSProtocol struct {
	available bool
}

// NewVLESS creates a new VLESS protocol adapter.
func NewVLESS() *VLESSProtocol {
	return &VLESSProtocol{available: true}
}

func (v *VLESSProtocol) Name() string     { return "vless" }
func (v *VLESSProtocol) Priority() int     { return 1 } // Highest priority
func (v *VLESSProtocol) Available() bool   { return v.available }

func (v *VLESSProtocol) DialContext(ctx context.Context, addr string) (net.Conn, error) {
	// TODO: Implement via Xray-core VLESS outbound
	return nil, fmt.Errorf("VLESS dial not yet implemented (requires xray-core integration)")
}

func (v *VLESSProtocol) Listen(addr string) (net.Listener, error) {
	// TODO: Implement via Xray-core VLESS inbound
	return nil, fmt.Errorf("VLESS listen not yet implemented (requires xray-core integration)")
}
