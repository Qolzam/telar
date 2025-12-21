#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
LOG_DIR="/tmp/telar-logs"
BIN_DIR="/tmp/telar-bin"

check_port_free() {
    local port=$1
    local name=$2
    if lsof -ti:$port >/dev/null 2>&1; then
        log_error "Port $port is in use by another process. Cannot start $name."
        log_error "Run 'make stop-servers' first."
        return 1
    fi
    return 0
}

prepare_env() {
    mkdir -p "$LOG_DIR" "$BIN_DIR"
    local api_port="${API_PORT:-9099}"
    local web_port="${WEB_PORT:-3000}"
    check_port_free "$api_port" "API Server" || exit 1
    check_port_free "$web_port" "Web Server" || exit 1
}

build_api() {
    log_info "Compiling Go API..."
    cd "$PROJECT_ROOT/apps/api"
    if go build -o "$BIN_DIR/server" cmd/server/main.go; then
        log_success "API Compiled."
    else
        log_error "API Build Failed."
        exit 1
    fi
}

start_api() {
    log_info "Starting API Server (background)..."
    cd "$PROJECT_ROOT/apps/api"
    local api_port="${API_PORT:-9099}"
    RECAPTCHA_DISABLED=true nohup "$BIN_DIR/server" > "$LOG_DIR/api.log" 2>&1 &
    local pid=$!
    echo "$pid" > "$LOG_DIR/api.pid"
    local attempts=0
    while ! lsof -ti:$api_port >/dev/null 2>&1; do
        sleep 0.5
        attempts=$((attempts+1))
        if [ $attempts -gt 20 ]; then
            log_error "API failed to bind port $api_port after 10s. Check logs:"
            tail -n 10 "$LOG_DIR/api.log"
            exit 1
        fi
    done
    log_success "API Server running (PID: $pid) on port $api_port"
}

start_web() {
    log_info "Starting Web Server (pnpm dev)..."
    cd "$PROJECT_ROOT/apps/web"
    local web_port="${WEB_PORT:-3000}"
    nohup pnpm dev > "$LOG_DIR/web.log" 2>&1 &
    local pid=$!
    echo "$pid" > "$LOG_DIR/web.pid"
    local attempts=0
    while ! lsof -ti:$web_port >/dev/null 2>&1; do
        sleep 1
        attempts=$((attempts+1))
        if [ $attempts -gt 30 ]; then
             log_warn "Web Server is taking a long time. It might still be compiling."
             break
        fi
    done
    log_success "Web Server process launched (PID: $pid)"
}

main() {
    log_banner "🚀 Launching Telar Environment (Background Mode)"
    prepare_env
    build_api
    start_api
    start_web
    local api_port="${API_PORT:-9099}"
    local web_port="${WEB_PORT:-3000}"
    echo ""
    log_banner "✅ Environment Active"
    echo "   📡 API: http://localhost:$api_port"
    echo "   🌐 Web: http://localhost:$web_port"
    echo "   📜 Logs: $LOG_DIR/api.log | $LOG_DIR/web.log"
    echo "   🛑 Stop: make stop-servers"
}

main "$@"

