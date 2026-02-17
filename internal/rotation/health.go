package rotation

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthChecker monitors endpoint health and triggers rotation on failure.
type HealthChecker struct {
	controller Controller
	interval   time.Duration
	timeout    time.Duration
	logger     *zap.Logger
	mu         sync.RWMutex
	results    map[string]*HealthResult
	stopCh     chan struct{}
	client     *http.Client
}

// HealthResult holds the health status of an endpoint.
type HealthResult struct {
	EndpointID  string        `json:"endpoint_id"`
	Healthy     bool          `json:"healthy"`
	Latency     time.Duration `json:"latency"`
	LastCheck   time.Time     `json:"last_check"`
	FailCount   int           `json:"fail_count"`
	Error       string        `json:"error,omitempty"`
}

// NewHealthChecker creates a health checker for the given controller.
func NewHealthChecker(ctrl Controller, interval, timeout time.Duration, logger *zap.Logger) *HealthChecker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HealthChecker{
		controller: ctrl,
		interval:   interval,
		timeout:    timeout,
		logger:     logger,
		results:    make(map[string]*HealthResult),
		stopCh:     make(chan struct{}),
		client:     &http.Client{Timeout: timeout},
	}
}

// Start begins periodic health checking.
func (hc *HealthChecker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-hc.stopCh:
				return
			case <-ticker.C:
				hc.checkAll(ctx)
			}
		}
	}()

	hc.logger.Info("health checker started",
		zap.Duration("interval", hc.interval),
		zap.Duration("timeout", hc.timeout),
	)
}

// Stop halts health checking.
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.stopCh = make(chan struct{})
}

// Results returns current health check results.
func (hc *HealthChecker) Results() map[string]*HealthResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	cp := make(map[string]*HealthResult, len(hc.results))
	for k, v := range hc.results {
		cp[k] = v
	}
	return cp
}

// checkAll probes all active endpoints.
func (hc *HealthChecker) checkAll(ctx context.Context) {
	endpoints := hc.controller.ActiveEndpoints()

	for _, ep := range endpoints {
		result := hc.checkEndpoint(ctx, ep)

		hc.mu.Lock()
		hc.results[ep.ID] = result
		hc.mu.Unlock()

		if !result.Healthy {
			hc.logger.Warn("endpoint unhealthy",
				zap.String("id", ep.ID),
				zap.Int("fail_count", result.FailCount),
				zap.String("error", result.Error),
			)

			// Auto-rotate after 3 consecutive failures
			if result.FailCount >= 3 {
				hc.logger.Info("triggering rotation due to unhealthy endpoint",
					zap.String("id", ep.ID),
				)
				go func(ep *Endpoint) {
					_ = hc.controller.Retire(ctx, ep)
					_, _ = hc.controller.Rotate(ctx)
				}(ep)
			}
		}
	}
}

// checkEndpoint probes a single endpoint.
func (hc *HealthChecker) checkEndpoint(ctx context.Context, ep *Endpoint) *HealthResult {
	result := &HealthResult{
		EndpointID: ep.ID,
		LastCheck:  time.Now(),
	}

	// Get previous result for fail count tracking
	hc.mu.RLock()
	prev, exists := hc.results[ep.ID]
	hc.mu.RUnlock()
	if exists {
		result.FailCount = prev.FailCount
	}

	start := time.Now()

	// Probe: TCP connection + optional HTTP check
	switch ep.Provider {
	case "cloudflare":
		err := hc.probeHTTPS(ctx, ep.Address)
		result.Latency = time.Since(start)
		if err != nil {
			result.Healthy = false
			result.FailCount++
			result.Error = err.Error()
		} else {
			result.Healthy = true
			result.FailCount = 0
		}

	case "aws":
		err := hc.probeHTTPS(ctx, ep.Address)
		result.Latency = time.Since(start)
		if err != nil {
			result.Healthy = false
			result.FailCount++
			result.Error = err.Error()
		} else {
			result.Healthy = true
			result.FailCount = 0
		}

	default:
		err := hc.probeTCP(ctx, ep.Address)
		result.Latency = time.Since(start)
		if err != nil {
			result.Healthy = false
			result.FailCount++
			result.Error = err.Error()
		} else {
			result.Healthy = true
			result.FailCount = 0
		}
	}

	return result
}

func (hc *HealthChecker) probeTCP(ctx context.Context, addr string) error {
	dialer := &net.Dialer{Timeout: hc.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("TCP probe failed: %w", err)
	}
	conn.Close()
	return nil
}

func (hc *HealthChecker) probeHTTPS(ctx context.Context, domain string) error {
	url := fmt.Sprintf("https://%s/", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTPS probe failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("HTTPS probe returned %d", resp.StatusCode)
	}

	return nil
}
