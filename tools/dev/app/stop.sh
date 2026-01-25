#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

LOG_DIR="/tmp/telar-logs"

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

kill_port() {
    local port=$1
    local service_name=$2

    log_info "Checking port $port ($service_name)..."

    if ! command_exists lsof; then
        log_error "lsof not found. Please install lsof to use this script."
        exit 1
    fi

    local pids=$(lsof -ti:$port 2>/dev/null || true)

    if [ -z "$pids" ]; then
        log_info "Port $port is already free."
        return 0
    fi

    log_warning "Found process(es) on port $port: $pids"
    echo "$pids" | xargs kill -15 2>/dev/null || true
    
    local i=0
    while [ $i -lt 10 ]; do
        if ! lsof -ti:$port >/dev/null 2>&1; then
            break
        fi
        sleep 0.5
        i=$((i+1))
    done

    local remaining=$(lsof -ti:$port 2>/dev/null || true)
    if [ -n "$remaining" ]; then
        log_warning "Process on port $port stuck. Force killing..."
        echo "$remaining" | xargs kill -9 2>/dev/null || true
        sleep 1
    fi

    if lsof -ti:$port >/dev/null 2>&1; then
        log_error "Failed to clear port $port. Manual intervention required."
    else
        log_success "Port $port cleared."
    fi
}

cleanup_node_zombies() {
    log_info "Hunting for orphaned Next.js processes..."
    local pids=$(pgrep -f "next-server|next dev|next-router" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        log_warning "Found potential zombie Next.js processes: $pids"
        echo "$pids" | xargs kill -9 2>/dev/null || true
        sleep 1
        log_success "Zombies neutralized."
    else
        log_info "No zombies found."
    fi
}

cleanup_files() {
    log_info "Cleaning up PID and Log files..."
    rm -f "$LOG_DIR"/*.pid 2>/dev/null || true
    log_success "Cleanup complete."
}

stop_ai_engine() {
    log_info "Stopping AI Engine services (Docker Compose)..."
    local project_root="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
    local docker_compose_dir="$project_root/apps/ai-engine/deployments/docker-compose"
    
    if [[ -f "$docker_compose_dir/docker-compose.yml" ]]; then
        cd "$docker_compose_dir"
        if docker compose down >/dev/null 2>&1; then
            log_success "AI Engine services stopped"
        else
            log_warn "AI Engine services may not have been running"
        fi
    else
        log_warn "Docker Compose file not found, skipping AI Engine stop"
    fi
}

main() {
    log_info "🛑 Stopping Telar Development Servers..."
    local web_port="${WEB_PORT:-4000}"
    local api_port="${API_PORT:-9099}"
    local profile_port="${PROFILE_PORT:-8081}"
    local posts_port="${POSTS_PORT:-8082}"
    local comments_port="${COMMENTS_PORT:-8083}"
    local ai_engine_port="${AI_ENGINE_PORT:-9066}"
    local weaviate_port="${WEAVIATE_PORT:-9077}"
    
    kill_port "$web_port" "Next.js Web"
    kill_port "$api_port" "Go API Server"
    kill_port "$profile_port" "Profile Service (Standalone)"
    kill_port "$posts_port" "Posts Service (Standalone)"
    kill_port "$comments_port" "Comments Service (Standalone)"
    stop_ai_engine
    kill_port "$ai_engine_port" "AI Engine" || true
    kill_port "$weaviate_port" "Weaviate" || true
    cleanup_node_zombies
    cleanup_files
    log_success "✅ All servers stopped and ports cleared."
}

main "$@"

