#!/bin/bash
# Disable system-wide VPN

SERVICE="Wi-Fi"

echo "🔴 Disabling EntropyTunnel system VPN..."

sudo /usr/sbin/networksetup -setsocksfirewallproxystate "$SERVICE" off
sudo /usr/sbin/networksetup -setwebproxystate "$SERVICE" off
sudo /usr/sbin/networksetup -setsecurewebproxystate "$SERVICE" off
sudo /usr/sbin/networksetup -setdnsservers "$SERVICE" empty

echo "✅ System VPN disabled"
echo ""
echo "Your current IP:"
curl -s ifconfig.me
echo ""
