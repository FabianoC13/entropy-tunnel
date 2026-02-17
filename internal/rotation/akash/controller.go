// Package akash provides Akash Network deployment integration for EntropyTunnel.
package akash

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/fabiano/entropy-tunnel/internal/rotation"
)

// Controller implements rotation.Controller for Akash Network.
type Controller struct {
	client       *Client
	sdlPath      string
	logger       *zap.Logger
	mu           sync.RWMutex
	endpoints    []*rotation.Endpoint
	deployments  map[string]*DeploymentInfo
	stopCh       chan struct{}
	counter      int
}

// Config holds configuration for the Akash controller.
type Config struct {
	APIKey  string
	SDLPath string
}

// NewController creates a new Akash rotation controller.
func NewController(cfg Config, logger *zap.Logger) (*Controller, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Akash API key is required")
	}
	if cfg.SDLPath == "" {
		cfg.SDLPath = "deployments/akash/xray-server.yaml"
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Controller{
		client:      NewClient(cfg.APIKey, logger),
		sdlPath:     cfg.SDLPath,
		logger:      logger,
		endpoints:   make([]*rotation.Endpoint, 0),
		deployments: make(map[string]*DeploymentInfo),
		stopCh:      make(chan struct{}),
	}, nil
}

// Rotate creates a new Akash deployment and returns the endpoint.
func (c *Controller) Rotate(ctx context.Context) (*rotation.Endpoint, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.counter++
	logger := c.logger.With(zap.Int("rotation", c.counter))

	logger.Info("rotating to new Akash deployment")

	// Deploy to Akash
	deployInfo, err := c.client.Deploy(ctx, c.sdlPath)
	if err != nil {
		return nil, fmt.Errorf("deploying to Akash: %w", err)
	}

	// Wait for lease
	deployInfo, err = c.client.WaitForLease(ctx, deployInfo.DSeq, 5*time.Minute)
	if err != nil {
		// Try to cleanup failed deployment
		_ = c.client.CloseDeployment(ctx, deployInfo.DSeq)
		return nil, fmt.Errorf("waiting for lease: %w", err)
	}

	// Get credentials from container
	creds, err := c.client.GetCredentials(ctx, deployInfo.DSeq)
	if err != nil {
		logger.Warn("failed to get credentials from logs, using deployment info", zap.Error(err))
		creds = &Credentials{
			UUID:     "", // Will be fetched from container
			ShortID:  "abcdef01",
			Hostname: deployInfo.URI,
		}
	}

	// Use URI from deployment if hostname not available
	address := creds.Hostname
	if address == "" {
		address = deployInfo.URI
	}
	if address == "" {
		_ = c.client.CloseDeployment(ctx, deployInfo.DSeq)
		return nil, fmt.Errorf("no address available from deployment")
	}

	// Create endpoint
	ep := &rotation.Endpoint{
		ID:        fmt.Sprintf("akash-%s", deployInfo.DSeq),
		Address:   fmt.Sprintf("%s:443", address),
		Region:    detectRegion(deployInfo.Provider),
		Provider:  "akash",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Akash leases are typically 24h
		Metadata: map[string]string{
			"dseq":       deployInfo.DSeq,
			"provider":   deployInfo.Provider,
			"uuid":       creds.UUID,
			"public_key": creds.PublicKey,
			"short_id":   creds.ShortID,
		},
	}

	c.endpoints = append(c.endpoints, ep)
	c.deployments[ep.ID] = deployInfo

	logger.Info("Akash endpoint rotated",
		zap.String("id", ep.ID),
		zap.String("address", ep.Address),
		zap.String("provider", deployInfo.Provider),
	)

	return ep, nil
}

// Retire closes the Akash deployment.
func (c *Controller) Retire(ctx context.Context, ep *rotation.Endpoint) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find deployment
	deployInfo, ok := c.deployments[ep.ID]
	if !ok {
		return fmt.Errorf("deployment not found for endpoint %s", ep.ID)
	}

	// Close deployment
	if err := c.client.CloseDeployment(ctx, deployInfo.DSeq); err != nil {
		c.logger.Warn("failed to close deployment", zap.Error(err), zap.String("dseq", deployInfo.DSeq))
	}

	// Remove from tracking
	delete(c.deployments, ep.ID)
	for i, e := range c.endpoints {
		if e.ID == ep.ID {
			c.endpoints = append(c.endpoints[:i], c.endpoints[i+1:]...)
			break
		}
	}

	c.logger.Info("Akash endpoint retired", zap.String("id", ep.ID))
	return nil
}

// ActiveEndpoints returns all active Akash endpoints.
func (c *Controller) ActiveEndpoints() []*rotation.Endpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()

	active := make([]*rotation.Endpoint, 0)
	for _, ep := range c.endpoints {
		if !ep.IsExpired() {
			active = append(active, ep)
		}
	}
	return active
}

// StartAutoRotation begins automatic rotation at the given interval.
func (c *Controller) StartAutoRotation(ctx context.Context, interval time.Duration) error {
	c.logger.Info("auto-rotation started for Akash", zap.Duration("interval", interval))

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
				// Rotate to new endpoint
				newEp, err := c.Rotate(ctx)
				if err != nil {
					c.logger.Error("auto-rotation failed", zap.Error(err))
					continue
				}

				// Retire old endpoints (keep only last 2)
				c.mu.Lock()
				if len(c.endpoints) > 2 {
					oldEp := c.endpoints[0]
					go func() {
						_ = c.Retire(ctx, oldEp)
					}()
				}
				c.mu.Unlock()

				c.logger.Info("auto-rotated to new Akash endpoint",
					zap.String("new_id", newEp.ID),
					zap.String("address", newEp.Address))
			}
		}
	}()

	return nil
}

// StopAutoRotation halts automatic rotation.
func (c *Controller) StopAutoRotation() {
	close(c.stopCh)
	c.stopCh = make(chan struct{})
	c.logger.Info("auto-rotation stopped for Akash")
}

// RotateToAkash is a convenience method that immediately switches to Akash.
func (c *Controller) RotateToAkash(ctx context.Context) (*rotation.Endpoint, error) {
	return c.Rotate(ctx)
}

// detectRegion attempts to detect region from provider info.
func detectRegion(provider string) string {
	// Akash providers are distributed globally
	// This is a simplified mapping
	provider = strings.ToLower(provider)
	switch {
	case strings.Contains(provider, "us") || strings.Contains(provider, "usa") || strings.Contains(provider, "america"):
		return "us-east"
	case strings.Contains(provider, "eu") || strings.Contains(provider, "europe") || strings.Contains(provider, "de") || strings.Contains(provider, "fr"):
		return "eu-west"
	case strings.Contains(provider, "asia") || strings.Contains(provider, "sg") || strings.Contains(provider, "jp"):
		return "ap-southeast"
	default:
		return "global"
	}
}
