#!/usr/bin/env bash
# scripts/deploy-test.sh — Deploy a test EntropyTunnel stack
# Usage: ./scripts/deploy-test.sh [server-ip]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
SERVER_IP="${1:-}"
VERSION="0.1.0"

banner() { printf "\n\033[1;36m▸ %s\033[0m\n" "$1"; }
ok()     { printf "  \033[32m✓ %s\033[0m\n" "$1"; }
fail()   { printf "  \033[31m✗ %s\033[0m\n" "$1"; exit 1; }

# ── 1. Pre-flight checks ─────────────────────────────────────────────
banner "Pre-flight checks"

command -v go    >/dev/null 2>&1 || fail "Go not installed"
ok "Go $(go version | awk '{print $3}')"

command -v docker >/dev/null 2>&1 || fail "Docker not installed"
ok "Docker available"

# ── 2. Run tests ──────────────────────────────────────────────────────
banner "Running tests"
cd "$PROJECT_DIR"
go test -count=1 ./... || fail "Tests failed"
ok "All tests pass"

# ── 3. Build binaries ────────────────────────────────────────────────
banner "Building binaries"
make build
ok "entropy-server built"
ok "entropy-client built"

# ── 4. Build Docker image ────────────────────────────────────────────
banner "Building Docker image"
make docker
ok "entropy-tunnel:${VERSION} image ready"

# ── 5. Generate x25519 keys (if keygen script exists) ────────────────
banner "Generating Reality keys"
if [ -f "$PROJECT_DIR/scripts/generate-keys.sh" ]; then
    source "$PROJECT_DIR/scripts/generate-keys.sh"
else
    echo "  (Using example keys — replace in production)"
    REALITY_PRIVATE_KEY="example-private-key"
    REALITY_PUBLIC_KEY="example-public-key"
fi
ok "Keys ready"

# ── 6. Deploy via Docker Compose ─────────────────────────────────────
banner "Deploying via Docker Compose"
cd "$PROJECT_DIR"
docker compose up --build -d
ok "Stack deployed"

# ── 7. Health check ──────────────────────────────────────────────────
banner "Health check"
sleep 3
if curl -sf http://localhost:9876/api/health >/dev/null 2>&1; then
    ok "API endpoint healthy"
else
    echo "  ⚠ API not reachable yet (server may need real TLS certs)"
fi

# ── 8. First Cloudflare Worker rotation (if configured) ──────────────
banner "Cloudflare Worker rotation"
if [ -n "${CF_API_TOKEN:-}" ] && [ -n "${CF_ACCOUNT_ID:-}" ]; then
    echo "  CF_API_TOKEN and CF_ACCOUNT_ID set — first rotation would happen here"
    echo "  (Automated rotation starts with the running server)"
    ok "Cloudflare rotation configured"
else
    echo "  ⚠ CF_API_TOKEN / CF_ACCOUNT_ID not set — skipping CF rotation"
    echo "  Set them in environment to enable Cloudflare Workers rotation"
fi

# ── 9. Summary ───────────────────────────────────────────────────────
banner "Deployment summary"
echo ""
echo "  EntropyTunnel v${VERSION} deployed"
echo ""
echo "  Server:     https://${SERVER_IP:-localhost}:443"
echo "  API:        http://localhost:9876/api/health"
echo "  SOCKS5:     socks5://127.0.0.1:1080 (after client connects)"
echo ""
echo "  Connect with:"
echo "    ./bin/entropy-client connect -c configs/client.yaml"
echo ""
echo "  Or double-click the GUI app:"
echo "    cd gui && npm start"
echo ""
ok "Done ✨"
