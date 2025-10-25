#!/bin/bash

# ============================================================================
# Telar Development Server Background Startup Script
# ============================================================================
# 
# This script follows the Telar Professional Architecture Blueprint:
# - Located in tools/dev/scripts/ as per architecture guidelines
# - Handles background server startup with proper process management
# - Supports both Go API and Next.js web servers
# - Provides comprehensive status reporting and user guidance
#
# Usage: ./tools/dev/scripts/start-servers-bg.sh
# Or via Makefile: make run-both-bg
# ============================================================================

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
readonly LOG_DIR="/tmp/telar-logs"

# Server configurations
readonly API_PORT="8080"
readonly WEB_PORT="3000"
readonly API_LOG_FILE="${LOG_DIR}/api.log"
readonly WEB_LOG_FILE="${LOG_DIR}/web.log"
readonly API_PID_FILE="${LOG_DIR}/api.pid"
readonly WEB_PID_FILE="${LOG_DIR}/web.pid"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m' # No Color

# ============================================================================
# Utility Functions
# ============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_highlight() {
    echo -e "${CYAN}[HIGHLIGHT]${NC} $1"
}

# ============================================================================
# Process Management Functions
# ============================================================================

# Check if a port is in use
check_port() {
    local port="$1"
    local service_name="$2"
    
    if lsof -i ":$port" >/dev/null 2>&1; then
        log_warning "Port $port is already in use. $service_name may already be running."
        return 1
    fi
    return 0
}

# Start a background process and save PID
start_background_process() {
    local working_dir="$1"
    local command="$2"
    local log_file="$3"
    local pid_file="$4"
    local service_name="$5"
    local port="$6"
    
    log_info "Starting $service_name in background..."
    
    # Check if port is available
    if ! check_port "$port" "$service_name"; then
        return 1
    fi
    
    # Change to working directory and start process
    cd "$working_dir"
    
    # Start the process in background with nohup
    nohup $command > "$log_file" 2>&1 &
    local pid=$!
    
    # Save PID to file
    echo "$pid" > "$pid_file"
    
    # Verify process started successfully
    if kill -0 "$pid" 2>/dev/null; then
        log_success "$service_name started successfully (PID: $pid)"
        return 0
    else
        log_error "Failed to start $service_name"
        rm -f "$pid_file"
        return 1
    fi
}

# Wait for server to be ready
wait_for_server() {
    local port="$1"
    local service_name="$2"
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for $service_name to be ready on port $port..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "http://localhost:$port" >/dev/null 2>&1 || \
           curl -s "http://localhost:$port/health" >/dev/null 2>&1 || \
           lsof -i ":$port" >/dev/null 2>&1; then
            log_success "$service_name is ready on port $port"
            return 0
        fi
        
        sleep 1
        attempt=$((attempt + 1))
        printf "."
    done
    
    log_warning "$service_name may not be fully ready yet"
    return 1
}

# ============================================================================
# Server Startup Functions
# ============================================================================

# Start Go API server
start_api_server() {
    local api_dir="${PROJECT_ROOT}/apps/api"
    local api_command="go run cmd/server/main.go"
    
    start_background_process \
        "$api_dir" \
        "$api_command" \
        "$API_LOG_FILE" \
        "$API_PID_FILE" \
        "Go API Server" \
        "$API_PORT"
}

# Start Next.js web server
start_web_server() {
    local web_dir="${PROJECT_ROOT}/apps/web"
    local web_command="pnpm dev"
    
    start_background_process \
        "$web_dir" \
        "$web_command" \
        "$WEB_LOG_FILE" \
        "$WEB_PID_FILE" \
        "Next.js Web Server" \
        "$WEB_PORT"
}

# ============================================================================
# Status Reporting Functions
# ============================================================================

# Display server status and URLs
display_server_status() {
    echo ""
    log_highlight "🚀 Telar Development Servers Started!"
    echo ""
    log_highlight "📡 API Server:  http://localhost:$API_PORT"
    log_highlight "🌐 Web Server:  http://localhost:$WEB_PORT"
    echo ""
    log_highlight "📋 View Logs:"
    echo "   API: make logs-api   (or tail -f $API_LOG_FILE)"
    echo "   Web: make logs-web   (or tail -f $WEB_LOG_FILE)"
    echo ""
    log_highlight "🛑 Stop Servers: make stop-servers"
    echo ""
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    log_info "🚀 Starting Telar Development Servers in Background"
    log_info "=================================================="
    
    # Change to project root for consistent behavior
    cd "$PROJECT_ROOT"
    
    # Create log directory
    log_info "Setting up log directory..."
    mkdir -p "$LOG_DIR"
    
    # Start servers
    log_info "Starting development servers..."
    
    local api_started=false
    local web_started=false
    
    # Start API server
    if start_api_server; then
        api_started=true
    else
        log_error "Failed to start API server"
    fi
    
    # Start Web server
    if start_web_server; then
        web_started=true
    else
        log_error "Failed to start Web server"
    fi
    
    # Wait for servers to be ready
    if [ "$api_started" = true ]; then
        wait_for_server "$API_PORT" "API Server"
    fi
    
    if [ "$web_started" = true ]; then
        wait_for_server "$WEB_PORT" "Web Server"
    fi
    
    # Display status
    if [ "$api_started" = true ] || [ "$web_started" = true ]; then
        display_server_status
    else
        log_error "Failed to start any servers"
        exit 1
    fi
    
    log_info "=================================================="
    log_success "✅ Background server startup complete"
}

# ============================================================================
# Script Entry Point
# ============================================================================

# Handle script interruption gracefully
trap 'log_warning "Script interrupted"; exit 130' INT TERM

# Run main function
main "$@"







