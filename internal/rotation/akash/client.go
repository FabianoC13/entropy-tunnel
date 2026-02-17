// Package akash provides Akash Network deployment integration for EntropyTunnel.
package akash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/fabiano/entropy-tunnel/internal/rotation"
)

const (
	akashAPIBase    = "https://api.cloudmos.io/v1"
	akashConsoleAPI = "https://console.akash.network/api/v1"
)

// Credentials holds the Xray server credentials generated in the container.
type Credentials struct {
	UUID      string `json:"uuid"`
	PublicKey string `json:"public_key"`
	ShortID   string `json:"short_id"`
	Hostname  string `json:"hostname"`
}

// DeploymentInfo holds Akash deployment details.
type DeploymentInfo struct {
	DSeq      string    `json:"dseq"`
	GSeq      int       `json:"gseq"`
	OSeq      int       `json:"oseq"`
	Provider  string    `json:"provider"`
	LeaseID   string    `json:"lease_id"`
	Status    string    `json:"status"`
	URI       string    `json:"uri,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Client provides Akash Network API interactions.
type Client struct {
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new Akash API client.
func NewClient(apiKey string, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Deploy creates a new deployment on Akash Network.
func (c *Client) Deploy(ctx context.Context, sdlPath string) (*DeploymentInfo, error) {
	c.logger.Info("deploying to Akash", zap.String("sdl", sdlPath))

	// Read SDL file
	sdlData, err := os.ReadFile(sdlPath)
	if err != nil {
		return nil, fmt.Errorf("reading SDL: %w", err)
	}

	// Create deployment via Cloudmos API
	payload := map[string]interface{}{
		"sdl": string(sdlData),
	}

	body, err := c.post(ctx, akashAPIBase+"/deployments", payload)
	if err != nil {
		return nil, fmt.Errorf("creating deployment: %w", err)
	}

	var result struct {
		DSeq string `json:"dseq"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing deployment response: %w", err)
	}

	info := &DeploymentInfo{
		DSeq:      result.DSeq,
		GSeq:      1,
		OSeq:      1,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	c.logger.Info("deployment created", zap.String("dseq", info.DSeq))
	return info, nil
}

// WaitForLease waits for the deployment to be leased and returns provider info.
func (c *Client) WaitForLease(ctx context.Context, dseq string, timeout time.Duration) (*DeploymentInfo, error) {
	c.logger.Info("waiting for lease", zap.String("dseq", dseq), zap.Duration("timeout", timeout))

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for lease")
		case <-ticker.C:
			info, err := c.GetDeployment(ctx, dseq)
			if err != nil {
				c.logger.Warn("failed to get deployment status", zap.Error(err))
				continue
			}

			if info.Status == "active" && info.Provider != "" {
				c.logger.Info("lease acquired",
					zap.String("provider", info.Provider),
					zap.String("uri", info.URI))
				return info, nil
			}

			c.logger.Info("deployment pending", zap.String("status", info.Status))
		}
	}
}

// GetDeployment retrieves deployment status.
func (c *Client) GetDeployment(ctx context.Context, dseq string) (*DeploymentInfo, error) {
	body, err := c.get(ctx, fmt.Sprintf("%s/deployments/%s", akashAPIBase, dseq))
	if err != nil {
		return nil, err
	}

	var result struct {
		DSeq     string `json:"dseq"`
		Status   string `json:"status"`
		Provider string `json:"provider,omitempty"`
		URI      string `json:"uri,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &DeploymentInfo{
		DSeq:     result.DSeq,
		Status:   result.Status,
		Provider: result.Provider,
		URI:      result.URI,
	}, nil
}

// GetCredentials retrieves Xray credentials from the deployed container.
func (c *Client) GetCredentials(ctx context.Context, leaseID string) (*Credentials, error) {
	c.logger.Info("retrieving credentials", zap.String("lease_id", leaseID))

	// Use Akash CLI to exec into container and get credentials
	cmd := exec.CommandContext(ctx, "akash",
		"provider", "lease-logs",
		"--dseq", leaseID,
		"--provider", "", // Will be set from deployment
		"--follow=false",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("getting logs: %w, output: %s", err, string(output))
	}

	// Parse credentials from logs
	return parseCredentials(string(output)), nil
}

// CloseDeployment closes the deployment.
func (c *Client) CloseDeployment(ctx context.Context, dseq string) error {
	c.logger.Info("closing deployment", zap.String("dseq", dseq))

	_, err := c.delete(ctx, fmt.Sprintf("%s/deployments/%s", akashAPIBase, dseq))
	return err
}

// Helper methods

func (c *Client) post(ctx context.Context, url string, payload interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) delete(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func parseCredentials(logs string) *Credentials {
	cred := &Credentials{}
	lines := strings.Split(logs, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "UUID: ") {
			cred.UUID = strings.TrimPrefix(line, "UUID: ")
		} else if strings.HasPrefix(line, "PUBLIC_KEY: ") {
			cred.PublicKey = strings.TrimPrefix(line, "PUBLIC_KEY: ")
		} else if strings.HasPrefix(line, "SHORT_ID: ") {
			cred.ShortID = strings.TrimPrefix(line, "SHORT_ID: ")
		} else if strings.HasPrefix(line, "HOSTNAME: ") {
			cred.Hostname = strings.TrimPrefix(line, "HOSTNAME: ")
		}
	}
	return cred
}

// Compile-time interface check
var _ rotation.Controller = (*Controller)(nil)
