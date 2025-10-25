#!/bin/bash

# ============================================================================
# Telar Pre-Flight Check Script
# ============================================================================
# 
# This script follows the Telar Professional Architecture Blueprint:
# - Located in tools/dev/scripts/ as per architecture guidelines
# - Comprehensive dependency validation before server startup
# - Database connectivity, gRPC health, and service readiness checks
# - Professional logging with detailed status reporting
#
# Usage: ./tools/dev/scripts/pre-flight-check.sh
# Or via Makefile: make pre-flight-check
# ============================================================================

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Database configurations
readonly POSTGRES_HOST="localhost"
readonly POSTGRES_PORT="5432"
readonly POSTGRES_DB="telar_social_test"
readonly POSTGRES_USER="postgres"
readonly POSTGRES_PASSWORD="postgres"

readonly MONGO_HOST="localhost"
readonly MONGO_PORT="27017"
readonly MONGO_DB="telar_social"

# Service configurations
readonly API_PORT="8080"
readonly WEB_PORT="3000"
readonly GRPC_PORT="9090"

# Health check timeouts
readonly DB_TIMEOUT=10
readonly SERVICE_TIMEOUT=5
readonly GRPC_TIMEOUT=5

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly PURPLE='\033[0;35m'
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

log_section() {
    echo -e "${PURPLE}[SECTION]${NC} $1"
}

# ============================================================================
# Database Connectivity Functions
# ============================================================================

