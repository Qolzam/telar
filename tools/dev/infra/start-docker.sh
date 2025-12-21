#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

DOCKER_START_TIMEOUT=120
DOCKER_CHECK_INTERVAL=2

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_highlight() {
    echo -e "${CYAN}[HIGHLIGHT]${NC} $1"
}

check_docker_running() {
    if docker info >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

get_docker_status() {
    if check_docker_running; then
        log_success "Docker daemon is running"
        docker version --format "Client: {{.Client.Version}}, Server: {{.Server.Version}}" 2>/dev/null || true
        docker system info --format "Containers: {{.Containers}}, Images: {{.Images}}, Memory: {{.MemTotal}}" 2>/dev/null || true
        return 0
    else
        log_warning "Docker daemon is not running"
        return 1
    fi
}

start_docker_desktop() {
    log_info "Starting Docker Desktop..."
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        log_info "Detected macOS - starting Docker Desktop application..."
        if command -v open >/dev/null 2>&1; then
            open -g -a Docker
            log_success "Docker Desktop application started"
        else
            log_error "Cannot start Docker Desktop - 'open' command not available"
            return 1
        fi
    else
        log_warning "Non-macOS system detected. Please start Docker manually."
        log_info "On Linux, try: sudo systemctl start docker"
        log_info "On Windows, start Docker Desktop from the Start menu"
        return 1
    fi
}

wait_for_docker() {
    local timeout="$1"
    local elapsed=0
    
    log_info "Waiting for Docker daemon to be ready..."
    log_info "Timeout: ${timeout}s, Check interval: ${DOCKER_CHECK_INTERVAL}s"
    
    while [ $elapsed -lt $timeout ]; do
        if check_docker_running; then
            log_success "Docker daemon is ready!"
            return 0
        fi
        
        sleep $DOCKER_CHECK_INTERVAL
        elapsed=$((elapsed + DOCKER_CHECK_INTERVAL))
        printf "."
        if [ $((elapsed % 10)) -eq 0 ]; then
            printf " (${elapsed}s)"
        fi
    done
    
    log_error "Docker daemon did not start within ${timeout}s"
    return 1
}

verify_docker_functionality() {
    log_info "Verifying Docker functionality..."
    
    if docker ps >/dev/null 2>&1; then
        log_success "Docker 'ps' command working"
    else
        log_warning "Docker 'ps' command failed"
        return 1
    fi
    
    if docker images >/dev/null 2>&1; then
        log_success "Docker 'images' command working"
    else
        log_warning "Docker 'images' command failed"
        return 1
    fi
    
    if docker system info >/dev/null 2>&1; then
        log_success "Docker system info accessible"
    else
        log_warning "Docker system info not accessible"
        return 1
    fi
    
    log_success "Docker functionality verified"
    return 0
}

main() {
    log_info "🐳 Telar Docker Daemon Management"
    log_info "================================="
    
    if check_docker_running; then
        log_success "Docker is already running"
        get_docker_status
        verify_docker_functionality
        log_info "================================="
        log_success "✅ Docker is ready for development"
        return 0
    fi
    
    log_warning "Docker daemon is not running"
    
    if start_docker_desktop; then
        if wait_for_docker $DOCKER_START_TIMEOUT; then
            if verify_docker_functionality; then
                log_info "================================="
                log_success "✅ Docker is ready for development"
                return 0
            else
                log_error "Docker started but functionality verification failed"
                return 1
            fi
        else
            log_error "Docker daemon failed to start within timeout"
            return 1
        fi
    else
        log_error "Failed to start Docker Desktop"
        log_info "Please start Docker manually and try again"
        return 1
    fi
}

trap 'log_warning "Script interrupted"; exit 130' INT TERM

main "$@"

