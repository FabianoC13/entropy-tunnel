#!/bin/bash
# Akash Network Deployment Script for EntropyTunnel
# Usage: ./deploy-akash.sh [create|status|close|logs] [DSEQ]

set -e

# Configuration
AKASH_API_KEY="${AKASH_API_KEY:-ac.sk.production.f27d65a063759be56d5ff429868ce99f1dc06f20e2cc520a76bd2b54d9aa1f08}"
SDL_FILE="${SDL_FILE:-deployments/akash/xray-server.yaml}"
AKASH_API="https://api.cloudmos.io/v1"
DEPLOYMENT_STATE_FILE=".akash-deployment.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check dependencies
check_deps() {
    local deps=("curl" "jq")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "$dep is required but not installed"
            exit 1
        fi
    done
}

# API helper functions
api_get() {
    local endpoint="$1"
    curl -s -H "Authorization: Bearer $AKASH_API_KEY" \
         "${AKASH_API}${endpoint}"
}

api_post() {
    local endpoint="$1"
    local data="$2"
    curl -s -X POST \
         -H "Content-Type: application/json" \
         -H "Authorization: Bearer $AKASH_API_KEY" \
         -d "$data" \
         "${AKASH_API}${endpoint}"
}

api_delete() {
    local endpoint="$1"
    curl -s -X DELETE \
         -H "Authorization: Bearer $AKASH_API_KEY" \
         "${AKASH_API}${endpoint}"
}

# Create new deployment
cmd_create() {
    log_info "Creating Akash deployment..."
    
    if [[ ! -f "$SDL_FILE" ]]; then
        log_error "SDL file not found: $SDL_FILE"
        exit 1
    fi
    
    # Read and escape SDL
    local sdl_content
    sdl_content=$(cat "$SDL_FILE")
    
    # Create deployment
    local response
    response=$(api_post "/deployments" "{\"sdl\": $(echo "$sdl_content" | jq -Rs .)}")
    
    local dseq
    dseq=$(echo "$response" | jq -r '.dseq // empty')
    
    if [[ -z "$dseq" ]]; then
        log_error "Failed to create deployment"
        log_error "Response: $response"
        exit 1
    fi
    
    log_success "Deployment created with DSEQ: $dseq"
    
    # Save deployment info
    echo "{\"dseq\": \"$dseq\", \"created_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" > "$DEPLOYMENT_STATE_FILE"
    
    log_info "Waiting for lease (this may take 2-5 minutes)..."
    
    # Wait for lease
    local attempts=0
    local max_attempts=30
    local provider=""
    local uri=""
    
    while [[ $attempts -lt $max_attempts ]]; do
        sleep 10
        attempts=$((attempts + 1))
        
        local status_response
        status_response=$(api_get "/deployments/$dseq")
        
        provider=$(echo "$status_response" | jq -r '.provider // empty')
        uri=$(echo "$status_response" | jq -r '.uri // empty')
        
        if [[ -n "$provider" && -n "$uri" ]]; then
            log_success "Lease acquired!"
            log_info "Provider: $provider"
            log_info "URI: $uri"
            
            # Update state file
            echo "$status_response" | jq ". + {dseq: \"$dseq\"}" > "$DEPLOYMENT_STATE_FILE"
            
            # Show connection info
            show_connection_info "$uri"
            return 0
        fi
        
        log_info "Waiting... ($attempts/$max_attempts)"
    done
    
    log_warn "Timeout waiting for lease. Check status manually."
    log_info "DSEQ: $dseq"
    return 1
}

# Show connection info for client config
show_connection_info() {
    local uri="$1"
    
    echo ""
    echo "═══════════════════════════════════════════════════════════"
    echo "          AKASH DEPLOYMENT READY"
    echo "═══════════════════════════════════════════════════════════"
    echo ""
    echo "Server Address: $uri:443"
    echo ""
    echo "Update your client.yaml with:"
    echo "  server: \"$uri:443\""
    echo ""
    echo "Note: UUID and keys are auto-generated in the container."
    echo "Retrieve them with: ./deploy-akash.sh logs $DSEQ"
    echo ""
    echo "═══════════════════════════════════════════════════════════"
}

