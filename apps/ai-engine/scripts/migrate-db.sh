#!/bin/bash

# AI Engine Database Migration Script
# Applies migrations to the ai-engine PostgreSQL database

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source common functions if available
if [[ -f "${PROJECT_ROOT}/tools/dev/lib/common.sh" ]]; then
    source "${PROJECT_ROOT}/tools/dev/lib/common.sh"
else
    # Fallback logging functions
    log_info() { echo -e "\033[0;34m[INFO]\033[0m $1"; }
    log_success() { echo -e "\033[0;32m[SUCCESS]\033[0m $1"; }
    log_warning() { echo -e "\033[1;33m[WARNING]\033[0m $1"; }
    log_error() { echo -e "\033[0;31m[ERROR]\033[0m $1"; }
fi

# Database configuration from environment variables (with defaults)
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5432}"
DB_USER="${POSTGRES_USERNAME:-postgres}"
DB_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
DB_NAME="${POSTGRES_DATABASE:-ai_engine}"

export PGPASSWORD="$DB_PASSWORD"

log_info "Applying AI Engine database migrations..."
log_info "  Host: $DB_HOST"
log_info "  Port: $DB_PORT"
log_info "  Database: $DB_NAME"
log_info "  User: $DB_USER"
echo

log_info "Waiting for database to be ready..."
USE_DOCKER_EXEC=false
for i in {1..10}; do
    if command -v psql > /dev/null 2>&1; then
        if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
            log_info "  ✓ Database is ready (via psql)"
            USE_DOCKER_EXEC=false
            break
        fi
    fi
    if docker exec telar-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
        log_info "  ✓ Database is ready (via docker exec)"
        USE_DOCKER_EXEC=true
        break
    fi
    if [[ $i -eq 10 ]]; then
        log_error "Failed to connect to database after 10 attempts. Please ensure PostgreSQL is running."
        exit 1
    fi
    sleep 1
done

# Migration files
MIGRATIONS=(
    "${SCRIPT_DIR}/../migrations/001_multitenancy.sql"
)

for migration_file in "${MIGRATIONS[@]}"; do
    if [[ ! -f "$migration_file" ]]; then
        log_warning "Migration file not found: $migration_file (skipping)"
        continue
    fi
    
    log_info "Applying: $(basename "$migration_file")"
    
    if [[ "$USE_DOCKER_EXEC" == "true" ]]; then
        if docker exec -i telar-postgres psql -U "$DB_USER" -d "$DB_NAME" <<EOF > /dev/null 2>&1; then
SET search_path TO public;
$(cat "$migration_file")
EOF
            log_info "  ✓ Success"
        else
            log_error "  ✗ Failed to apply $(basename "$migration_file")"
            log_error "  Run manually to see errors:"
            log_error "  docker exec -i telar-postgres psql -U $DB_USER -d $DB_NAME < $migration_file"
            exit 1
        fi
    else
        if (echo "SET search_path TO public;" && cat "$migration_file") | PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1; then
            log_info "  ✓ Success"
        else
            log_error "  ✗ Failed to apply $(basename "$migration_file")"
            log_error "  Run manually to see errors:"
            log_error "  psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migration_file"
            exit 1
        fi
    fi
done

log_info ""
log_success "✓ All migrations applied successfully"
