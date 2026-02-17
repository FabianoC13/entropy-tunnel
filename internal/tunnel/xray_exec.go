//go:build exec

package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	defaultLoader = execLoader
}

// execLoader shells out to the system xray binary.
func execLoader(jsonCfg []byte) (XrayInstance, error) {
	// Write config to temp file
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, "entropy-xray-config.json")
	
	if err := os.WriteFile(configPath, jsonCfg, 0600); err != nil {
		return nil, fmt.Errorf("failed to write temp config: %w", err)
	}
	
	return &execXrayInstance{configPath: configPath}, nil
}

type execXrayInstance struct {
	configPath string
	cmd        *exec.Cmd
}

func (e *execXrayInstance) Start() error {
	// Find xray binary
	xrayPath, err := exec.LookPath("xray")
	if err != nil {
		// Try common locations
		candidates := []string{
			"/opt/homebrew/bin/xray",
			"/usr/local/bin/xray",
			"/usr/bin/xray",
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				xrayPath = candidate
				break
			}
		}
		if xrayPath == "" {
			return fmt.Errorf("xray binary not found in PATH or common locations")
		}
	}
	
	e.cmd = exec.Command(xrayPath, "run", "-config", e.configPath)
	e.cmd.Stdout = os.Stdout
	e.cmd.Stderr = os.Stderr
	
	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start xray: %w", err)
	}
	
	return nil
}

func (e *execXrayInstance) Close() error {
	if e.cmd != nil && e.cmd.Process != nil {
		// Try graceful shutdown first
		e.cmd.Process.Signal(os.Interrupt)
		// Give it a moment to cleanup
		// In production, you'd want proper process management
	}
	// Clean up temp config
	os.Remove(e.configPath)
	return nil
}
