#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

API_DIR="${SCRIPT_DIR}/../../../apps/api"
ENV_FILE="${API_DIR}/.env"

log_info() {
    echo -e "${GREEN}[MIGRATE]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

if [[ ! -f "$ENV_FILE" ]]; then
    log_error ".env file not found at $ENV_FILE"
    exit 1
fi

DB_HOST=$(grep -E "^POSTGRES_HOST=|^DB_HOST=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "localhost")
DB_PORT=$(grep -E "^POSTGRES_PORT=|^DB_PORT=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "5432")
DB_USER=$(grep -E "^POSTGRES_USERNAME=|^POSTGRES_USER=|^DB_USER=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "postgres")
DB_PASSWORD=$(grep -E "^POSTGRES_PASSWORD=|^DB_PASSWORD=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "postgres")
DB_NAME=$(grep -E "^POSTGRES_DATABASE=|^POSTGRES_DB=|^DB_NAME=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "telar_social_test")

export PGPASSWORD="$DB_PASSWORD"

log_info "Applying database migrations..."
log_info "  Host: $DB_HOST"
log_info "  Port: $DB_PORT"
log_info "  Database: $DB_NAME"
log_info "  User: $DB_USER"
echo

log_info "Waiting for database to be ready..."
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

MIGRATIONS=(
    "${API_DIR}/posts/migrations/001_create_posts_table.sql"
    "${API_DIR}/posts/migrations/002_add_search_index.sql"
    "${API_DIR}/auth/migrations/003_create_auth_tables.sql"
    "${API_DIR}/profile/migrations/002_create_profiles_table.sql"
    "${API_DIR}/profile/migrations/003_add_search_index.sql"
    "${API_DIR}/auth/migrations/004_create_admin_tables.sql"
    "${API_DIR}/comments/migrations/005_create_comments_table.sql"
    "${API_DIR}/votes/migrations/006_create_votes_table.sql"
    "${API_DIR}/comments/migrations/007_create_comment_votes.sql"
    "${API_DIR}/comments/migrations/008_add_reply_to_user.sql"
    "${API_DIR}/bookmarks/migrations/001_create_bookmarks_table.sql"
    "${API_DIR}/storage/migrations/001_create_storage_tables.sql"
    "${API_DIR}/storage/migrations/002_add_usage_tracking.sql"
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
log_info "✓ All migrations applied successfully"

