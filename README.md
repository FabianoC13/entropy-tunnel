# EntropyTunnel

> Anti-censorship VPN using traffic camouflage, dynamic endpoint rotation, and economic asymmetry for practical unblockability.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Five-Layer Defense                         │
├─────────────┬─────────────┬────────────┬──────────┬─────────────┤
│  Layer 1    │  Layer 2    │  Layer 3   │ Layer 4  │  Layer 5    │
│  Traffic    │  Domain     │  Dynamic   │ Moving   │  Nuclear    │
│  Camouflage │  Concealment│  Endpoints │ Target   │  Resistance │
│             │             │            │ Defense  │             │
│ VLESS+uTLS  │  ECH / SNI  │ Serverless │ Protocol │ Snowflake   │
│ XTLS-Reality│  Hiding     │ Rotation   │ Hopping  │ P2P / DNS   │
│ Browser FP  │  Blind TLS  │ CF Workers │ Noise    │ Tunneling   │
└─────────────┴─────────────┴────────────┴──────────┴─────────────┘
```

## Features

- **VLESS + XTLS-Reality** — indistinguishable from normal HTTPS
- **uTLS fingerprinting** — mimics Chrome/Firefox/Safari/Edge TLS handshakes
- **ECH (Encrypted Client Hello)** — hides SNI from ISP inspection
- **Protocol fallbacks** — Trojan-GO (WebSocket), automatic fallback chain
- **Dynamic endpoint rotation** — sub-hourly IP churn via Cloudflare Workers & AWS Lambda
- **Decoy traffic** — padding + noise injection defeat traffic analysis
- **Snowflake P2P fallback** — emergency connectivity through volunteer proxies
- **Sports Mode** — low-latency + extra noise for streaming
- **Desktop GUI** — Electron app with premium dark theme
- **Crypto payments** — BTCPay Server integration for privacy

## Quick Start

> **TL;DR — one command to connect:**
> ```bash
> ./entropy-client connect -c client.yaml
> ```
> Edit `configs/client-example.yaml` with your server details, rename it to `client.yaml`, and run the command above. Or just open the GUI app.

### Install

```bash
# Clone
git clone https://github.com/fabiano/entropy-tunnel.git
cd entropy-tunnel

# Build server + client
make build

# Or with real xray-core integration
make server-xray client-xray
```

### Run Server

```bash
# Generate example config
./bin/entropy-server generate-config > configs/server.yaml

# Edit with your keys (generate x25519 keypair externally)
vim configs/server.yaml

# Start server
./bin/entropy-server serve -c configs/server.yaml
```

### Run Client

```bash
# Basic connection
./bin/entropy-client connect \
  --server your-server:443 \
  --uuid your-uuid \
  --sni www.google.com \
  --public-key your-server-pubkey \
  --short-id abcdef01

# With Sports Mode
./bin/entropy-client connect -c configs/client.yaml --sports-mode

# List supported fingerprints
./bin/entropy-client fingerprints
```

### Docker

```bash
make docker
docker run -d -p 443:443 -v ./configs:/app/configs entropy-tunnel:latest
```

### Desktop GUI

```bash
make gui-dev   # Development mode
make gui       # Package for distribution
```

## Build

```bash
make build          # Build server + client
make test           # Run all 62 tests
make test-cover     # Tests with coverage report
make lint           # Run linters
make release        # Cross-compile for all platforms
make checksums      # Generate SHA256 checksums
make docker         # Build Docker image
make gui            # Build desktop GUI (macOS .dmg, Windows .exe, Linux .deb)
make gui-release    # Build CLI + GUI together
make generate-keys  # Generate Reality x25519 keys + UUID
make deploy-test    # Full test deployment (Docker + health check)
make version        # Show version info
make help           # Show all targets
```

## Project Structure

```
cmd/
  entropy-server/       # Server CLI (serve, generate-config, show-config)
  entropy-client/       # Client CLI (connect, fingerprints, show-config)
internal/
  tunnel/               # Core engine: Xray-core wrapper, config builder, loader
  camouflage/           # uTLS fingerprints, ECH, noise injection, padding
  protocols/            # Protocol registry (VLESS, Trojan, Snowflake)
  rotation/             # Endpoint rotation (NoOp, Cloudflare, AWS, health checks)
  payment/              # BTCPay Server crypto payments
  api/                  # Local HTTP API server for GUI
gui/                    # Electron desktop client
configs/                # Example YAML configurations
scripts/                # Key generation & utilities
.github/workflows/      # CI/CD: test → build → release
```

## Roadmap

- [x] Phase 1: Project scaffolding & CI/CD
- [x] Phase 2: Core tunnel (VLESS/XTLS-Reality + uTLS)
- [x] Phase 3: Dynamic endpoint rotation (CF Workers / AWS Lambda)
- [x] Phase 4: Advanced features (ECH, Snowflake P2P, GUI, payments)
- [x] Phase 5: Testing (62 tests) & release infrastructure
- [ ] Phase 6: Community beta & audit

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Core protocol | VLESS + XTLS-Reality |
| Camouflage | uTLS (Chrome/Firefox/Safari/Edge fingerprints) |
| SNI hiding | ECH (GREASE + Full mode) |
| Fallbacks | Trojan-GO, Snowflake P2P |
| Rotation | Cloudflare Workers, AWS Lambda |
| Payments | BTCPay Server (BTC/Monero) |
| Language | Go 1.22+ |
| GUI | Electron |
| Deployment | Docker, GitHub Actions, multi-platform |

## Crypto Payment

Support the project and get access:

- **BTC**: `bc1q...` (via BTCPay Server)
- **Monero**: `4...` (via BTCPay Server)

Plans: Monthly ($9.99) / Yearly ($79.99)

## Disclaimer

EntropyTunnel is intended for legitimate privacy and anti-censorship purposes. Users are responsible for complying with local laws.
