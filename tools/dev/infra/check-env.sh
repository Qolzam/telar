#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

POSTGRES_HOST="localhost"
POSTGRES_PORT="5432"
POSTGRES_DB="telar_social_test"
POSTGRES_USER="postgres"
POSTGRES_PASSWORD="postgres"

API_PORT="9099"
WEB_PORT="3000"
GRPC_PORT="9090"

DB_TIMEOUT=10
SERVICE_TIMEOUT=5
GRPC_TIMEOUT=5

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_highlight() {
    echo -e "${CYAN}[HIGHLIGHT]${NC} $1"
}

log_section() {
    echo -e "\033[0;35m[SECTION]\033[0m $1"
}

check_postgresql() {
    log_info "Checking PostgreSQL connectivity..."
    
    if ! docker ps --format "table {{.Names}}" | grep -q "telar-postgres"; then
        log_error "PostgreSQL container is not running"
        return 1
    fi
    
    if ! nc -z "$POSTGRES_HOST" "$POSTGRES_PORT" 2>/dev/null; then
        log_error "PostgreSQL port $POSTGRES_PORT is not accessible"
        return 1
    fi
    
    if command -v psql >/dev/null 2>&1; then
        if PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;" >/dev/null 2>&1; then
            log_success "PostgreSQL connection successful"
            local pg_version
            pg_version=$(PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT version();" 2>/dev/null | head -n1 | xargs)
            log_info "PostgreSQL Version: $pg_version"
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

check_grpc_health() {
    log_info "Checking gRPC service health..."
    
    if ! nc -z "localhost" "$GRPC_PORT" 2>/dev/null; then
        log_warning "gRPC port $GRPC_PORT is not accessible"
        return 1
    fi
    
    if command -v grpcurl >/dev/null 2>&1; then
        log_info "Using grpcurl for gRPC health check..."
        if grpcurl -plaintext "localhost:$GRPC_PORT" list >/dev/null 2>&1; then
            log_success "gRPC service is responding"
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

check_system_resources() {
    log_section "System Resources Check"
    local available_memory
    available_memory=$(free -h | awk '/^Mem:/ {print $7}')
    log_info "Available Memory: $available_memory"
    local disk_usage
    disk_usage=$(df -h . | awk 'NR==2 {print $5}')
    log_info "Disk Usage: $disk_usage"
    if command -v docker >/dev/null 2>&1; then
        local docker_info
        docker_info=$(docker system df --format "table {{.Type}}\t{{.TotalCount}}\t{{.Size}}" 2>/dev/null | head -3)
        log_info "Docker System Usage:"
        echo "$docker_info" | sed 's/^/  /'
    fi
}

check_network_connectivity() {
    log_section "Network Connectivity Check"
    if ping -c 1 localhost >/dev/null 2>&1; then
        log_success "Localhost connectivity: OK"
    else
        log_error "Localhost connectivity: FAILED"
        return 1
    fi
    if docker info >/dev/null 2>&1; then
        log_success "Docker daemon: Running"
    else
        log_error "Docker daemon: Not running"
        return 1
    fi
}

main() {
    log_highlight "🚀 Telar Pre-Flight Check"
    log_highlight "=========================="
    
    cd "$PROJECT_ROOT"
    
    local overall_status=0
    
    log_section "Network & System Checks"
    if ! check_network_connectivity; then
        overall_status=1
    fi
    check_system_resources
    
    log_section "Database Connectivity Checks"
    local postgres_status=0
    if ! check_postgresql; then
        postgres_status=1
        overall_status=1
    fi
    
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
    
    if ! check_grpc_health; then
        grpc_status=1
    fi
    
    log_section "Pre-Flight Check Summary"
    echo ""
    log_highlight "📊 Status Summary:"
    echo ""
    
    if [ $postgres_status -eq 0 ]; then
        log_success "✅ PostgreSQL: Ready"
    else
        log_error "❌ PostgreSQL: Not Ready"
    fi
    
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
    
    if [ $postgres_status -ne 0 ]; then
        log_warning "• Start PostgreSQL: make up-dbs-dev"
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

trap 'log_warning "Script interrupted"; exit 130' INT TERM

main "$@"

