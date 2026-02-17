package rotation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// CloudflareController rotates endpoints via Cloudflare Workers.
type CloudflareController struct {
	NoOpController
	apiToken  string
	accountID string
	zoneID    string
	client    *http.Client
}

// NewCloudflareController creates a Cloudflare Workers rotation backend.
func NewCloudflareController(apiToken, accountID, zoneID string, logger *zap.Logger) *CloudflareController {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CloudflareController{
		NoOpController: NoOpController{
			endpoints: make([]*Endpoint, 0),
			logger:    logger,
			stopCh:    make(chan struct{}),
		},
		apiToken:  apiToken,
		accountID: accountID,
		zoneID:    zoneID,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Rotate deploys a new Cloudflare Worker and updates DNS.
func (c *CloudflareController) Rotate(ctx context.Context) (*Endpoint, error) {
	c.mu.Lock()
	c.counter++
	workerName := fmt.Sprintf("entropy-worker-%d-%d", time.Now().Unix(), c.counter)
	c.mu.Unlock()

	c.logger.Info("deploying new cloudflare worker",
		zap.String("name", workerName),
	)

	// 1. Deploy Worker
	if err := c.deployWorker(ctx, workerName); err != nil {
		return nil, fmt.Errorf("deploy worker %s: %w", workerName, err)
	}

	// 2. Create DNS record pointing to the worker
	workerDomain := fmt.Sprintf("%s.workers.dev", workerName)

	ep := &Endpoint{
		ID:        workerName,
		Address:   workerDomain,
		Region:    "global", // Cloudflare Workers are global anycast
		Provider:  "cloudflare",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Metadata: map[string]string{
			"worker_name": workerName,
			"type":        "workers",
		},
	}

	c.mu.Lock()
	c.endpoints = append(c.endpoints, ep)
	c.mu.Unlock()

	c.logger.Info("cloudflare worker deployed",
		zap.String("id", ep.ID),
		zap.String("domain", workerDomain),
	)

	return ep, nil
}

// Retire tears down a Cloudflare Worker endpoint.
func (c *CloudflareController) Retire(ctx context.Context, ep *Endpoint) error {
	if ep.Provider != "cloudflare" {
		return c.NoOpController.Retire(ctx, ep)
	}

	workerName := ep.ID
	c.logger.Info("retiring cloudflare worker", zap.String("name", workerName))

	if err := c.deleteWorker(ctx, workerName); err != nil {
		c.logger.Warn("failed to delete worker (may not exist)", zap.Error(err))
	}

	// Remove from active list
	c.mu.Lock()
	for i, e := range c.endpoints {
		if e.ID == ep.ID {
			c.endpoints = append(c.endpoints[:i], c.endpoints[i+1:]...)
			break
		}
	}
	c.mu.Unlock()

	return nil
}

// deployWorker pushes a new Worker script to the Cloudflare API.
func (c *CloudflareController) deployWorker(ctx context.Context, name string) error {
	// Worker script: forwards VLESS/WS connections to the actual tunnel server.
	workerScript := `
export default {
    async fetch(request) {
        const url = new URL(request.url);
        // Forward to actual tunnel server via WebSocket upgrade
        if (request.headers.get("Upgrade") === "websocket") {
            const upstream = new URL(url.pathname, "wss://YOUR_TUNNEL_SERVER");
            return fetch(new Request(upstream, request));
        }
        // Decoy: return plausible web content
        return new Response("<!DOCTYPE html><html><body><h1>Welcome</h1></body></html>", {
            headers: { "content-type": "text/html" },
        });
    }
};`

	apiURL := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/workers/scripts/%s",
		c.accountID, name,
	)

	// Workers API uses multipart form for script upload
	body := bytes.NewBufferString(workerScript)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/javascript")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("cloudflare API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// deleteWorker removes a Worker script via the Cloudflare API.
func (c *CloudflareController) deleteWorker(ctx context.Context, name string) error {
	apiURL := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/workers/scripts/%s",
		c.accountID, name,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudflare delete error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// UpdateDNS creates/updates a DNS record pointing to a worker endpoint.
func (c *CloudflareController) UpdateDNS(ctx context.Context, recordName, target string) error {
	apiURL := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/zones/%s/dns_records",
		c.zoneID,
	)

	payload, _ := json.Marshal(map[string]any{
		"type":    "CNAME",
		"name":    recordName,
		"content": target,
		"ttl":     60,
		"proxied": true,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DNS update error %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.Info("DNS record updated",
		zap.String("name", recordName),
		zap.String("target", target),
	)
	return nil
}
