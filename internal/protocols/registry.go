package protocols

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// Protocol defines the interface for a tunnel protocol.
type Protocol interface {
	// Name returns the protocol identifier.
	Name() string

	// DialContext connects to a remote address using this protocol.
	DialContext(ctx context.Context, addr string) (net.Conn, error)

	// Listen starts listening for incoming connections.
	Listen(addr string) (net.Listener, error)

	// Priority returns the protocol priority (lower = preferred).
	Priority() int

	// Available reports whether the protocol is currently usable.
	Available() bool
}

// Registry manages available protocols and provides selection logic.
type Registry struct {
	mu        sync.RWMutex
	protocols map[string]Protocol
}

// NewRegistry creates an empty protocol registry.
func NewRegistry() *Registry {
	return &Registry{
		protocols: make(map[string]Protocol),
	}
}

// Register adds a protocol to the registry.
func (r *Registry) Register(p Protocol) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.protocols[p.Name()]; exists {
		return fmt.Errorf("protocol %q already registered", p.Name())
	}

	r.protocols[p.Name()] = p
	return nil
}

// Get returns a protocol by name.
func (r *Registry) Get(name string) (Protocol, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.protocols[name]
	if !ok {
		return nil, fmt.Errorf("protocol %q not found", name)
	}
	return p, nil
}

// List returns all registered protocol names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.protocols))
	for name := range r.protocols {
		names = append(names, name)
	}
	return names
}

// SelectBest returns the highest-priority available protocol.
func (r *Registry) SelectBest() (Protocol, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best Protocol
	for _, p := range r.protocols {
		if !p.Available() {
			continue
		}
		if best == nil || p.Priority() < best.Priority() {
			best = p
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no available protocols")
	}
	return best, nil
}

// FallbackChain returns protocols in priority order for sequential fallback.
func (r *Registry) FallbackChain() []Protocol {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chain := make([]Protocol, 0, len(r.protocols))
	for _, p := range r.protocols {
		chain = append(chain, p)
	}

	// Sort by priority (simple insertion sort for small N)
	for i := 1; i < len(chain); i++ {
		for j := i; j > 0 && chain[j].Priority() < chain[j-1].Priority(); j-- {
			chain[j], chain[j-1] = chain[j-1], chain[j]
		}
	}

	return chain
}
