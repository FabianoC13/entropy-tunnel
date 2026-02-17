#!/bin/bash
# EntropyTunnel System-Wide VPN Toggle for macOS
# Routes all system traffic through the SOCKS5 proxy

VPN_HOST="127.0.0.1"
VPN_PORT="1080"
LOG_FILE="$HOME/.entropy-tunnel-vpn.log"

# Use full path for networksetup
NETWORKSETUP="/usr/sbin/networksetup"

# Detect primary network interface
get_primary_interface() {
    /sbin/route -n get default 2>/dev/null | /usr/bin/grep interface | awk '{print $2}'
}

# Get current service name
get_network_service() {
    local interface=$(get_primary_interface)
    if [ -z "$interface" ]; then
        echo "Wi-Fi"
        return
    fi
    # Try to get service name from interface
    $NETWORKSETUP -listallhardwareports | grep -B1 "$interface" | /usr/bin/grep "Hardware Port" | /usr/bin/cut -d: -f2 | /usr/bin/xargs
}

# Check if VPN is running
check_vpn() {
    nc -z $VPN_HOST $VPN_PORT 2>/dev/null
}

# Enable system-wide VPN
enable_vpn() {
    local service=$(get_network_service)
    
    echo "[$$(date)] Enabling EntropyTunnel system-wide VPN..." | tee -a "$LOG_FILE"
    
    # Check if VPN is running
    if ! check_vpn; then
        echo "❌ VPN is not running! Start it first with:"
        echo "   ./bin/entropy-client connect -c configs/client.yaml"
        exit 1
    fi
    
    # Save current settings
    echo "$service" > /tmp/entropy-vpn-service
    $NETWORKSETUP -getsocksfirewallproxy "$service" > /tmp/entropy-vpn-backup-socks
    $NETWORKSETUP -getwebproxy "$service" > /tmp/entropy-vpn-backup-http
    $NETWORKSETUP -getsecurewebproxy "$service" > /tmp/entropy-vpn-backup-https
    
    # Set SOCKS proxy (catches most traffic)
    sudo $NETWORKSETUP -setsocksfirewallproxy "$service" $VPN_HOST $VPN_PORT
    sudo $NETWORKSETUP -setsocksfirewallproxystate "$service" on
    
    # Set HTTP proxy (some apps use this)
    sudo $NETWORKSETUP -setwebproxy "$service" $VPN_HOST $VPN_PORT
    sudo $NETWORKSETUP -setwebproxystate "$service" on
    
    # Set HTTPS proxy
    sudo $NETWORKSETUP -setsecurewebproxy "$service" $VPN_HOST $VPN_PORT
    sudo $NETWORKSETUP -setsecurewebproxystate "$service" on
    
    # Configure DNS to prevent leaks (use Cloudflare)
    sudo $NETWORKSETUP -setdnsservers "$service" 1.1.1.1 1.0.0.1
    
    echo "✅ System-wide VPN enabled on: $service"
    echo "   SOCKS5: $VPN_HOST:$VPN_PORT"
    echo "   DNS: 1.1.1.1, 1.0.0.1"
    
    # Test connection
    sleep 1
    local ip=$(curl -s ifconfig.me 2>/dev/null)
    local location=$(curl -s ipinfo.io/country 2>/dev/null)
    
    if [ "$location" = "IE" ]; then
        echo "✅ VPN active! IP: $ip (Ireland)"
    else
        echo "⚠️  Warning: IP shows $ip ($location) - some apps may bypass proxy"
    fi
    
    echo ""
    echo "To disable: sudo ./scripts/vpn-system-toggle.sh off"
}

# Disable system-wide VPN
disable_vpn() {
    local service=$(cat /tmp/entropy-vpn-service 2>/dev/null || echo "Wi-Fi")
    
    echo "[$$(date)] Disabling EntropyTunnel system-wide VPN..." | tee -a "$LOG_FILE"
    
    # Turn off proxies
    sudo $NETWORKSETUP -setsocksfirewallproxystate "$service" off
    sudo $NETWORKSETUP -setwebproxystate "$service" off
    sudo $NETWORKSETUP -setsecurewebproxystate "$service" off
    
    # Restore DNS
    sudo $NETWORKSETUP -setdnsservers "$service" empty
    
    echo "✅ System-wide VPN disabled on: $service"
    
    # Show current IP
    sleep 1
    local ip=$(curl -s ifconfig.me 2>/dev/null)
    echo "   Current IP: $ip"
}

# Status check
show_status() {
    local service=$(get_network_service)
    
    echo "=== EntropyTunnel System VPN Status ==="
    echo "Network Service: $service"
    echo ""
    
    echo "SOCKS Proxy:"
    $NETWORKSETUP -getsocksfirewallproxy "$service"
    echo ""
    
    echo "HTTP Proxy:"
    $NETWORKSETUP -getwebproxy "$service"
    echo ""
    
    echo "DNS Servers:"
    $NETWORKSETUP -getdnsservers "$service"
    echo ""
    
    echo "Current IP:"
    curl -s ifconfig.me 2>/dev/null || echo "Unable to detect"
}

# Main
case "${1:-status}" in
    on|enable|start)
        enable_vpn
        ;;
    off|disable|stop)
        disable_vpn
        ;;
    status)
        show_status
        ;;
    *)
        echo "Usage: $0 [on|off|status]"
        echo ""
        echo "Commands:"
        echo "  on      - Enable system-wide VPN routing"
        echo "  off     - Disable system-wide VPN routing"
        echo "  status  - Show current proxy settings"
        echo ""
        echo "Prerequisites:"
        echo "  1. Start entropy-client: ./bin/entropy-client connect -c configs/client.yaml"
        echo "  2. Then enable system VPN: sudo $0 on"
        exit 1
        ;;
esac
