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
readonly LOG_DIR="/tmp/telar-logs"

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

log_error() {
    echo -e "\033[0;31m[ERROR]\033[0m $1"
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
    
    # Pattern 2: Kill processes running /tmp/telar-api-server binary
    pids=$(pgrep -f "/tmp/telar-api-server" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        for pid in $pids; do
            kill -TERM "$pid" 2>/dev/null || true
            sleep 1
            if kill -0 "$pid" 2>/dev/null; then
                kill -KILL "$pid" 2>/dev/null || true
            fi
        done
    fi
    
    # Pattern 3: Only kill "main" processes that are in our project directory
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

# Ensure log directory exists
setup_log_directory() {
    mkdir -p "$LOG_DIR" 2>/dev/null || true
}

# Clean Go build artifacts and cache
clean_go_build() {
    log_info "Cleaning Go build cache and artifacts..."
    
    cd "$PROJECT_ROOT/apps/api"
    
    # Clean Go build cache (forces fresh rebuild)
    # Note: -modcache removes ALL modules, but we use it to ensure fresh dependencies
    # This is safe in development but might be slow on first rebuild
    log_info "Clearing Go build cache..."
    go clean -cache 2>/dev/null || true
    go clean -testcache 2>/dev/null || true
    # Only clean modcache if you want to re-download all dependencies (commented for speed)
    # go clean -modcache 2>/dev/null || true
    
    # Remove any compiled binaries in the project directory
    find . -maxdepth 3 -name "main" -type f -executable -delete 2>/dev/null || true
    find . -maxdepth 3 -name "*.exe" -type f -delete 2>/dev/null || true
    find . -maxdepth 3 -name "*.test" -type f -executable -delete 2>/dev/null || true
    
    # Remove Go's temporary build files
    find . -type d -name "__pycache__" -prune -o -name "*.a" -type f -delete 2>/dev/null || true
    
    log_info "Go build cache and artifacts cleaned"
}

# Build Go API server (compiles and outputs binary)
build_go_api() {
    log_info "Building Go API server from source..."
    
    cd "$PROJECT_ROOT/apps/api"
    
    # Build the server binary (will be used to run the server)
    if go build -o /tmp/telar-api-server cmd/server/main.go 2>&1; then
        log_info "Go API server compiled successfully"
        return 0
    else
        log_error "Go API server failed to compile!"
        rm -f /tmp/telar-api-server 2>/dev/null || true
        return 1
    fi
}

# Start servers
start_servers() {
    log_info "Starting servers with fresh build..."
    
    cd "$PROJECT_ROOT"
    
    # Step 1: Clean Go build cache
    clean_go_build
    
    # Step 2: Build Go API server (compiles from source)
    if ! build_go_api; then
        log_error "Cannot start server - compilation failed"
        return 1
    fi
    
    # Step 3: Run the freshly built binary
    log_info "Starting Go API server with freshly built binary..."
    cd "$PROJECT_ROOT/apps/api"
    nohup /tmp/telar-api-server > "$LOG_DIR/api.log" 2>&1 &
    local api_pid=$!
    echo "$api_pid" > "$LOG_DIR/api.pid" 2>/dev/null || true
    log_info "Go API server starting (PID: $api_pid)"
    sleep 2
    
    # Step 4: Start web server
    log_info "Starting Next.js web server..."
    cd "$PROJECT_ROOT/apps/web"
    nohup pnpm dev > "$LOG_DIR/web.log" 2>&1 &
    local web_pid=$!
    echo "$web_pid" > "$LOG_DIR/web.pid" 2>/dev/null || true
    log_info "Next.js web server starting (PID: $web_pid)"
    sleep 2
    
    cd "$PROJECT_ROOT"
    log_info "Servers started with fresh builds"
}

# Main function
main() {
    echo "🔄 Restarting servers with full rebuild (preserving Cursor processes)..."
    
    cd "$PROJECT_ROOT"
    
    # Step 1: Setup log directory
    setup_log_directory
    
    # Step 2: Stop servers safely
    stop_servers_safe
    
    # Step 3: Start databases
    start_databases
    
    # Step 4: Start servers with fresh build
    start_servers
    
    log_info "✅ Servers restarted with fresh builds"
    echo "📡 API: http://localhost:8080"
    echo "🌐 Web: http://localhost:3000"
    echo ""
    echo "📋 Logs:"
    echo "   API: tail -f $LOG_DIR/api.log"
    echo "   Web: tail -f $LOG_DIR/web.log"
}

# Run main function
main "$@"
