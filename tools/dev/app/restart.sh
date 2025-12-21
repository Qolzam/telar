#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
LOG_DIR="/tmp/telar-logs"

kill_port() {
    local port=$1
    local name=$2
    log_info "Checking port $port ($name)..."
    local pids=$(lsof -ti:$port 2>/dev/null || true)
    if [ -z "$pids" ]; then
        return 0
    fi
    log_warn "Found processes holding port $port: $pids"
    echo "$pids" | xargs kill -15 2>/dev/null || true
    sleep 1
    local remaining=$(lsof -ti:$port 2>/dev/null || true)
    if [ -n "$remaining" ]; then
        log_warn "Processes refused to die. Force killing..."
        echo "$remaining" | xargs kill -9 2>/dev/null || true
    fi
    if lsof -ti:$port >/dev/null 2>&1; then
        log_error "Failed to clear port $port. Manual intervention required."
        exit 1
    fi
    log_info "Port $port cleared."
}

stop_servers_safe() {
    log_info "Stopping servers by Port Authority..."
    kill_port 9099 "API Server"
    kill_port 3000 "Web Server"
    kill_port 8081 "Profile Service"
    rm -f "$LOG_DIR/api.pid" "$LOG_DIR/web.pid"
    log_info "Servers stopped safely"
}

start_databases() {
    log_info "Starting databases..."
    cd "$PROJECT_ROOT"
    make up-dbs-dev >/dev/null 2>&1
    log_info "Databases started"
}

setup_log_directory() {
    mkdir -p "$LOG_DIR" 2>/dev/null || true
}

clean_go_build() {
    log_info "Cleaning Go build cache and artifacts..."
    cd "$PROJECT_ROOT/apps/api"
    log_info "Clearing Go build cache..."
    go clean -cache 2>/dev/null || true
    go clean -testcache 2>/dev/null || true
    find . -maxdepth 3 -name "main" -type f -executable -delete 2>/dev/null || true
    find . -maxdepth 3 -name "*.exe" -type f -delete 2>/dev/null || true
    find . -maxdepth 3 -name "*.test" -type f -executable -delete 2>/dev/null || true
    find . -type d -name "__pycache__" -prune -o -name "*.a" -type f -delete 2>/dev/null || true
    log_info "Go build cache and artifacts cleaned"
}

build_go_api() {
    log_info "Building Go API server from source..."
    cd "$PROJECT_ROOT/apps/api"
    if go build -o /tmp/telar-api-server cmd/server/main.go 2>&1; then
        log_info "Go API server compiled successfully"
        return 0
    else
        log_error "Go API server failed to compile!"
        rm -f /tmp/telar-api-server 2>/dev/null || true
        return 1
    fi
}

start_web_server() {
    log_info "Starting Next.js web server..."
    cd "$PROJECT_ROOT/apps/web" || { log_error "Could not find apps/web"; return 1; }
    if ! command -v pnpm >/dev/null 2>&1; then
        log_error "pnpm not found in script PATH. Ensure it is installed and PATH is exported."
        return 1
    fi
    nohup setsid pnpm dev > "$LOG_DIR/web.log" 2>&1 < /dev/null &
    local web_pid=$!
    sleep 1
    if ! kill -0 "$web_pid" 2>/dev/null; then
        log_error "Web server process died immediately after launch"
        log_error "Check web.log for errors:"
        tail -n 20 "$LOG_DIR/web.log"
        return 1
    fi
    echo "$web_pid" > "$LOG_DIR/web.pid" 2>/dev/null || true
    log_info "Next.js process launched (PID: $web_pid)"
    log_info "Waiting for Web Server to bind port 3000..."
    local attempts=0
    local max_attempts=90
    while [ $attempts -lt $max_attempts ]; do
        local port_bound=false
        if lsof -ti:3000 >/dev/null 2>&1; then
            port_bound=true
        elif ss -tlnp 2>/dev/null | grep -q ":3000 "; then
            port_bound=true
        elif netstat -tlnp 2>/dev/null | grep -q ":3000 "; then
            port_bound=true
        fi
        if [ "$port_bound" = true ]; then
            if curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/login 2>/dev/null | grep -q "200\|404\|500"; then
                log_info "✓ Web Server is listening on port 3000 and responding to HTTP."
                return 0
            fi
        fi
        if curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/login 2>/dev/null | grep -q "200\|404\|500"; then
            log_info "✓ Web Server is responding to HTTP requests."
            return 0
        fi
        if grep -q "Ready in" "$LOG_DIR/web.log" 2>/dev/null; then
            sleep 5
            for i in 1 2 3 4 5; do
                if lsof -ti:3000 >/dev/null 2>&1; then
                    log_info "✓ Web Server is listening on port 3000."
                    return 0
                fi
                sleep 2
            done
            if [ -f "$LOG_DIR/web.pid" ]; then
                local saved_pid=$(cat "$LOG_DIR/web.pid" 2>/dev/null)
                if kill -0 "$saved_pid" 2>/dev/null; then
                    log_warn "Server says Ready but port not bound. Process alive, waiting longer..."
                    sleep 5
                    if lsof -ti:3000 >/dev/null 2>&1; then
                        log_info "✓ Web Server is listening on port 3000 (delayed binding)."
                        return 0
                    fi
                fi
            fi
        fi
        sleep 1
        attempts=$((attempts+1))
    done
    local port_bound=false
    if lsof -ti:3000 >/dev/null 2>&1; then
        port_bound=true
    elif ss -tlnp 2>/dev/null | grep -q ":3000 "; then
        port_bound=true
    elif netstat -tlnp 2>/dev/null | grep -q ":3000 "; then
        port_bound=true
    fi
    if [ "$port_bound" = true ]; then
        log_info "✓ Web Server is listening on port 3000 (HTTP check timed out, but port is bound)."
        return 0
    fi
    log_error "Web Server failed to start after $max_attempts seconds."
    log_error "Tail of web.log:"
    tail -n 30 "$LOG_DIR/web.log"
    return 1
}

start_servers() {
    log_info "Starting servers with fresh build..."
    cd "$PROJECT_ROOT"
    clean_go_build
    if ! build_go_api; then
        log_error "Cannot start server - compilation failed"
        return 1
    fi
    log_info "Starting Go API server with freshly built binary..."
    cd "$PROJECT_ROOT/apps/api"
    RECAPTCHA_DISABLED=true nohup /tmp/telar-api-server > "$LOG_DIR/api.log" 2>&1 &
    local api_pid=$!
    echo "$api_pid" > "$LOG_DIR/api.pid" 2>/dev/null || true
    log_info "Go API server starting (PID: $api_pid)"
    sleep 2
    start_web_server
    cd "$PROJECT_ROOT"
    log_info "Servers started with fresh builds"
}

main() {
    echo "🔄 Restarting servers with full rebuild (preserving Cursor processes)..."
    cd "$PROJECT_ROOT"
    setup_log_directory
    stop_servers_safe
    start_databases
    start_servers
    log_info "✅ Servers restarted with fresh builds"
    echo "📡 API: http://localhost:9099"
    echo "🌐 Web: http://localhost:3000"
    echo ""
    echo "📋 Logs:"
    echo "   API: tail -f $LOG_DIR/api.log"
    echo "   Web: tail -f $LOG_DIR/web.log"
}

main "$@"

