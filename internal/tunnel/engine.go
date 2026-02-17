package tunnel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// EngineStatus represents the current state of the tunnel engine.
type EngineStatus string

const (
	StatusStopped  EngineStatus = "stopped"
	StatusStarting EngineStatus = "starting"
	StatusRunning  EngineStatus = "running"
	StatusStopping EngineStatus = "stopping"
	StatusError    EngineStatus = "error"
)

// EngineMode distinguishes server from client engine.
type EngineMode string

const (
	ModeServer EngineMode = "server"
	ModeClient EngineMode = "client"
)

// XrayInstance abstracts the xray-core runtime so we can swap in a real
// implementation (via build-tag "xray") or keep a lightweight stub for
// unit-testing and environments where xray-core is unavailable.
type XrayInstance interface {
	Start() error
	Close() error
}

// XrayLoader converts raw JSON config bytes into an XrayInstance.
// Production code (xray_real.go) uses core.New + serial.LoadJSONConfig.
// The default stub (xray_stub.go) returns a no-op instance.
type XrayLoader func(jsonCfg []byte) (XrayInstance, error)

// defaultLoader is set by init() in the appropriate build-tag file.
var defaultLoader XrayLoader

func init() {
	if defaultLoader == nil {
		// Fallback: stub loader so the binary always compiles.
		defaultLoader = stubLoader
	}
}

// Engine wraps the Xray-core instance and manages its lifecycle.
type Engine struct {
	config       *Config
	clientConfig *ClientConfig
	mode         EngineMode
	logger       *zap.Logger
	status       EngineStatus
	mu           sync.RWMutex
	instance     XrayInstance
	stopCh       chan struct{}
	loader       XrayLoader
	jsonConfig   []byte // cached generated config
}

// NewEngine creates a new server-mode tunnel engine.
func NewEngine(cfg *Config, logger *zap.Logger) (*Engine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &Engine{
		config: cfg,
		mode:   ModeServer,
		logger: logger,
		status: StatusStopped,
		stopCh: make(chan struct{}),
		loader: defaultLoader,
	}, nil
}

// NewClientEngine creates a new client-mode tunnel engine.
func NewClientEngine(cfg *ClientConfig, logger *zap.Logger) (*Engine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("client config must not be nil")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid client config: %w", err)
	}
	return &Engine{
		clientConfig: cfg,
		mode:         ModeClient,
		logger:       logger,
		status:       StatusStopped,
		stopCh:       make(chan struct{}),
		loader:       defaultLoader,
	}, nil
}

// Start boots the tunnel engine by building and loading the xray-core config.
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status == StatusRunning {
		return fmt.Errorf("engine is already running")
	}

	e.status = StatusStarting

	// Build JSON config for xray-core
	var (
		jsonCfg []byte
		err     error
	)
	switch e.mode {
	case ModeServer:
		e.logger.Info("building server config",
			zap.String("listen", e.config.Listen),
			zap.String("protocol", e.config.Protocol),
			zap.String("sni", e.config.Reality.SNI),
		)
		jsonCfg, err = BuildServerJSON(e.config)
	case ModeClient:
		e.logger.Info("building client config",
			zap.String("server", e.clientConfig.Server),
			zap.String("sni", e.clientConfig.SNI),
			zap.String("fingerprint", e.clientConfig.Fingerprint),
		)
		jsonCfg, err = BuildClientJSON(e.clientConfig)
	default:
		e.status = StatusError
		return fmt.Errorf("unknown engine mode: %s", e.mode)
	}
	if err != nil {
		e.status = StatusError
		return fmt.Errorf("failed to build xray config: %w", err)
	}
	e.jsonConfig = jsonCfg

	e.logger.Debug("xray-core JSON config generated",
		zap.Int("bytes", len(jsonCfg)),
	)

	// Load into xray-core
	instance, err := e.loader(jsonCfg)
	if err != nil {
		e.status = StatusError
		return fmt.Errorf("failed to load xray config: %w", err)
	}

	// Start xray-core instance
	if err := instance.Start(); err != nil {
		e.status = StatusError
		return fmt.Errorf("failed to start xray instance: %w", err)
	}

	e.instance = instance
	e.status = StatusRunning
	e.logger.Info("entropy tunnel engine is running", zap.String("mode", string(e.mode)))
	return nil
}

// Stop gracefully shuts down the tunnel engine.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status != StatusRunning {
		return fmt.Errorf("engine is not running (status: %s)", e.status)
	}

	e.status = StatusStopping
	e.logger.Info("stopping entropy tunnel engine")

	close(e.stopCh)

	if e.instance != nil {
		if err := e.instance.Close(); err != nil {
			e.logger.Error("error closing xray instance", zap.Error(err))
		}
	}

	e.instance = nil
	e.status = StatusStopped
	e.stopCh = make(chan struct{})
	e.logger.Info("tunnel engine stopped")
	return nil
}

// Status returns the current engine status.
func (e *Engine) Status() EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

// StopCh returns a channel that is closed when the engine is stopping.
func (e *Engine) StopCh() <-chan struct{} {
	return e.stopCh
}

// JSONConfig returns the generated xray-core JSON config (for debugging).
func (e *Engine) JSONConfig() json.RawMessage {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.jsonConfig
}

// ConfigPretty returns indented JSON config for display.
func (e *Engine) ConfigPretty() string {
	raw := e.JSONConfig()
	if raw == nil {
		return "{}"
	}
	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return string(raw)
	}
	return out.String()
}

// --- Stub loader (always available) ---

type stubInstance struct{}

func (s *stubInstance) Start() error { return nil }
func (s *stubInstance) Close() error { return nil }

func stubLoader(jsonCfg []byte) (XrayInstance, error) {
	// Validate that the JSON is well-formed
	var raw json.RawMessage
	if err := json.Unmarshal(jsonCfg, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON config: %w", err)
	}
	return &stubInstance{}, nil
}
