package camouflage

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"time"

	"go.uber.org/zap"
)

// Supported browser fingerprints for uTLS.
// Maps user-friendly names to uTLS ClientHelloID equivalents.
var SupportedFingerprints = map[string]string{
	"chrome":      "HelloChrome_Auto",
	"chrome-120":  "HelloChrome_120",
	"chrome-115":  "HelloChrome_115",
	"firefox":     "HelloFirefox_Auto",
	"firefox-121": "HelloFirefox_121",
	"firefox-120": "HelloFirefox_120",
	"safari":      "HelloSafari_Auto",
	"edge":        "HelloEdge_Auto",
	"ios":         "HelloIOS_Auto",
	"android":     "HelloAndroid_11_OkHttp",
	"random":      "HelloRandomized",
	"randomized":  "HelloRandomizedALPN",
}

// SelectFingerprint resolves a user-friendly fingerprint name to its uTLS identifier.
// Returns the uTLS ClientHelloID string name.
func SelectFingerprint(name string) (string, error) {
	if fp, ok := SupportedFingerprints[name]; ok {
		return fp, nil
	}
	return "", fmt.Errorf("unsupported fingerprint %q, supported: %v", name, ListFingerprints())
}

// RandomFingerprint picks a random plausible browser fingerprint for moving-target defense.
func RandomFingerprint() string {
	// Weight towards common browsers for plausibility
	weighted := []string{
		"chrome", "chrome", "chrome", "chrome", // 40% Chrome
		"firefox", "firefox",                    // 20% Firefox
		"edge",                                  // 10% Edge
		"safari",                                // 10% Safari
		"chrome-120",                            // 10% specific Chrome
		"firefox-121",                           // 10% specific Firefox
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(weighted))))
	if err != nil {
		return "chrome" // Safe fallback
	}

	return weighted[n.Int64()]
}

// ListFingerprints returns all supported fingerprint names.
func ListFingerprints() []string {
	names := make([]string, 0, len(SupportedFingerprints))
	for name := range SupportedFingerprints {
		names = append(names, name)
	}
	return names
}

// PadPayload pads data to the target size with random bytes.
// If data is already >= target, it is returned unchanged.
func PadPayload(data []byte, targetSize int) []byte {
	if len(data) >= targetSize {
		return data
	}

	padding := make([]byte, targetSize-len(data))
	_, _ = rand.Read(padding)

	return append(data, padding...)
}

// NoiseInjector generates periodic noise bursts on a connection
// to defeat traffic analysis.
type NoiseInjector struct {
	interval    time.Duration
	minBytes    int
	maxBytes    int
	logger      *zap.Logger
	stopCh      chan struct{}
}

// NewNoiseInjector creates a noise injector with the given parameters.
func NewNoiseInjector(interval time.Duration, minBytes, maxBytes int, logger *zap.Logger) *NoiseInjector {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &NoiseInjector{
		interval: interval,
		minBytes: minBytes,
		maxBytes: maxBytes,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

// Start begins injecting noise on the given connection.
func (ni *NoiseInjector) Start(conn net.Conn) {
	go func() {
		ticker := time.NewTicker(ni.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ni.stopCh:
				return
			case <-ticker.C:
				size := ni.minBytes
				if ni.maxBytes > ni.minBytes {
					n, err := rand.Int(rand.Reader, big.NewInt(int64(ni.maxBytes-ni.minBytes)))
					if err == nil {
						size += int(n.Int64())
					}
				}

				noise := make([]byte, size)
				_, _ = rand.Read(noise)

				if _, err := conn.Write(noise); err != nil {
					ni.logger.Debug("noise injection ended", zap.Error(err))
					return
				}

				ni.logger.Debug("injected noise burst", zap.Int("bytes", size))
			}
		}
	}()
}

// Stop halts noise injection.
func (ni *NoiseInjector) Stop() {
	close(ni.stopCh)
}
