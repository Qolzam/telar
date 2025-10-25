#!/bin/bash

# ============================================================================
# Telar Docker Daemon Management Script
# ============================================================================
# 
# This script follows the Telar Professional Architecture Blueprint:
# - Located in tools/dev/scripts/ as per architecture guidelines
# - Handles Docker daemon startup and verification
# - Supports macOS Docker Desktop with proper timeout handling
# - Provides comprehensive status reporting and error handling
#
# Usage: ./tools/dev/scripts/docker-start.sh
# Or via Makefile: make docker-start
# ============================================================================

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Docker configuration
readonly DOCKER_START_TIMEOUT=120
readonly DOCKER_CHECK_INTERVAL=2

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
# Docker Management Functions
# ============================================================================

# Check if Docker is running
check_docker_running() {
    if docker info >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Get Docker daemon status
get_docker_status() {
    if check_docker_running; then
        log_success "Docker daemon is running"
        
        # Display Docker version info
        log_info "Docker version information:"
        docker version --format "Client: {{.Client.Version}}, Server: {{.Server.Version}}" 2>/dev/null || true
        
        # Display Docker system info
        log_info "Docker system information:"
        docker system info --format "Containers: {{.Containers}}, Images: {{.Images}}, Memory: {{.MemTotal}}" 2>/dev/null || true
        
        return 0
    else
        log_warning "Docker daemon is not running"
        return 1
    fi
}

# Start Docker Desktop on macOS
start_docker_desktop() {
    log_info "Starting Docker Desktop..."
    
    # Check if we're on macOS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        log_info "Detected macOS - starting Docker Desktop application..."
        
        # Start Docker Desktop application
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

# Wait for Docker daemon to be ready
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
        
        # Show progress
        printf "."
        
        # Show progress every 10 seconds
        if [ $((elapsed % 10)) -eq 0 ]; then
            printf " (${elapsed}s)"
        fi
    done
    
    log_error "Docker daemon did not start within ${timeout}s"
    return 1
}

# Verify Docker functionality
verify_docker_functionality() {
    log_info "Verifying Docker functionality..."
    
    # Test basic Docker commands
    if docker ps >/dev/null 2>&1; then
        log_success "Docker 'ps' command working"
    else
        log_warning "Docker 'ps' command failed"
        return 1
    fi
    
    # Test Docker image listing
    if docker images >/dev/null 2>&1; then
        log_success "Docker 'images' command working"
    else
        log_warning "Docker 'images' command failed"
        return 1
    fi
    
    # Test Docker system info
    if docker system info >/dev/null 2>&1; then
        log_success "Docker system info accessible"
    else
        log_warning "Docker system info not accessible"
        return 1
    fi
    
    log_success "Docker functionality verified"
    return 0
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    log_info "🐳 Telar Docker Daemon Management"
    log_info "================================="
    
    # Check if Docker is already running
    if check_docker_running; then
        log_success "Docker is already running"
        get_docker_status
        verify_docker_functionality
        log_info "================================="
        log_success "✅ Docker is ready for development"
        return 0
    fi
    
    # Docker is not running, attempt to start it
    log_warning "Docker daemon is not running"
    
    # Try to start Docker Desktop
    if start_docker_desktop; then
        # Wait for Docker to be ready
        if wait_for_docker $DOCKER_START_TIMEOUT; then
            # Verify Docker functionality
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

# ============================================================================
# Script Entry Point
# ============================================================================

# Handle script interruption gracefully
trap 'log_warning "Script interrupted"; exit 130' INT TERM

# Run main function
main "$@"







