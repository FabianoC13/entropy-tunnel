package rotation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Endpoint represents a tunnel endpoint (IP/domain + metadata).
type Endpoint struct {
	ID        string            `json:"id"`
	Address   string            `json:"address"`
	Region    string            `json:"region"`
	Provider  string            `json:"provider"` // "cloudflare", "aws", "self-hosted"
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// IsExpired reports whether the endpoint has passed its expiry time.
func (e *Endpoint) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Controller manages dynamic endpoint rotation.
type Controller interface {
	// Rotate provisions a new endpoint and returns it.
	Rotate(ctx context.Context) (*Endpoint, error)

	// Retire tears down an existing endpoint.
	Retire(ctx context.Context, ep *Endpoint) error

	// ActiveEndpoints returns all currently active endpoints.
	ActiveEndpoints() []*Endpoint

	// StartAutoRotation begins automatic rotation at the given interval.
	StartAutoRotation(ctx context.Context, interval time.Duration) error

	// StopAutoRotation halts automatic rotation.
	StopAutoRotation()
}

// NoOpController is a placeholder controller for Phase 2.
// It manages a static list of endpoints without actual cloud provisioning.
type NoOpController struct {
	mu        sync.RWMutex
	endpoints []*Endpoint
	logger    *zap.Logger
	stopCh    chan struct{}
	counter   int
}

// NewNoOpController creates a no-op rotation controller.
func NewNoOpController(logger *zap.Logger) *NoOpController {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &NoOpController{
		endpoints: make([]*Endpoint, 0),
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

func (c *NoOpController) Rotate(ctx context.Context) (*Endpoint, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.counter++
	ep := &Endpoint{
		ID:        fmt.Sprintf("noop-%d", c.counter),
		Address:   fmt.Sprintf("127.0.0.1:%d", 10000+c.counter),
		Region:    "local",
		Provider:  "noop",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	c.endpoints = append(c.endpoints, ep)
	c.logger.Info("rotated endpoint (noop)",
		zap.String("id", ep.ID),
		zap.String("address", ep.Address),
	)

	return ep, nil
}

func (c *NoOpController) Retire(ctx context.Context, ep *Endpoint) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, e := range c.endpoints {
		if e.ID == ep.ID {
			c.endpoints = append(c.endpoints[:i], c.endpoints[i+1:]...)
			c.logger.Info("retired endpoint (noop)", zap.String("id", ep.ID))
			return nil
		}
	}

	return fmt.Errorf("endpoint %s not found", ep.ID)
}

func (c *NoOpController) ActiveEndpoints() []*Endpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()

	active := make([]*Endpoint, 0)
	for _, ep := range c.endpoints {
		if !ep.IsExpired() {
			active = append(active, ep)
		}
	}
	return active
}

func (c *NoOpController) StartAutoRotation(ctx context.Context, interval time.Duration) error {
	c.logger.Info("auto-rotation started (noop)", zap.Duration("interval", interval))

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stopCh:
				return
			case <-ticker.C:
				if _, err := c.Rotate(ctx); err != nil {
					c.logger.Error("auto-rotation failed", zap.Error(err))
				}
				// Retire expired endpoints
				c.mu.RLock()
				for _, ep := range c.endpoints {
					if ep.IsExpired() {
						go func(ep *Endpoint) {
							_ = c.Retire(ctx, ep)
						}(ep)
					}
				}
				c.mu.RUnlock()
			}
		}
	}()

	return nil
}

func (c *NoOpController) StopAutoRotation() {
	close(c.stopCh)
	c.stopCh = make(chan struct{})
	c.logger.Info("auto-rotation stopped (noop)")
}
