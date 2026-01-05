#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
AI_ENGINE_PORT="${AI_ENGINE_PORT:-9066}"
WEAVIATE_PORT="${WEAVIATE_PORT:-9077}"

check_docker() {
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
}

check_port_free() {
    local port=$1
    local name=$2
    if lsof -ti:$port >/dev/null 2>&1; then
        log_error "Port $port is in use by another process. Cannot start $name."
        log_error "Run 'make ai-engine-stop' first."
        return 1
    fi
    return 0
}

prepare_env() {
    check_docker
    check_port_free "$AI_ENGINE_PORT" "AI Engine" || exit 1
    check_port_free "$WEAVIATE_PORT" "Weaviate" || exit 1
}

start_ai_engine() {
    log_info "Starting AI Engine services (Docker Compose)..."
    cd "$PROJECT_ROOT"
    
    # Use run_dev.sh to start services (it handles its own output)
    if bash apps/ai-engine/run_dev.sh start; then
        log_success "AI Engine services started successfully"
    else
        log_error "Failed to start AI Engine services"
        exit 1
    fi
    
    # Wait for services to be ready
    log_info "Waiting for AI Engine to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s http://localhost:${AI_ENGINE_PORT}/health >/dev/null 2>&1; then
            log_success "AI Engine is healthy and ready!"
            break
        fi
        
        if [[ $attempt -eq $max_attempts ]]; then
            log_error "AI Engine failed to start within expected time"
            log_info "Check Docker logs: docker compose -f apps/ai-engine/deployments/docker-compose/docker-compose.yml logs"
            exit 1
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    echo ""
}

main() {
    log_info "🚀 Starting AI Engine Services"
    prepare_env
    start_ai_engine
    log_info "AI Engine available at: http://localhost:$AI_ENGINE_PORT"
    log_info "Weaviate available at: http://localhost:$WEAVIATE_PORT"
    log_info "Health check: http://localhost:$AI_ENGINE_PORT/health"
}

main "$@"

