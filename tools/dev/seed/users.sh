#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

COUNT=${1:-5}
TEST_PASSWORD="LifecycleTestPassword123!@#"
TOKENS_FILE="test_tokens.txt"
USERS_FILE="test_users.json"

check_database_schema() {
    log_info "Verifying database schema..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    API_DIR="${SCRIPT_DIR}/../../../apps/api"
    ENV_FILE="${API_DIR}/.env"
    if [[ ! -f "$ENV_FILE" ]]; then
        log_error ".env file not found at $ENV_FILE"
        return 1
    fi
    DB_HOST=$(grep -E "^POSTGRES_HOST=|^DB_HOST=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "localhost")
    DB_PORT=$(grep -E "^POSTGRES_PORT=|^DB_PORT=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "5432")
    DB_USER=$(grep -E "^POSTGRES_USERNAME=|^POSTGRES_USER=|^DB_USER=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "postgres")
    DB_PASSWORD=$(grep -E "^POSTGRES_PASSWORD=|^DB_PASSWORD=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "postgres")
    DB_NAME=$(grep -E "^POSTGRES_DATABASE=|^POSTGRES_DB=|^DB_NAME=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "telar_social_test")
    local missing_tables=()
    if docker exec telar-postgres psql -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='verifications'" 2>/dev/null | grep -q 1; then
        log_success "✓ verifications table exists"
    else
        log_error "✗ verifications table missing"
        missing_tables+=("verifications")
    fi
    if docker exec telar-postgres psql -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='user_auths'" 2>/dev/null | grep -q 1; then
        log_success "✓ user_auths table exists"
    else
        log_error "✗ user_auths table missing"
        missing_tables+=("user_auths")
    fi
    if docker exec telar-postgres psql -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='profiles'" 2>/dev/null | grep -q 1; then
        log_success "✓ profiles table exists"
    else
        log_error "✗ profiles table missing"
        missing_tables+=("profiles")
    fi
    if [[ ${#missing_tables[@]} -gt 0 ]]; then
        log_error ""
        log_error "Database schema is incomplete. Missing tables: ${missing_tables[*]}"
        log_warn "Run migrations before seeding users:"
        log_warn "  bash tools/dev/infra/db-migrate.sh"
        log_warn ""
        log_warn "Or run migrations automatically now? (y/n)"
        read -r -t 10 response || response="n"
        if [[ "$response" =~ ^[Yy]$ ]]; then
            log_info "Running migrations..."
            if bash "${SCRIPT_DIR}/../infra/db-migrate.sh"; then
                log_success "✓ Migrations applied successfully"
                return 0
            else
                log_error "✗ Migration failed"
                return 1
            fi
        else
            log_error "Cannot proceed without database schema. Exiting."
            return 1
        fi
    fi
    log_success "✓ Database schema verified"
    return 0
}

make_request() {
    local method="$1"
    local url="$2"
    local data="$3"
    local token="$4"
    local expected_status="$5"
    local description="$6"
    if [[ -n "$description" ]]; then
        log_info "$description" >&2
    fi
    local response
    local status_code
    if [[ -n "$data" ]]; then
        if [[ "$data" == *"="* && "$data" != *"{"* ]]; then
            response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
                -H "Content-Type: application/x-www-form-urlencoded" \
                ${token:+-H "Authorization: Bearer $token"} \
                -d "$data" 2>/dev/null)
        else
            response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
                -H "Content-Type: application/json" \
                ${token:+-H "Authorization: Bearer $token"} \
                -d "$data" 2>/dev/null)
        fi
    else
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
            -H "Content-Type: application/json" \
            ${token:+-H "Authorization: Bearer $token"} 2>/dev/null)
    fi
    status_code=$(echo "$response" | tail -n1)
    response=$(echo "$response" | sed '$d')
    if [[ -z "$status_code" || "$status_code" == "000" ]]; then
        log_error "Request failed - no response from server" >&2
        return 1
    fi
    if [[ "$status_code" != "$expected_status" ]]; then
        log_error "Expected status $expected_status, got $status_code" >&2
        echo "Response: $response" >&2
        return 1
    fi
    log_success "✓ $description" >&2
    echo "$response"
    return 0
}

extract_json_field() {
    local json="$1"
    local field="$2"
    echo "$json" | grep -o "\"$field\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | sed "s/\"$field\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\"/\1/" | head -1
}

get_verification_code() {
    local email="$1"
    local encoded_email=$(echo "$email" | sed 's/@/%40/g')
    local MAX_RETRIES=20
    local SLEEP_TIME=2
    local code=""
    for i in $(seq 1 $MAX_RETRIES); do
        local mailhog_response=$(curl -s --max-time 5 "${MAILHOG_URL}/api/v2/search?kind=to&query=${encoded_email}" 2>/dev/null || echo "{}")
        if command -v python3 >/dev/null 2>&1; then
            local email_body=$(echo "$mailhog_response" | python3 -c "import sys, json; data=json.load(sys.stdin); items=data.get('items', []); print(items[0]['Content']['Body'] if items else '')" 2>/dev/null || echo "")
            code=$(echo "$email_body" | grep -oE 'code=[0-9]{6}' | grep -oE '[0-9]{6}' | head -1)
            if [[ -z "$code" ]]; then
                code=$(echo "$email_body" | grep -oE '(code[:\s]+|verification[:\s]+|Your code is[:\s]+)[0-9]{6}' | grep -oE '[0-9]{6}' | head -1)
            fi
            if [[ -z "$code" ]]; then
                code=$(echo "$email_body" | grep -oE '[0-9]{6}' | head -1)
            fi
        else
            code=$(echo "$mailhog_response" | grep -oE '[0-9]{6}' | head -1)
        fi
        if [[ -n "$code" && ${#code} -eq 6 ]]; then
            echo "$code"
            return 0
        fi
        if [[ $i -lt $MAX_RETRIES ]]; then
            sleep $SLEEP_TIME
        fi
    done
    log_error "Verification code never arrived for $email" >&2
    return 1
}

create_user() {
    local index="$1"
    local timestamp=$(date +%s)
    local email="user_${timestamp}_${index}@example.com"
    local fullname="Test User ${index}"
    log_info "Creating user $index: $email"
    local signup_data="fullName=${fullname}&email=${email}&newPassword=${TEST_PASSWORD}&responseType=spa&verifyType=email&g-recaptcha-response=ok"
    local signup_response=$(make_request "POST" "${AUTH_BASE}/signup" "$signup_data" "" "200" "Signup user: $email")
    if [[ $? -ne 0 ]]; then
        log_error "Signup failed for $email"
        return 1
    fi
    local verification_id=$(extract_json_field "$signup_response" "verificationId")
    if [[ -z "$verification_id" ]]; then
        log_error "Failed to get verification ID for $email"
        return 1
    fi
    log_info "Waiting for verification email..."
    sleep 5
    local code=""
    local retries=0
    while [[ -z "$code" && $retries -lt 10 ]]; do
        code=$(get_verification_code "$email")
        if [[ -z "$code" ]]; then
            retries=$((retries + 1))
            sleep 3
        fi
    done
    if [[ -z "$code" ]]; then
        log_error "Failed to get verification code for $email"
        return 1
    fi
    log_info "Verifying email with code: $code"
    local verify_data="verificationId=${verification_id}&code=${code}&responseType=spa"
    local verify_response=$(make_request "POST" "${AUTH_BASE}/signup/verify" "$verify_data" "" "200" "Verify email: $email")
    if [[ $? -ne 0 ]]; then
        log_error "Verification failed for $email"
        return 1
    fi
    local token=$(extract_json_field "$verify_response" "accessToken")
    if [[ -z "$token" ]]; then
        token=$(extract_json_field "$verify_response" "token")
    fi
    if [[ -z "$token" ]]; then
        log_error "Failed to get access token for $email"
        return 1
    fi
    echo "$token" >> "$TOKENS_FILE"
    echo "{\"email\":\"$email\",\"password\":\"$TEST_PASSWORD\",\"fullname\":\"$fullname\",\"token\":\"$token\"}" >> "$USERS_FILE"
    CREATED_USERS+=("$index|$email|$TEST_PASSWORD|$fullname")
    log_success "✓ User $index created successfully"
    echo "   Username: $email"
    echo "   Password: $TEST_PASSWORD"
    echo "   Full Name: $fullname"
    echo "   Token: ${token:0:50}..."
    return 0
}

main() {
    log_info "=========================================="
    log_info "Seeding $COUNT users"
    log_info "=========================================="
    echo ""
    if ! check_database_schema; then
        log_error "Database schema check failed. Cannot proceed with seeding."
        exit 1
    fi
    echo ""
    rm -f "$TOKENS_FILE" "$USERS_FILE"
    touch "$TOKENS_FILE" "$USERS_FILE"
    CREATED_USERS=()
    local success_count=0
    for i in $(seq 1 $COUNT); do
        if create_user "$i"; then
            success_count=$((success_count + 1))
        fi
        echo ""
    done
    log_info "=========================================="
    log_info "Seeding complete: $success_count/$COUNT users created"
    log_info "=========================================="
    log_info "Tokens saved to: $TOKENS_FILE"
    log_info "User details saved to: $USERS_FILE"
    echo ""
    if [[ ${#CREATED_USERS[@]} -gt 0 ]]; then
        log_info "Created Users Summary:"
        echo ""
        printf "%-6s %-40s %-35s %-20s\n" "Index" "Email (Username)" "Password" "Full Name"
        printf "%-6s %-40s %-35s %-20s\n" "-----" "----------------------------------------" "-----------------------------------" "--------------------"
        for user_info in "${CREATED_USERS[@]}"; do
            IFS='|' read -r index email password fullname <<< "$user_info"
            printf "%-6s %-40s %-35s %-20s\n" "$index" "$email" "$password" "$fullname"
        done
        echo ""
    fi
    if [[ $success_count -eq $COUNT ]]; then
        log_success "All users created successfully!"
        return 0
    else
        log_error "Some users failed to create"
        return 1
    fi
}

main "$@"

