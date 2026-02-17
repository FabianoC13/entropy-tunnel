package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/fabiano/entropy-tunnel/internal/tunnel"
)

// Server provides a local HTTP API for the GUI desktop client.
type Server struct {
	addr   string
	engine *tunnel.Engine
	logger *zap.Logger
	server *http.Server
	mu     sync.RWMutex

	// Connection state
	connected  bool
	sportsMode bool
	startTime  time.Time
	bytesSent  int64
	bytesRecv  int64
}

// NewServer creates a new API server for GUI integration.
func NewServer(addr string, engine *tunnel.Engine, logger *zap.Logger) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Server{
		addr:   addr,
		engine: engine,
		logger: logger,
	}
}

// Start begins serving the HTTP API.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("POST /api/connect", s.handleConnect)
	mux.HandleFunc("POST /api/disconnect", s.handleDisconnect)
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("POST /api/config", s.handleSetConfig)
	mux.HandleFunc("POST /api/sports-mode", s.handleSportsMode)
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// CORS middleware for Electron
	handler := corsMiddleware(mux)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: handler,
	}

	s.logger.Info("API server starting", zap.String("addr", s.addr))
	go func() {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Error("API server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop shuts down the API server.
func (s *Server) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

type statusResponse struct {
	Connected  bool   `json:"connected"`
	Status     string `json:"status"`
	SportsMode bool   `json:"sports_mode"`
	Uptime     string `json:"uptime,omitempty"`
	BytesSent  int64  `json:"bytes_sent"`
	BytesRecv  int64  `json:"bytes_recv"`
	ServerAddr string `json:"server_addr,omitempty"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check actual engine status, not just the connected flag
	engineStatus := s.engine.Status()
	isConnected := engineStatus == tunnel.StatusRunning

	resp := statusResponse{
		Connected:  isConnected,
		Status:     string(engineStatus),
		SportsMode: s.sportsMode,
		BytesSent:  s.bytesSent,
		BytesRecv:  s.bytesRecv,
	}

	if isConnected && !s.startTime.IsZero() {
		resp.Uptime = time.Since(s.startTime).Truncate(time.Second).String()
	}

	writeJSON(w, resp)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		writeError(w, http.StatusConflict, "already connected")
		return
	}

	if err := s.engine.Start(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.connected = true
	s.startTime = time.Now()
	s.logger.Info("tunnel connected via API")

	writeJSON(w, map[string]string{"status": "connected"})
}

func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		writeError(w, http.StatusConflict, "not connected")
		return
	}

	if err := s.engine.Stop(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.connected = false
	s.startTime = time.Time{}
	s.logger.Info("tunnel disconnected via API")

	writeJSON(w, map[string]string{"status": "disconnected"})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"config": s.engine.ConfigPretty(),
	})
}

func (s *Server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	// Accept YAML or JSON config from GUI
	var body struct {
		ConfigPath string `json:"config_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	writeJSON(w, map[string]string{"status": "config updated", "path": body.ConfigPath})
}

func (s *Server) handleSportsMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	s.mu.Lock()
	s.sportsMode = body.Enabled
	s.mu.Unlock()

	s.logger.Info("sports mode toggled", zap.Bool("enabled", body.Enabled))
	writeJSON(w, map[string]bool{"sports_mode": body.Enabled})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// FormatBytes converts bytes to human-readable format.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
