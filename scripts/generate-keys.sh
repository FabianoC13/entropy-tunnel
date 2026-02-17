#!/usr/bin/env bash
# generate-keys.sh — Generate X25519 key pair and UUID for EntropyTunnel
set -euo pipefail

echo "========================================"
echo " EntropyTunnel Key Generator"
echo "========================================"
echo ""

# Generate UUID
UUID=$(uuidgen | tr '[:upper:]' '[:lower:]')
echo "📋 VLESS UUID:"
echo "   $UUID"
echo ""

# Generate X25519 key pair using openssl
PRIVATE_KEY_RAW=$(openssl genpkey -algorithm X25519 2>/dev/null)
PRIVATE_KEY_HEX=$(echo "$PRIVATE_KEY_RAW" | openssl pkey -outform DER 2>/dev/null | tail -c 32 | xxd -p -c 32)
PUBLIC_KEY_HEX=$(echo "$PRIVATE_KEY_RAW" | openssl pkey -pubout -outform DER 2>/dev/null | tail -c 32 | xxd -p -c 32)

# Base64url encode for Xray compatibility
PRIVATE_KEY_B64=$(echo -n "$PRIVATE_KEY_HEX" | xxd -r -p | base64 | tr '+/' '-_' | tr -d '=')
PUBLIC_KEY_B64=$(echo -n "$PUBLIC_KEY_HEX" | xxd -r -p | base64 | tr '+/' '-_' | tr -d '=')

echo "🔑 X25519 Key Pair (Reality):"
echo "   Private Key: $PRIVATE_KEY_B64"
echo "   Public Key:  $PUBLIC_KEY_B64"
echo ""

# Generate short IDs
SHORT_ID1=$(openssl rand -hex 4)
SHORT_ID2=$(openssl rand -hex 4)
echo "🆔 Short IDs:"
echo "   $SHORT_ID1"
echo "   $SHORT_ID2"
echo ""

echo "========================================"
echo " Server Config Snippet"
echo "========================================"
cat << EOF

uuid: "$UUID"
reality:
  sni: "www.google.com"
  private_key: "$PRIVATE_KEY_B64"
  public_key: "$PUBLIC_KEY_B64"
  short_ids:
    - "$SHORT_ID1"
    - "$SHORT_ID2"

EOF

echo "========================================"
echo " Client Config Snippet"
echo "========================================"
cat << EOF

uuid: "$UUID"
sni: "www.google.com"
public_key: "$PUBLIC_KEY_B64"
short_id: "$SHORT_ID1"

EOF
