#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
LOG_DIR="/tmp/telar-logs"
BIN_DIR="/tmp/telar-bin"
API_PORT="${API_PORT:-9099}"

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
    check_port_free "$API_PORT" "API Server" || exit 1
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
    log_info "Starting API Server (headless mode)..."
    cd "$PROJECT_ROOT/apps/api"
    RECAPTCHA_DISABLED=true nohup "$BIN_DIR/server" > "$LOG_DIR/api.log" 2>&1 &
    local pid=$!
    echo "$pid" > "$LOG_DIR/api.pid"
    
    local attempts=0
    while ! lsof -ti:$API_PORT >/dev/null 2>&1; do
        sleep 0.5
        attempts=$((attempts+1))
        if [ $attempts -gt 20 ]; then
            log_error "API failed to bind port $API_PORT after 10s. Check logs:"
            tail -n 10 "$LOG_DIR/api.log"
            exit 1
        fi
    done
    log_success "API Server running (PID: $pid) on port $API_PORT"
}

main() {
    log_info "🚀 Starting API Server (Headless Mode)"
    prepare_env
    build_api
    start_api
    log_info "API available at: http://localhost:$API_PORT"
    log_info "Logs: $LOG_DIR/api.log"
    log_info "PID: $(cat "$LOG_DIR/api.pid")"
}

main "$@"

