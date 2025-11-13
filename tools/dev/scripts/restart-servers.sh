#!/bin/bash

# ============================================================================
# SAFE Telar Development Server Restart Script
# ============================================================================
# 
# This script restarts servers WITHOUT killing Cursor processes
# and WITHOUT triggering Cursor reloads.
# ============================================================================

set -euo pipefail

# Configuration
readonly PROJECT_ROOT="/home/office/projects/telar/web-team/telar-new-arch"

# Colors
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Stop servers safely (using the fixed stop script)
stop_servers_safe() {
    log_info "Stopping servers safely..."
    
    # Stop Go API server (MORE SPECIFIC PATTERNS)
    local pids=""
    
    # Pattern 1: "go run cmd/server/main.go" (very specific)
    pids=$(pgrep -f "go run cmd/server/main.go" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        for pid in $pids; do
            kill -TERM "$pid" 2>/dev/null || true
            sleep 1
            if kill -0 "$pid" 2>/dev/null; then
                kill -KILL "$pid" 2>/dev/null || true
            fi
        done
    fi
    
    # Pattern 2: Only kill "main" processes that are in our project directory
    pids=$(pgrep -f "main" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        for pid in $pids; do
            local cmdline=$(cat /proc/$pid/cmdline 2>/dev/null | tr '\0' ' ' || echo "")
            if [[ "$cmdline" == *"$PROJECT_ROOT"* ]] && [[ "$cmdline" == *"main"* ]]; then
                kill -TERM "$pid" 2>/dev/null || true
                sleep 1
                if kill -0 "$pid" 2>/dev/null; then
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
        done
    fi
    
    # Stop Next.js web server
    pids=$(pgrep -f "next dev" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        for pid in $pids; do
            kill -TERM "$pid" 2>/dev/null || true
            sleep 1
            if kill -0 "$pid" 2>/dev/null; then
                kill -KILL "$pid" 2>/dev/null || true
            fi
        done
    fi
    
    log_info "Servers stopped safely"
}

# Start databases
start_databases() {
    log_info "Starting databases..."
    cd "$PROJECT_ROOT"
    make up-dbs-dev >/dev/null 2>&1
    log_info "Databases started"
}

# Start servers
start_servers() {
    log_info "Starting servers..."
    
    cd "$PROJECT_ROOT"
    
    # Start API server
    cd apps/api
    nohup go run cmd/server/main.go >/dev/null 2>&1 &
    sleep 2
    
    # Start web server
    cd ../web
    nohup pnpm dev >/dev/null 2>&1 &
    sleep 2
    
    cd "$PROJECT_ROOT"
    log_info "Servers started"
}

# Main function
main() {
    echo "🔄 Restarting servers safely (preserving Cursor processes)..."
    
    cd "$PROJECT_ROOT"
    
    # Step 1: Stop servers safely
    stop_servers_safe
    
    # Step 2: Start databases
    start_databases
    
    # Step 3: Start servers
    start_servers
    
    log_info "✅ Servers restarted safely"
    echo "📡 API: http://localhost:8080"
    echo "🌐 Web: http://localhost:3000"
}

# Run main function
main "$@"