# Check PostgreSQL connectivity
check_postgresql() {
    log_info "Checking PostgreSQL connectivity..."
    
    # Check if PostgreSQL container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "telar-postgres"; then
        log_error "PostgreSQL container is not running"
        return 1
    fi
    
    # Check PostgreSQL port connectivity
    if ! nc -z "$POSTGRES_HOST" "$POSTGRES_PORT" 2>/dev/null; then
        log_error "PostgreSQL port $POSTGRES_PORT is not accessible"
        return 1
    fi
    
    # Test database connection
    if command -v psql >/dev/null 2>&1; then
        if PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;" >/dev/null 2>&1; then
            log_success "PostgreSQL connection successful"
            
            # Get PostgreSQL version and status
            local pg_version
            pg_version=$(PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT version();" 2>/dev/null | head -n1 | xargs)
            log_info "PostgreSQL Version: $pg_version"
            
            # Check database size
            local db_size
            db_size=$(PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT pg_size_pretty(pg_database_size('$POSTGRES_DB'));" 2>/dev/null | xargs)
            log_info "Database Size: $db_size"
            
            return 0
        else
            log_error "PostgreSQL authentication failed"
            return 1
        fi
    else
        log_warning "psql client not available, skipping detailed PostgreSQL checks"
        log_success "PostgreSQL port is accessible"
        return 0
    fi
}

# Check MongoDB connectivity
check_mongodb() {
    log_info "Checking MongoDB connectivity..."
    
    # Check if MongoDB container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "telar-mongo"; then
        log_error "MongoDB container is not running"
        return 1
    fi
    
    # Check MongoDB port connectivity
    if ! nc -z "$MONGO_HOST" "$MONGO_PORT" 2>/dev/null; then
        log_error "MongoDB port $MONGO_PORT is not accessible"
        return 1
    fi
    
    # Test MongoDB connection
    if command -v mongosh >/dev/null 2>&1; then
        if mongosh --host "$MONGO_HOST:$MONGO_PORT" --eval "db.runCommand('ping')" >/dev/null 2>&1; then
            log_success "MongoDB connection successful"
            
            # Get MongoDB version and status
            local mongo_version
            mongo_version=$(mongosh --host "$MONGO_HOST:$MONGO_PORT" --quiet --eval "db.version()" 2>/dev/null | xargs)
            log_info "MongoDB Version: $mongo_version"
            
            # Check database stats
            local db_stats
            db_stats=$(mongosh --host "$MONGO_HOST:$MONGO_PORT" --quiet --eval "db.stats().collections" 2>/dev/null | xargs)
            log_info "Collections Count: $db_stats"
            
            return 0
        else
            log_error "MongoDB connection failed"
            return 1
        fi
    else
        log_warning "mongosh client not available, skipping detailed MongoDB checks"
        log_success "MongoDB port is accessible"
        return 0
    fi
}

# ============================================================================
# Service Health Check Functions
# ============================================================================

# Check if service is running on port
check_service_port() {
    local service_name="$1"
    local port="$2"
    local timeout="$3"
    
    log_info "Checking $service_name on port $port..."
    
    if nc -z "localhost" "$port" 2>/dev/null; then
        log_success "$service_name is running on port $port"
        return 0
    else
        log_warning "$service_name is not running on port $port"
        return 1
    fi
}

# Check HTTP service health
check_http_service() {
    local service_name="$1"
    local port="$2"
    local endpoint="$3"
    
    log_info "Checking $service_name HTTP health..."
    
    if curl -s --max-time "$SERVICE_TIMEOUT" "http://localhost:$port$endpoint" >/dev/null 2>&1; then
        log_success "$service_name HTTP service is responding"
        return 0
    else
        log_warning "$service_name HTTP service is not responding"
        return 1
    fi
}

# ============================================================================
# gRPC Health Check Functions
# ============================================================================

# Check gRPC service health
check_grpc_health() {
    log_info "Checking gRPC service health..."
    
    # Check if gRPC port is accessible
    if ! nc -z "localhost" "$GRPC_PORT" 2>/dev/null; then
        log_warning "gRPC port $GRPC_PORT is not accessible"
        return 1
    fi
    
    # Check if grpcurl is available for health checks
    if command -v grpcurl >/dev/null 2>&1; then
        log_info "Using grpcurl for gRPC health check..."
        
        # Try to list services
        if grpcurl -plaintext "localhost:$GRPC_PORT" list >/dev/null 2>&1; then
            log_success "gRPC service is responding"
            
            # List available services
            local services
            services=$(grpcurl -plaintext "localhost:$GRPC_PORT" list 2>/dev/null | head -5)
            log_info "Available gRPC services:"
            echo "$services" | sed 's/^/  /'
            
            return 0
        else
            log_warning "gRPC service is not responding to health checks"
            return 1
        fi
    else
        log_warning "grpcurl not available, skipping detailed gRPC health checks"
        log_success "gRPC port is accessible"
        return 0
    fi
}

# ============================================================================
# System Resource Functions
# ============================================================================

# Check system resources
check_system_resources() {
    log_section "System Resources Check"
    
    # Check available memory
    local available_memory
    available_memory=$(free -h | awk '/^Mem:/ {print $7}')
    log_info "Available Memory: $available_memory"
    
    # Check disk space
    local disk_usage
    disk_usage=$(df -h . | awk 'NR==2 {print $5}')
    log_info "Disk Usage: $disk_usage"
    
    # Check Docker resources
    if command -v docker >/dev/null 2>&1; then
        local docker_info
        docker_info=$(docker system df --format "table {{.Type}}\t{{.TotalCount}}\t{{.Size}}" 2>/dev/null | head -3)
        log_info "Docker System Usage:"
        echo "$docker_info" | sed 's/^/  /'
    fi
}

# ============================================================================
# Network Connectivity Functions
# ============================================================================

# Check network connectivity
check_network_connectivity() {
    log_section "Network Connectivity Check"
    
    # Check localhost connectivity
    if ping -c 1 localhost >/dev/null 2>&1; then
        log_success "Localhost connectivity: OK"
    else
        log_error "Localhost connectivity: FAILED"
        return 1
    fi
    
    # Check Docker daemon
    if docker info >/dev/null 2>&1; then
        log_success "Docker daemon: Running"
    else
        log_error "Docker daemon: Not running"
        return 1
    fi
}

# ============================================================================
# Main Pre-Flight Check Function
# ============================================================================

main() {
    log_highlight "🚀 Telar Pre-Flight Check"
    log_highlight "=========================="
    
    # Change to project root for consistent behavior
    cd "$PROJECT_ROOT"
    
    local overall_status=0
    
    # 1. Network and System Checks
    log_section "Network & System Checks"
    if ! check_network_connectivity; then
        overall_status=1
    fi
    
    check_system_resources
    
    # 2. Database Connectivity Checks
    log_section "Database Connectivity Checks"
    
    local postgres_status=0
    local mongo_status=0
    
    if ! check_postgresql; then
        postgres_status=1
        overall_status=1
    fi
    
    if ! check_mongodb; then
        mongo_status=1
        overall_status=1
    fi
    
    # 3. Service Health Checks
    log_section "Service Health Checks"
    
    local api_status=0
    local web_status=0
    local grpc_status=0
    
    if ! check_service_port "API Server" "$API_PORT" "$SERVICE_TIMEOUT"; then
        api_status=1
    else
        if ! check_http_service "API Server" "$API_PORT" "/"; then
            api_status=1
        fi
    fi
    
    if ! check_service_port "Web Server" "$WEB_PORT" "$SERVICE_TIMEOUT"; then
        web_status=1
    else
        if ! check_http_service "Web Server" "$WEB_PORT" "/"; then
            web_status=1
        fi
    fi
    
    # 4. gRPC Health Check
    if ! check_grpc_health; then
        grpc_status=1
    fi
    
    # 5. Summary Report
    log_section "Pre-Flight Check Summary"
    
    echo ""
    log_highlight "📊 Status Summary:"
    echo ""
    
    # Database status
    if [ $postgres_status -eq 0 ]; then
        log_success "✅ PostgreSQL: Ready"
    else
        log_error "❌ PostgreSQL: Not Ready"
    fi
    
    if [ $mongo_status -eq 0 ]; then
        log_success "✅ MongoDB: Ready"
    else
        log_error "❌ MongoDB: Not Ready"
    fi
    
    # Service status
    if [ $api_status -eq 0 ]; then
        log_success "✅ API Server: Ready"
    else
        log_warning "⚠️  API Server: Not Running (Expected if not started)"
    fi
    
    if [ $web_status -eq 0 ]; then
        log_success "✅ Web Server: Ready"
    else
        log_warning "⚠️  Web Server: Not Running (Expected if not started)"
    fi
    
    if [ $grpc_status -eq 0 ]; then
        log_success "✅ gRPC Service: Ready"
    else
        log_warning "⚠️  gRPC Service: Not Running (Expected if not started)"
    fi
    
    echo ""
    log_highlight "🎯 Recommendations:"
    
    if [ $postgres_status -ne 0 ] || [ $mongo_status -ne 0 ]; then
        log_warning "• Start databases: make up-dbs-dev"
    fi
    
    if [ $api_status -ne 0 ] && [ $web_status -ne 0 ]; then
        log_info "• Start servers: make run-both-bg"
    fi
    
    if [ $grpc_status -ne 0 ]; then
        log_info "• gRPC service may need to be started separately"
    fi
    
    echo ""
    log_highlight "=========================="
    
    if [ $overall_status -eq 0 ]; then
        log_success "✅ Pre-flight check completed successfully"
        log_info "System is ready for server startup"
    else
        log_warning "⚠️  Pre-flight check completed with warnings"
        log_info "Some dependencies may need attention before server startup"
    fi
    
    return $overall_status
}

# ============================================================================
# Script Entry Point
# ============================================================================

# Handle script interruption gracefully
trap 'log_warning "Script interrupted"; exit 130' INT TERM

# Run main function
main "$@"







