package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/fabiano/entropy-tunnel/internal/api"
	"github.com/fabiano/entropy-tunnel/internal/camouflage"
	"github.com/fabiano/entropy-tunnel/internal/tunnel"
)

var (
	version   = "0.1.0"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "entropy-client",
		Short: "EntropyTunnel Client — Connect to anti-censorship tunnel",
		Long: `EntropyTunnel Client establishes a VLESS/XTLS-Reality connection to
an EntropyTunnel server, providing a local SOCKS5/HTTP proxy for bypassing
censorship and ISP blocks. Includes a local API for GUI integration.`,
	}

	var configPath string
	var server, uuid, sni, fingerprint, publicKey, shortID, localListen, apiListen string
	var sportsMode bool

	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to an EntropyTunnel server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClient(configPath, server, uuid, sni, fingerprint, publicKey, shortID, localListen, apiListen, sportsMode)
		},
	}
	connectCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to client config file")
	connectCmd.Flags().StringVar(&server, "server", "", "Server address (host:port)")
	connectCmd.Flags().StringVar(&uuid, "uuid", "", "VLESS UUID")
	connectCmd.Flags().StringVar(&sni, "sni", "", "SNI for Reality")
	connectCmd.Flags().StringVar(&fingerprint, "fingerprint", "chrome", "uTLS fingerprint")
	connectCmd.Flags().StringVar(&publicKey, "public-key", "", "Server Reality public key")
	connectCmd.Flags().StringVar(&shortID, "short-id", "", "Reality short ID")
	connectCmd.Flags().StringVar(&localListen, "local", "127.0.0.1:1080", "Local SOCKS5 listen address")
	connectCmd.Flags().StringVar(&apiListen, "api", "127.0.0.1:9876", "Local API address for GUI")
	connectCmd.Flags().BoolVar(&sportsMode, "sports-mode", false, "Enable low-latency sports streaming mode")

	listFPCmd := &cobra.Command{
		Use:   "fingerprints",
		Short: "List supported uTLS fingerprints",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Supported browser fingerprints:")
			for _, fp := range camouflage.ListFingerprints() {
				utlsID, _ := camouflage.SelectFingerprint(fp)
				fmt.Printf("  %-15s → %s\n", fp, utlsID)
			}
		},
	}

	showConfigCmd := &cobra.Command{
		Use:   "show-config",
		Short: "Show the generated xray-core client JSON config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return fmt.Errorf("--config is required")
			}
			cfg, err := tunnel.LoadClientConfig(configPath)
			if err != nil {
				return err
			}
			jsonBytes, err := tunnel.BuildClientJSON(cfg)
			if err != nil {
				return err
			}
			fmt.Println(string(jsonBytes))
			return nil
		},
	}
	showConfigCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to client config file")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("entropy-client %s (commit: %s, built: %s)\n", version, commit, buildDate)
		},
	}

	rootCmd.AddCommand(connectCmd, listFPCmd, showConfigCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runClient(configPath, server, uuid, sni, fingerprint, publicKey, shortID, localListen, apiListen string, sportsMode bool) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer logger.Sync()

	var cfg *tunnel.ClientConfig

	if configPath != "" {
		cfg, err = tunnel.LoadClientConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		cfg = &tunnel.ClientConfig{
			Server:      server,
			UUID:        uuid,
			SNI:         sni,
			Fingerprint: fingerprint,
			PublicKey:   publicKey,
			ShortID:     shortID,
			LocalListen: localListen,
			APIListen:   apiListen,
			SportsMode:  sportsMode,
		}
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	fpID, err := camouflage.SelectFingerprint(cfg.Fingerprint)
	if err != nil {
		return err
	}

	logger.Info("connecting to entropy tunnel",
		zap.String("server", cfg.Server),
		zap.String("sni", cfg.SNI),
		zap.String("fingerprint", fpID),
		zap.String("local_socks5", cfg.LocalListen),
		zap.Bool("sports_mode", cfg.SportsMode),
	)

	// Create client engine
	engine, err := tunnel.NewClientEngine(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create client engine: %w", err)
	}

	// Start tunnel
	if err := engine.Start(); err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	logger.Info("tunnel connected",
		zap.String("socks5", cfg.LocalListen),
	)

	// Start local API server for GUI
	apiSrv := api.NewServer(cfg.APIListen, engine, logger)
	if err := apiSrv.Start(); err != nil {
		logger.Warn("failed to start API server", zap.Error(err))
	} else {
		logger.Info("API server running", zap.String("addr", cfg.APIListen))
	}

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Info("received signal, disconnecting", zap.String("signal", sig.String()))

	_ = apiSrv.Stop()
	_ = engine.Stop()
	return nil
}
