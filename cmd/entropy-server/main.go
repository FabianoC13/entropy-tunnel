package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/fabiano/entropy-tunnel/internal/tunnel"
)

var (
	version   = "0.1.0"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "entropy-server",
		Short: "EntropyTunnel Server — Anti-censorship tunnel with traffic camouflage",
		Long: `EntropyTunnel Server provides a VLESS/XTLS-Reality tunnel with uTLS
fingerprinting for traffic camouflage. It supports protocol fallbacks
(Trojan, Hysteria2) and dynamic endpoint rotation.`,
	}

	var configPath string
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the tunnel server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(configPath)
		},
	}
	serveCmd.Flags().StringVarP(&configPath, "config", "c", "configs/server-example.yaml", "Path to server config file")

	genConfigCmd := &cobra.Command{
		Use:   "generate-config",
		Short: "Generate an example server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateExampleConfig()
		},
	}

	showConfigCmd := &cobra.Command{
		Use:   "show-config",
		Short: "Show the generated xray-core JSON config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showConfig(configPath)
		},
	}
	showConfigCmd.Flags().StringVarP(&configPath, "config", "c", "configs/server-example.yaml", "Path to server config file")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("entropy-server %s (commit: %s, built: %s)\n", version, commit, buildDate)
		},
	}

	rootCmd.AddCommand(serveCmd, genConfigCmd, showConfigCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer(configPath string) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("loading configuration", zap.String("path", configPath))

	cfg, err := tunnel.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	engine, err := tunnel.NewEngine(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	if err := engine.Start(); err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}

	logger.Info("entropy tunnel server is running",
		zap.String("listen", cfg.Listen),
		zap.String("protocol", cfg.Protocol),
		zap.String("sni", cfg.Reality.SNI),
		zap.String("fingerprint", cfg.Fingerprint),
		zap.String("version", version),
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Info("received signal, shutting down", zap.String("signal", sig.String()))

	if err := engine.Stop(); err != nil {
		logger.Error("error stopping engine", zap.Error(err))
	}

	logger.Info("server shutdown complete")
	return nil
}

func showConfig(configPath string) error {
	cfg, err := tunnel.LoadConfig(configPath)
	if err != nil {
		return err
	}
	jsonBytes, err := tunnel.BuildServerJSON(cfg)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonBytes))
	return nil
}

func generateExampleConfig() error {
	example := `# EntropyTunnel Server Configuration
listen: ":443"
protocol: vless
uuid: "your-uuid-here"

reality:
  sni: "www.google.com"
  private_key: "your-x25519-private-key"
  public_key: "your-x25519-public-key"
  short_ids:
    - "abcdef01"

fingerprint: "chrome"

fallbacks:
  - protocol: trojan
    listen: ":8443"
    transport: ws
    path: "/ws"

log_level: "info"

# Rotation (optional)
rotation:
  enabled: false
  provider: "cloudflare"
  interval: "30m"

# Payment (optional)
payment:
  enabled: false
  btcpay_url: "https://your-btcpay-server.com"
`
	fmt.Print(example)
	return nil
}
