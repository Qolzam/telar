#!/bin/bash

# ============================================================================
# Telar MongoDB Replica Set Setup Script
# ============================================================================
# 
# This script follows the Telar Professional Architecture Blueprint:
# - Located in tools/dev/scripts/ as per architecture guidelines
# - Handles MongoDB replica set configuration for transaction testing
# - Provides comprehensive error handling and status reporting
# - Supports enterprise transaction management testing
#
# Usage: ./tools/dev/scripts/setup-mongo-replica.sh
# Or via Makefile: make up-both-replica
# ============================================================================

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
readonly TEST_ENV_SCRIPT="${PROJECT_ROOT}/tools/dev/test_env.sh"

# MongoDB configuration
readonly MONGO_CONTAINER_NAME="telar-mongo"
readonly MONGO_PORT="27017"
readonly REPLICA_SET_NAME="rs0"
readonly MONGO_IMAGE="mongo:6"

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
# Container Management Functions
# ============================================================================

# Check if container exists
container_exists() {
    local container_name="$1"
    docker ps -a --format "{{.Names}}" | grep -q "^${container_name}$" 2>/dev/null
}

# Check if container is running
container_running() {
    local container_name="$1"
    docker ps --format "{{.Names}}" | grep -q "^${container_name}$" 2>/dev/null
}

# Stop and remove existing container
cleanup_existing_container() {
    local container_name="$1"
    
    if container_exists "$container_name"; then
        log_info "Found existing $container_name container"
        
        if container_running "$container_name"; then
            log_info "Stopping $container_name container..."
            docker stop "$container_name" >/dev/null 2>&1 || true
        fi
        
        log_info "Removing $container_name container..."
        docker rm "$container_name" >/dev/null 2>&1 || true
        
        log_success "Cleaned up existing $container_name container"
    fi
}

# Wait for MongoDB to be ready
wait_for_mongodb() {
    local container_name="$1"
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for MongoDB to be ready..."
    
    while [ $attempt -lt $max_attempts ]; do
        if docker exec "$container_name" mongosh --eval "db.runCommand('ping')" >/dev/null 2>&1; then
            log_success "MongoDB is ready"
            return 0
        fi
        
        sleep 1
        attempt=$((attempt + 1))
        printf "."
    done
    
    log_warning "MongoDB may not be fully ready yet"
    return 1
}

# ============================================================================
# MongoDB Replica Set Functions
# ============================================================================

# Start MongoDB container with replica set
start_mongodb_replica() {
    log_info "Starting MongoDB container with replica set configuration..."
    
    # Clean up existing container
    cleanup_existing_container "$MONGO_CONTAINER_NAME"
    
    # Start new MongoDB container with replica set
    log_info "Creating MongoDB container with replica set: $REPLICA_SET_NAME"
    docker run -d \
        --name "$MONGO_CONTAINER_NAME" \
        -p "$MONGO_PORT:$MONGO_PORT" \
        "$MONGO_IMAGE" \
        --replSet "$REPLICA_SET_NAME"
    
    if [ $? -eq 0 ]; then
        log_success "MongoDB container started successfully"
    else
        log_error "Failed to start MongoDB container"
        return 1
    fi
}

# Initialize replica set
initialize_replica_set() {
    local container_name="$1"
    local replica_set_name="$2"
    
    log_info "Initializing MongoDB replica set: $replica_set_name"
    
    # Wait for MongoDB to be ready
    if ! wait_for_mongodb "$container_name"; then
        log_warning "MongoDB may not be fully ready, attempting replica set initialization anyway"
    fi
    
    # Initialize replica set
    local init_command="rs.initiate({_id: '$replica_set_name', members: [{_id: 0, host: 'localhost:$MONGO_PORT'}]})"
    
    log_info "Executing replica set initialization..."
    if docker exec "$container_name" mongosh --eval "$init_command" >/dev/null 2>&1; then
        log_success "MongoDB replica set initialized successfully"
    else
        # Check if replica set is already initialized
        if docker exec "$container_name" mongosh --eval "rs.status()" >/dev/null 2>&1; then
            log_success "MongoDB replica set was already initialized"
        else
            log_error "Failed to initialize MongoDB replica set"
            return 1
        fi
    fi
}

# Verify replica set status
verify_replica_set() {
    local container_name="$1"
    
    log_info "Verifying replica set status..."
    
    if docker exec "$container_name" mongosh --eval "rs.status()" >/dev/null 2>&1; then
        log_success "Replica set is active and ready"
        
        # Display replica set status
        log_info "Replica set status:"
        docker exec "$container_name" mongosh --eval "rs.status().ok" 2>/dev/null || true
    else
        log_error "Replica set verification failed"
        return 1
    fi
}

# ============================================================================
# PostgreSQL Setup Functions
# ============================================================================

# Start PostgreSQL using test environment script
start_postgresql() {
    log_info "Starting PostgreSQL database..."
    
    if [ -f "$TEST_ENV_SCRIPT" ]; then
        log_info "Using test environment script to start PostgreSQL..."
        "$TEST_ENV_SCRIPT" up postgres
        log_success "PostgreSQL started successfully"
    else
        log_error "Test environment script not found: $TEST_ENV_SCRIPT"
        return 1
    fi
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    log_info "🗄️  Telar MongoDB Replica Set Setup"
    log_info "===================================="
    
    # Change to project root for consistent behavior
    cd "$PROJECT_ROOT"
    
    # Step 1: Clean up existing databases
    log_info "Step 1: Cleaning up existing database containers..."
    if [ -f "$TEST_ENV_SCRIPT" ]; then
        log_info "Stopping existing database containers..."
        "$TEST_ENV_SCRIPT" down both 2>/dev/null || true
    fi
    
    # Step 2: Start MongoDB with replica set
    log_info "Step 2: Starting MongoDB with replica set configuration..."
    if ! start_mongodb_replica; then
        log_error "Failed to start MongoDB container"
        exit 1
    fi
    
    # Step 3: Wait for MongoDB to be ready
    log_info "Step 3: Waiting for MongoDB to be ready..."
    if ! wait_for_mongodb "$MONGO_CONTAINER_NAME"; then
        log_warning "MongoDB may not be fully ready, continuing with setup"
    fi
    
    # Step 4: Initialize replica set
    log_info "Step 4: Initializing MongoDB replica set..."
    if ! initialize_replica_set "$MONGO_CONTAINER_NAME" "$REPLICA_SET_NAME"; then
        log_error "Failed to initialize replica set"
        exit 1
    fi
    
    # Step 5: Verify replica set
    log_info "Step 5: Verifying replica set status..."
    if ! verify_replica_set "$MONGO_CONTAINER_NAME"; then
        log_error "Replica set verification failed"
        exit 1
    fi
    
    # Step 6: Start PostgreSQL
    log_info "Step 6: Starting PostgreSQL database..."
    if ! start_postgresql; then
        log_error "Failed to start PostgreSQL"
        exit 1
    fi
    
    # Final status
    log_info "===================================="
    log_success "✅ MongoDB replica set setup complete"
    log_highlight "MongoDB: localhost:$MONGO_PORT (replica set: $REPLICA_SET_NAME)"
    log_highlight "PostgreSQL: localhost:5432"
    log_info "Ready for enterprise transaction testing"
}

# ============================================================================
# Script Entry Point
# ============================================================================

# Handle script interruption gracefully
trap 'log_warning "Script interrupted"; exit 130' INT TERM

# Run main function
main "$@"