# Get deployment status
cmd_status() {
    local dseq="${1:-}"
    
    if [[ -z "$dseq" && -f "$DEPLOYMENT_STATE_FILE" ]]; then
        dseq=$(jq -r '.dseq // empty' "$DEPLOYMENT_STATE_FILE")
    fi
    
    if [[ -z "$dseq" ]]; then
        log_error "No DSEQ provided and no saved deployment found"
        exit 1
    fi
    
    log_info "Checking deployment $dseq..."
    
    local response
    response=$(api_get "/deployments/$dseq")
    
    echo "$response" | jq .
}

# Close deployment
cmd_close() {
    local dseq="${1:-}"
    
    if [[ -z "$dseq" && -f "$DEPLOYMENT_STATE_FILE" ]]; then
        dseq=$(jq -r '.dseq // empty' "$DEPLOYMENT_STATE_FILE")
    fi
    
    if [[ -z "$dseq" ]]; then
        log_error "No DSEQ provided and no saved deployment found"
        exit 1
    fi
    
    log_info "Closing deployment $dseq..."
    
    local response
    response=$(api_delete "/deployments/$dseq")
    
    if [[ $? -eq 0 ]]; then
        log_success "Deployment closed"
        rm -f "$DEPLOYMENT_STATE_FILE"
    else
        log_error "Failed to close deployment"
        log_error "Response: $response"
    fi
}

# Get logs (placeholder - requires akash CLI)
cmd_logs() {
    local dseq="${1:-}"
    
    if [[ -z "$dseq" && -f "$DEPLOYMENT_STATE_FILE" ]]; then
        dseq=$(jq -r '.dseq // empty' "$DEPLOYMENT_STATE_FILE")
    fi
    
    if [[ -z "$dseq" ]]; then
        log_error "No DSEQ provided"
        exit 1
    fi
    
    log_info "To view logs, use Akash CLI:"
    echo "  akash provider lease-logs --dseq $dseq --provider <provider_address>"
    echo ""
    log_info "Or check Cloudmos Console:"
    echo "  https://deploy.cloudmos.io/deployment/$dseq"
}

# List active deployments
cmd_list() {
    log_info "Fetching deployments..."
    
    local response
    response=$(api_get "/deployments")
    
    echo "$response" | jq -r '.deployments[] | select(.status == "active") | [.dseq, .provider, .uri] | @tsv' 2>/dev/null || \
        echo "$response" | jq .
}

# Show help
cmd_help() {
    cat << EOF
Akash Deployment Script for EntropyTunnel

Usage: $0 [command] [options]

Commands:
  create              Create new deployment
  status [DSEQ]       Check deployment status
  close [DSEQ]        Close deployment
  logs [DSEQ]         Show how to get logs
  list                List active deployments
  help                Show this help

Environment Variables:
  AKASH_API_KEY       Your Akash API key (required)
  SDL_FILE            Path to SDL file (default: deployments/akash/xray-server.yaml)

Examples:
  # Create new deployment
  AKASH_API_KEY=your-key ./deploy-akash.sh create

  # Check status of saved deployment
  ./deploy-akash.sh status

  # Close deployment
  ./deploy-akash.sh close

EOF
}

# Main
main() {
    check_deps
    
    local cmd="${1:-help}"
    shift || true
    
    case "$cmd" in
        create)
            cmd_create "$@"
            ;;
        status)
            cmd_status "$@"
            ;;
        close)
            cmd_close "$@"
            ;;
        logs)
            cmd_logs "$@"
            ;;
        list)
            cmd_list "$@"
            ;;
        help|--help|-h)
            cmd_help
            ;;
        *)
            log_error "Unknown command: $cmd"
            cmd_help
            exit 1
            ;;
    esac
}

main "$@"
