#!/bin/bash
# Quick system VPN toggle - run this in Terminal
# This routes all Mac traffic through EntropyTunnel

SERVICE="Wi-Fi"
PROXY_HOST="127.0.0.1"
PROXY_PORT="1080"

echo "🔐 EntropyTunnel System-Wide VPN Setup"
echo "======================================"
echo ""

# Check if VPN is running
if ! nc -z $PROXY_HOST $PROXY_PORT 2>/dev/null; then
    echo "❌ VPN is not running!"
    echo ""
    echo "Start it first:"
    echo "  cd /Users/fabiano/Documents/vpn"
    echo "  ./bin/entropy-client connect -c configs/client.yaml"
    exit 1
fi

echo "✅ VPN detected on port 1080"
echo ""
echo "Current IP (direct):"
curl -s ifconfig.me
echo ""
echo ""

# Enable proxies
echo "🌐 Enabling system-wide proxy..."

sudo /usr/sbin/networksetup -setsocksfirewallproxy "$SERVICE" $PROXY_HOST $PROXY_PORT
sudo /usr/sbin/networksetup -setsocksfirewallproxystate "$SERVICE" on

sudo /usr/sbin/networksetup -setwebproxy "$SERVICE" $PROXY_HOST $PROXY_PORT
sudo /usr/sbin/networksetup -setwebproxystate "$SERVICE" on

sudo /usr/sbin/networksetup -setsecurewebproxy "$SERVICE" $PROXY_HOST $PROXY_PORT
sudo /usr/sbin/networksetup -setsecurewebproxystate "$SERVICE" on

# Set DNS (optional but recommended for privacy)
sudo /usr/sbin/networksetup -setdnsservers "$SERVICE" 1.1.1.1 1.0.0.1

echo ""
echo "✅ System VPN enabled!"
echo ""
echo "Testing..."
sleep 1

NEW_IP=$(curl -s ifconfig.me)
LOCATION=$(curl -s ipinfo.io/country)

echo "Your IP: $NEW_IP ($LOCATION)"

if [ "$LOCATION" = "IE" ]; then
    echo "🎉 VPN is working! You're in Ireland!"
else
    echo "⚠️  Location: $LOCATION"
fi

echo ""
echo "To disable, run: sudo /usr/sbin/networksetup -setsocksfirewallproxystate Wi-Fi off"
