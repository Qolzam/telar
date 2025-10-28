#!/bin/bash

# ============================================================================
# Telar Development Server Management Script
# ============================================================================
# 
# This script follows the Telar Professional Architecture Blueprint:
# - Located in tools/dev/scripts/ as per architecture guidelines
# - Handles graceful shutdown of development servers
# - Supports both Go API and Next.js web servers
# - Provides clear feedback and error handling
#
# Usage: ./tools/dev/scripts/stop-servers.sh
# Or via Makefile: make stop-servers
# ============================================================================

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
readonly LOG_DIR="/tmp/telar-logs"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
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

# ============================================================================
# Process Management Functions
# ============================================================================

# Graceful shutdown of a process
graceful_shutdown() {
    local pid="$1"
    local service_name="$2"
    
    log_info "Stopping ${service_name} (PID: ${pid})..."
    
    # Send TERM signal for graceful shutdown
    if kill -TERM "$pid" 2>/dev/null; then
        # Wait for graceful shutdown
        local count=0
        while kill -0 "$pid" 2>/dev/null && [ $count -lt 10 ]; do
            sleep 0.5
            count=$((count + 1))
        done
        
        # Check if process is still running
        if kill -0 "$pid" 2>/dev/null; then
            log_warning "${service_name} (PID: ${pid}) did not shutdown gracefully, force killing..."
            kill -KILL "$pid" 2>/dev/null || true
        else
            log_success "${service_name} (PID: ${pid}) stopped gracefully"
        fi
    else
        log_warning "Could not send TERM signal to ${service_name} (PID: ${pid})"
    fi
}

# ============================================================================
# Server Management Functions
# ============================================================================

# Stop Go API server
stop_go_api() {
    local pids=""
    
    # Pattern 1: "go run cmd/server/main.go" (very specific)
    pids=$(pgrep -f "go run cmd/server/main.go" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        log_info "Found Go API server processes (go run): $pids"
        for pid in $pids; do
            graceful_shutdown "$pid" "Go API Server"
        done
    fi
    
    # Pattern 2: Only kill "main" processes that are in our project directory
    pids=$(pgrep -f "main" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        for pid in $pids; do
            local cmdline=$(cat /proc/$pid/cmdline 2>/dev/null | tr '\0' ' ' || echo "")
            if [[ "$cmdline" == *"$PROJECT_ROOT"* ]] && [[ "$cmdline" == *"main"* ]]; then
                log_info "Found Go API server process (main binary): $pid"
                graceful_shutdown "$pid" "Go API Server"
            fi
        done
    fi
    
    if [ -z "$pids" ]; then
        log_info "No Go API server processes found"
    fi
}

# Stop Next.js web server
stop_nextjs() {
    local pids
    pids=$(pgrep -f "next dev" 2>/dev/null || true)
    
    if [ -z "$pids" ]; then
        log_info "No Next.js server processes found"
        return 0
    fi
    
    log_info "Found Next.js server processes: $pids"
    
    for pid in $pids; do
        graceful_shutdown "$pid" "Next.js Server"
    done
}

# Clean up log files and PID files
cleanup_logs() {
    log_info "Cleaning up log files and PID files..."
    
    if [ -d "$LOG_DIR" ]; then
        rm -f "${LOG_DIR}"/*.pid 2>/dev/null || true
        log_success "Cleaned up PID files from ${LOG_DIR}"
    else
        log_info "Log directory ${LOG_DIR} does not exist, skipping cleanup"
    fi
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    log_info "🛑 Telar Development Server Shutdown"
    log_info "======================================"
    
    # Change to project root for consistent behavior
    cd "$PROJECT_ROOT"
    
    # Stop servers in order (web first, then API)
    log_info "Stopping development servers..."
    stop_nextjs
    stop_go_api
    
    # Clean up
    cleanup_logs
    
    # Final verification
    local remaining_processes=""
    
    # Check for remaining Go API processes
    local go_pids=$(pgrep -f "go run cmd/server/main.go" 2>/dev/null || true)
    if [ -n "$go_pids" ]; then
        remaining_processes="$go_pids"
    fi
    
    # Check for remaining main processes in project directory
    local main_pids=$(pgrep -f "main" 2>/dev/null || true)
    if [ -n "$main_pids" ]; then
        for pid in $main_pids; do
            local cmdline=$(cat /proc/$pid/cmdline 2>/dev/null | tr '\0' ' ' || echo "")
            if [[ "$cmdline" == *"$PROJECT_ROOT"* ]] && [[ "$cmdline" == *"main"* ]]; then
                if [ -n "$remaining_processes" ]; then
                    remaining_processes="$remaining_processes $pid"
                else
                    remaining_processes="$pid"
                fi
            fi
        done
    fi
    
    # Check for remaining Next.js processes
    local nextjs_pids=$(pgrep -f "next dev" 2>/dev/null || true)
    if [ -n "$nextjs_pids" ]; then
        if [ -n "$remaining_processes" ]; then
            remaining_processes="$remaining_processes $nextjs_pids"
        else
            remaining_processes="$nextjs_pids"
        fi
    fi
    
    if [ -n "$remaining_processes" ]; then
        log_warning "Some processes may still be running: $remaining_processes"
        log_warning "You may need to manually kill them if they persist"
    else
        log_success "✅ All development servers stopped successfully"
    fi
    
    log_info "======================================"
    log_info "Development environment shutdown complete"
}

# ============================================================================
# Script Entry Point
# ============================================================================

# Handle script interruption gracefully
trap 'log_warning "Script interrupted"; exit 130' INT TERM

# Run main function
main "$@"
