# EntropyTunnel v0.1.0 — Release Notes

_February 17, 2026_

## Your internet, uncensored. 🛡️

Just days after [Spain's courts ordered ISPs to block ProtonVPN, Windscribe, and 30+ services](https://en.wikipedia.org/wiki/La_Liga_website_blocking) to enforce La Liga's piracy crackdown, EntropyTunnel ships its first release.

**EntropyTunnel is an anti-censorship VPN that is practically unblockable.** Unlike traditional VPNs whose IPs get blocked within hours, EntropyTunnel uses five layers of defense to make censorship economically and technically infeasible.

---

## What's in v0.1.0

### 🎭 Layer 1 — Traffic Camouflage
- **VLESS + XTLS-Reality**: Your traffic is indistinguishable from visiting google.com
- **12 browser fingerprints**: Mimics Chrome, Firefox, Safari, Edge TLS handshakes
- **Decoy traffic + noise injection**: Defeats timing-based traffic analysis

### 🔒 Layer 2 — Domain Concealment
- **ECH (Encrypted Client Hello)**: Even the SNI is hidden — ISPs can't see what you're connecting to
- **GREASE mode**: Blends in with real browser ECH extensions

### 🔄 Layer 3 — Dynamic Endpoints
- **Cloudflare Workers rotation**: New IP every 30 minutes via serverless
- **AWS Lambda@Edge fallback**: Secondary rotation through Lambda function URLs
- **Auto-healing**: Health checker rotates endpoints after 3 consecutive failures

### 🎯 Layer 4 — Moving Target Defense
- **Protocol hopping**: Automatic fallback from VLESS → Trojan → Snowflake
- **Multi-protocol registry**: Priority-based selection with fallback chain

### ☢️ Layer 5 — Nuclear Resistance
- **Snowflake P2P**: Emergency connectivity through volunteer WebRTC proxies
- Works even when ALL known IPs are blocked

### ⚽ Sports Mode
- Low-latency + extra noise injection optimized for live streaming
- Perfect for watching La Liga, Champions League, F1 without ISP throttling

### 🖥️ Desktop GUI
- Premium dark theme with glassmorphism effects
- One-click connect, live stats, status ring
- Sports Mode toggle
- Config import + QR scan

### 💰 Privacy-First Payments
- BTCPay Server integration (BTC + Monero)
- No credit cards, no KYC, no logs

---

## Downloads

| Platform | CLI | GUI |
|----------|-----|-----|
| macOS (Apple Silicon) | `entropy-client-darwin-arm64` | `EntropyTunnel-darwin.dmg` |
| macOS (Intel) | `entropy-client-darwin-amd64` | `EntropyTunnel-darwin.zip` |
| Linux (x64) | `entropy-client-linux-amd64` | `EntropyTunnel-linux.deb` |
| Linux (ARM64) | `entropy-client-linux-arm64` | — |
| Windows | `entropy-client-windows-amd64.exe` | `EntropyTunnel-win32.exe` |
| Docker | `docker pull entropy-tunnel:0.1.0` | — |

## Quick Start

```bash
# 1. Download the client for your platform
chmod +x entropy-client-*

# 2. Connect (edit values with your server details)
./entropy-client connect \
  --server your-server:443 \
  --uuid your-uuid \
  --sni www.google.com \
  --public-key your-server-pubkey \
  --short-id abcdef01

# 3. Use the SOCKS5 proxy at 127.0.0.1:1080
# Or just configure your browser's proxy settings
```

## Verification

```
62/62 tests passing
go vet: 0 errors
Platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
```

See `checksums.txt` for SHA256 verification.
