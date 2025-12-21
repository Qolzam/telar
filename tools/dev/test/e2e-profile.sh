#!/bin/bash

set -euo pipefail

# Configuration
BASE_URL="http://127.0.0.1:9099"
PROFILE_BASE="${BASE_URL}/profile"
AUTH_URL="http://127.0.0.1:9099"
AUTH_BASE="${AUTH_URL}/auth"
MAILHOG_URL="http://localhost:8025"

# Test configuration flags
DEBUG_MODE="${DEBUG_MODE:-false}"
CLEANUP_ON_SUCCESS="${CLEANUP_ON_SUCCESS:-true}"
FAIL_FAST="${FAIL_FAST:-true}"
GENERATE_CI_REPORT="${GENERATE_CI_REPORT:-false}"
CI_REPORT_PATH="${CI_REPORT_PATH:-./test-results/profile-e2e-report.json}"

# Test metrics tracking
TEST_COUNT_TOTAL=0
TEST_COUNT_PASSED=0
TEST_COUNT_FAILED=0
TEST_COUNT_SKIPPED=0
TEST_START_TIME=$(date +%s)
CRITICAL_FAILURE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Test data
TIMESTAMP=$(date +%s)
TEST_EMAIL="profiletest-${TIMESTAMP}@example.com"
TEST_PASSWORD="MyVerySecureProfilePassword123!@#\$%^&*()"
TEST_FULLNAME="Profile Test User"
TEST_SOCIAL_NAME="profiletest${TIMESTAMP}"

# Global variables
JWT_TOKEN=""
USER_ID=""
HMAC_SECRET="${HMAC_SECRET:-a-super-secret-key-for-local-dev-and-testing}"
TEST_SOCIAL_PROFILE_ID=""
TEST_PROFILE_IDS=()
DB_TYPE=""

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

log_test() {
    echo -e "${PURPLE}[TEST]${NC} $1"
}

make_request() {
    local method="$1"
    local url="$2"
    local data="$3"
    local headers="$4"
    local expected_status="$5"
    local description="$6"
    local is_critical="${7:-false}"
    
    TEST_COUNT_TOTAL=$((TEST_COUNT_TOTAL + 1))
    
    log_test "$description" >&2
    echo "  → $method $url" >&2
    
    if [[ -n "$data" && "$DEBUG_MODE" == "true" ]]; then
        echo "  → Data: $data" >&2
    elif [[ -n "$data" ]]; then
        echo "  → Data: ${data:0:100}..." >&2
    fi
    
    local response
    local status_code
    
    if [[ -n "$data" && -n "$headers" ]]; then
        local curl_cmd="curl -s -w '\n%{http_code}' --max-time 10 -X $method '$url' -H 'Content-Type: application/json'"
        
        while IFS= read -r header; do
            if [[ -n "$header" ]]; then
                curl_cmd="$curl_cmd -H '$header'"
            fi
        done <<< "$headers"
        
        curl_cmd="$curl_cmd -d '$data'"
        response=$(eval "$curl_cmd")
    elif [[ -n "$data" ]]; then
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$data")
    elif [[ -n "$headers" ]]; then
        local curl_cmd="curl -s -w '\n%{http_code}' --max-time 10 -X $method '$url'"
        
        while IFS= read -r header; do
            if [[ -n "$header" ]]; then
                curl_cmd="$curl_cmd -H '$header'"
            fi
        done <<< "$headers"
        
        response=$(eval "$curl_cmd")
    else
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url")
    fi
    
    status_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | sed '$d')
    
    if [[ "$DEBUG_MODE" == "true" ]]; then
        echo "  ← Full Response: $response_body" >&2
    fi
    
    echo "  ← Status: $status_code" >&2
    echo "  ← Response: ${response_body:0:200}..." >&2
    echo >&2
    
    if [[ "$status_code" == "$expected_status" ]]; then
        TEST_COUNT_PASSED=$((TEST_COUNT_PASSED + 1))
        log_success "$description (Status: $status_code)" >&2
        echo "$response_body"
        return 0
    else
        TEST_COUNT_FAILED=$((TEST_COUNT_FAILED + 1))
        log_error "$description - Expected $expected_status, got $status_code" >&2
        
        if [[ "$FAIL_FAST" == "true" ]] && { [[ "$is_critical" == "true" ]] || [[ "$status_code" =~ ^5 ]]; }; then
            CRITICAL_FAILURE=true
            log_error "CRITICAL FAILURE DETECTED - Aborting test suite" >&2
            cleanup_and_exit 1
        fi
        return 1
    fi
}

extract_json_field() {
    local json="$1"
    local field="$2"
    # Match both string values ("field":"value") and numeric values ("field":123)
    echo "$json" | grep -o "\"$field\":[^,}]*" | sed 's/.*://; s/[",]//g' | head -n1 || echo ""
}

generate_hmac_signature() {
    local method="$1"
    local path="$2"
    local query="$3"
    local body="$4"
    local uid="$5"
    local timestamp="$6"
    
    local body_hash=$(echo -n "$body" | openssl dgst -sha256 -hex | cut -d' ' -f2)
    
    local canonical=$(printf "%s\n%s\n%s\n%s\n%s\n%s" "$method" "$path" "$query" "$body_hash" "$uid" "$timestamp")
    
    local signature=$(echo -n "$canonical" | openssl dgst -sha256 -hmac "$HMAC_SECRET" -hex | cut -d' ' -f2)
    
    echo "sha256=$signature"
}

build_hmac_headers() {
    local method="$1"
    local path="$2"
    local query="${3:-}"
    local body="${4:-}"
    local uid="${USER_ID}"
    local timestamp=$(date +%s)
    
    local signature=$(generate_hmac_signature "$method" "$path" "$query" "$body" "$uid" "$timestamp")
    
    echo "X-Telar-Signature: $signature"
    echo "uid: $uid"
    echo "X-Timestamp: $timestamp"
}

validate_uuid_format() {
    local value="$1"
    local field_name="$2"
    
    if [[ ! "$value" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]; then
        log_error "Invalid UUID format for $field_name: $value"
        return 1
    fi
    return 0
}

validate_profile_response() {
    local response="$1"
    local context="${2:-profile}"
    
    local object_id=$(extract_json_field "$response" "objectId")
    local full_name=$(extract_json_field "$response" "fullName")
    local created_date=$(extract_json_field "$response" "createdDate")
    
    if [[ -z "$object_id" ]]; then
        log_error "Missing objectId in $context response"
        return 1
    fi
    
    if [[ -z "$full_name" ]]; then
        log_error "Missing fullName in $context response"
        return 1
    fi
    
    validate_uuid_format "$object_id" "objectId" || return 1
    
    if [[ -n "$created_date" ]] && [[ ! "$created_date" =~ ^[0-9]+$ ]]; then
        log_error "Invalid createdDate format in $context response: $created_date"
        return 1
    fi
    
    log_success "✓ $context response validation passed"
    return 0
}

validate_profile_array_response() {
    local response="$1"
    
    if [[ ! "$response" =~ ^\[ ]]; then
        log_error "Expected array response, got: ${response:0:50}"
        return 1
    fi
    
    log_success "✓ Profile array response validation passed"
    return 0
}

get_latest_email_for_recipient() {
    local email="$1"
    local encoded_email=$(echo "$email" | sed 's/@/%40/g')
    
    local response=$(curl -s --max-time 5 "${MAILHOG_URL}/api/v2/search?kind=to&query=${encoded_email}" 2>/dev/null || echo "{}")
    echo "$response"
}

extract_email_body() {
    local mailhog_response="$1"
    if command -v python3 >/dev/null 2>&1; then
        echo "$mailhog_response" | python3 -c "import sys, json; data=json.load(sys.stdin); items=data.get('items', []); print(items[0]['Content']['Body'] if items else '')" 2>/dev/null || echo ""
    else
        echo "$mailhog_response" | grep -oP '"Body":"[^\"]*(?<!\\)"' | head -1 | sed 's/"Body":"\(.*\)"/\1/' | sed 's/\\n/ /g' | sed 's/\\r//g' | sed 's/\\t/ /g' | sed 's/\\"/"/g'
    fi
}

extract_verification_code() {
    local email_body="$1"
    local code=$(echo "$email_body" | grep -oE 'code=[0-9]{6}' | grep -oE '[0-9]{6}' | head -1)
    if [[ -z "$code" ]]; then
        code=$(echo "$email_body" | grep -oE '(code[:\s]+|verification[:\s]+|Your code is[:\s]+)[0-9]{6}' | grep -oE '[0-9]{6}' | head -1)
    fi
    if [[ -z "$code" ]]; then
        code=$(echo "$email_body" | grep -oE '[0-9]{6}' | head -1)
    fi
    echo "$code"
}

generate_uuid() {
    if command -v uuidgen >/dev/null 2>&1; then
        uuidgen
    elif command -v python3 >/dev/null 2>&1; then
        python3 -c "import uuid; print(str(uuid.uuid4()))"
    else
        cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "$(od -x /dev/urandom | head -1 | awk '{OFS="-"; print $2$3,$4,$5,$6,$7$8$9}')"
    fi
}

detect_database_type() {
    # Try to read DB_TYPE from .env file
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local env_file="${script_dir}/../../../apps/api/.env"
    
    if [[ -f "$env_file" ]]; then
        DB_TYPE=$(grep "^DB_TYPE=" "$env_file" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "unknown")
    else
        DB_TYPE="unknown (.env not found)"
    fi
    
    # Fallback to environment variable if .env not found or empty
    if [[ -z "$DB_TYPE" ]] || [[ "$DB_TYPE" == "unknown" ]]; then
        DB_TYPE="${DB_TYPE:-postgresql (default)}"
    fi
}

verify_database_connection() {
    log_info "=== Verifying Database Connection ==="
    
    local postgres_running=$(docker ps --filter "name=telar-postgres" --format "{{.Names}}" 2>/dev/null || echo "")
    
    log_info "Database Configuration:"
    log_info "  DB_TYPE (from .env): ${DB_TYPE}"
    echo
    log_info "Database Containers Status:"
    
    if [[ -n "$postgres_running" ]]; then
        log_success "  ✓ PostgreSQL container running: $postgres_running"
    else
        log_info "  ✗ PostgreSQL container not running"
    fi
    
    echo
    
    if [[ "$DB_TYPE" == "postgresql" ]] && [[ -z "$postgres_running" ]]; then
        log_warning "⚠️  DB_TYPE is 'postgresql' but PostgreSQL container is not running!"
    fi
}

cleanup_test_data() {
    if [[ "$CLEANUP_ON_SUCCESS" != "true" ]] || [[ "$CRITICAL_FAILURE" == "true" ]]; then
        log_info "Skipping cleanup (CLEANUP_ON_SUCCESS=$CLEANUP_ON_SUCCESS, CRITICAL_FAILURE=$CRITICAL_FAILURE)"
        return 0
    fi
    
    log_info "=== Cleaning Up Test Data ==="
    log_info "Test data cleanup not implemented (no delete endpoints available)"
}

cleanup_and_exit() {
    local exit_code="$1"
    cleanup_test_data
    generate_test_report
    exit "$exit_code"
}

generate_test_report() {
    local end_time=$(date +%s)
    local duration=$((end_time - TEST_START_TIME))
    
    if [[ "$GENERATE_CI_REPORT" != "true" ]]; then
        return 0
    fi
    
    mkdir -p "$(dirname "$CI_REPORT_PATH")"
    
    local success_rate=0
    if [[ $TEST_COUNT_TOTAL -gt 0 ]]; then
        success_rate=$(awk "BEGIN {printf \"%.2f\", ($TEST_COUNT_PASSED/$TEST_COUNT_TOTAL)*100}")
    fi
    
    cat > "$CI_REPORT_PATH" <<EOF
{
  "service": "profile-microservice",
  "timestamp": $(date +%s),
  "duration_seconds": $duration,
  "total_tests": $TEST_COUNT_TOTAL,
  "passed": $TEST_COUNT_PASSED,
  "failed": $TEST_COUNT_FAILED,
  "skipped": $TEST_COUNT_SKIPPED,
  "success_rate": $success_rate,
  "critical_failure": $CRITICAL_FAILURE
}
EOF
    
    log_success "CI report generated: $CI_REPORT_PATH"
}

trap 'cleanup_and_exit $?' EXIT INT TERM

test_server_health() {
    log_info "=== Testing Profile Service Health ==="
    
    if ! curl -s "$BASE_URL" > /dev/null 2>&1; then
        log_error "Profile service is not running at $BASE_URL"
        log_error "Please start the service with: make run-profile"
        exit 1
    fi
    
    log_success "Profile service is running and accessible"
}

setup_test_user() {
    log_info "=== Setting Up Test User via Auth Service ==="
    
    local signup_data="fullName=${TEST_FULLNAME}&email=${TEST_EMAIL}&newPassword=${TEST_PASSWORD}&responseType=spa&verifyType=email"
    
    local signup_response
    signup_response=$(curl -s -X POST "${AUTH_BASE}/signup" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$signup_data")
    
    VERIFICATION_ID=$(extract_json_field "$signup_response" "verificationId")
    
    if [[ -z "$VERIFICATION_ID" ]]; then
        log_error "Failed to create test user"
        return 1
    fi
    
    log_info "Waiting for verification email..."
    sleep 3
    
    local mailhog_response=$(get_latest_email_for_recipient "$TEST_EMAIL")
    local email_body=$(extract_email_body "$mailhog_response")
    local verification_code=$(extract_verification_code "$email_body")
    
    if [[ -z "$verification_code" ]]; then
        log_warning "Could not extract verification code, skipping verification"
        return 0
    fi
    
    local verification_data="verificationId=${VERIFICATION_ID}&code=${verification_code}&responseType=spa"
    
    local verify_response
    verify_response=$(curl -s -X POST "${AUTH_BASE}/signup/verify" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$verification_data")
    
    JWT_TOKEN=$(extract_json_field "$verify_response" "accessToken")
    if [[ -z "$JWT_TOKEN" ]]; then
        JWT_TOKEN=$(extract_json_field "$verify_response" "token")
    fi
    
    USER_ID=$(extract_json_field "$verify_response" "objectId")
    if [[ -z "$USER_ID" ]]; then
        USER_ID=$(extract_json_field "$verify_response" "userId")
    fi
    
    if [[ -n "$JWT_TOKEN" ]]; then
        log_success "Test user created and verified. User ID: ${USER_ID}"
        if [[ "$DEBUG_MODE" == "true" ]]; then
            log_info "DEBUG: JWT Token: ${JWT_TOKEN:0:100}..."
        fi
    else
        log_warning "User created but no JWT token received"
    fi
}

create_test_profiles() {
    log_info "=== Creating Dedicated Test Profiles ==="
    
    # Helper function to create profile without counting as test
    create_profile_setup() {
        local profile_id="$1"
        local full_name="$2"
        local social_name="$3"
        local email="$4"
        
        local profile_data="{\"objectId\":\"${profile_id}\",\"fullName\":\"${full_name}\",\"email\":\"${email}\"}"
        if [[ -n "$social_name" ]]; then
            profile_data="{\"objectId\":\"${profile_id}\",\"fullName\":\"${full_name}\",\"socialName\":\"${social_name}\",\"email\":\"${email}\"}"
        fi
        
        local hmac_headers=$(build_hmac_headers "POST" "/profile/dto" "" "$profile_data")
        local response
        local status_code
        
        # Build curl command with HMAC headers
        local curl_cmd="curl -s -w '\n%{http_code}' --max-time 10 -X POST '${PROFILE_BASE}/dto' -H 'Content-Type: application/json'"
        while IFS= read -r header; do
            if [[ -n "$header" ]]; then
                curl_cmd="$curl_cmd -H '$header'"
            fi
        done <<< "$hmac_headers"
        curl_cmd="$curl_cmd -d '$profile_data'"
        
        response=$(eval "$curl_cmd" 2>&1)
        status_code=$(echo "$response" | tail -n1)
        if [[ "$status_code" == "201" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    TEST_SOCIAL_PROFILE_ID=$(generate_uuid)
    # Use TEST_SOCIAL_NAME so the test can find it later
    if ! create_profile_setup "$TEST_SOCIAL_PROFILE_ID" "Social Test User" "$TEST_SOCIAL_NAME" "social-${TIMESTAMP}-${RANDOM}@example.com"; then
        log_warning "Failed to create social profile (may already exist)"
    fi
    
    TEST_PROFILE_IDS=()
    for i in {1..3}; do
        local profile_id=$(generate_uuid)
        TEST_PROFILE_IDS+=("$profile_id")
        # Use nanosecond precision for truly unique emails
        local unique_suffix="${TIMESTAMP}-$(date +%s%N | cut -b10-19)-${RANDOM}-${i}"
        local unique_email="batch${i}-${unique_suffix}@example.com"
        if ! create_profile_setup "$profile_id" "Batch Test User $i" "" "$unique_email"; then
            log_warning "Failed to create batch profile $i (may already exist)"
        fi
    done
    
    log_success "Created test profiles: 1 with social name + 3 for batch operations"
}

test_read_my_profile() {
    log_info "=== Testing Read My Profile ==="
    
    make_request "GET" "${PROFILE_BASE}/my" "" "" "401" "Read my profile without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "GET" "${PROFILE_BASE}/my" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read my profile with JWT" "true")
        validate_profile_response "$response" "my profile" || CRITICAL_FAILURE=true
    fi
}

test_query_profiles() {
    log_info "=== Testing Query Profiles ==="
    
    make_request "GET" "${PROFILE_BASE}/?search=test&page=1&limit=10" "" "" "401" "Query profiles without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "GET" "${PROFILE_BASE}/?search=test&page=1&limit=10" "" "Authorization: Bearer $JWT_TOKEN" "200" "Query profiles with JWT" "true")
        validate_profile_array_response "$response" || CRITICAL_FAILURE=true
        
        response=$(make_request "GET" "${PROFILE_BASE}/?page=1&limit=5" "" "Authorization: Bearer $JWT_TOKEN" "200" "Query profiles without search" "true")
        validate_profile_array_response "$response" || CRITICAL_FAILURE=true
    fi
}

test_read_profile_by_id() {
    log_info "=== Testing Read Profile by ID ==="
    
    make_request "GET" "${PROFILE_BASE}/id/${USER_ID}" "" "" "401" "Read profile by ID without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "GET" "${PROFILE_BASE}/id/${USER_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read profile by ID with JWT" "true")
        validate_profile_response "$response" "profile by ID" || CRITICAL_FAILURE=true
    fi
}

test_get_by_social_name() {
    log_info "=== Testing Get Profile by Social Name ==="
    
    make_request "GET" "${PROFILE_BASE}/social/${TEST_SOCIAL_NAME}" "" "" "401" "Get by social name without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "GET" "${PROFILE_BASE}/social/${TEST_SOCIAL_NAME}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get by social name with JWT" "true")
        validate_profile_response "$response" "profile by social name" || CRITICAL_FAILURE=true
    fi
}

test_get_profiles_by_ids() {
    log_info "=== Testing Get Profiles by IDs ==="
    
    local user_ids="[\"${USER_ID}\",\"${TEST_PROFILE_IDS[0]}\",\"${TEST_PROFILE_IDS[1]}\"]"
    
    make_request "POST" "${PROFILE_BASE}/ids" "$user_ids" "" "401" "Get profiles by IDs without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "POST" "${PROFILE_BASE}/ids" "$user_ids" "Authorization: Bearer $JWT_TOKEN" "200" "Get profiles by IDs with JWT" "true")
        validate_profile_array_response "$response" || CRITICAL_FAILURE=true
    fi
}

test_update_profile() {
    log_info "=== Testing Update Profile ==="
    
    local update_data="{\"fullName\":\"Updated Name ${TIMESTAMP}\",\"tagLine\":\"Updated tagline\"}"
    
    make_request "PUT" "${PROFILE_BASE}/" "$update_data" "" "401" "Update profile without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        make_request "PUT" "${PROFILE_BASE}/" "$update_data" "Authorization: Bearer $JWT_TOKEN" "200" "Update profile with JWT" "true"
        
        local response=$(make_request "GET" "${PROFILE_BASE}/my" "" "Authorization: Bearer $JWT_TOKEN" "200" "Verify profile update" "true")
        local updated_name=$(extract_json_field "$response" "fullName")
        if [[ "$updated_name" == "Updated Name ${TIMESTAMP}" ]]; then
            log_success "✓ Profile update verified successfully"
        else
            log_error "Profile update verification failed: expected 'Updated Name ${TIMESTAMP}', got '$updated_name'"
            CRITICAL_FAILURE=true
        fi
    fi
}

test_hmac_protected_endpoints() {
    log_info "=== Testing HMAC-Protected Endpoints ==="
    
    log_info "--- Negative Tests: Without HMAC Authentication ---"
    
    make_request "POST" "${PROFILE_BASE}/index" "" "" "401" "Init index without HMAC (should fail)" "false"
    
    local last_seen_data="{\"userId\":\"${USER_ID}\"}"
    make_request "PUT" "${PROFILE_BASE}/last-seen" "$last_seen_data" "" "401" "Update last seen without HMAC (should fail)" "false"
    
    make_request "GET" "${PROFILE_BASE}/dto/id/${USER_ID}" "" "" "401" "Read DTO profile without HMAC (should fail)" "false"
    
    local dto_data="{\"objectId\":\"${USER_ID}\",\"fullName\":\"Test User\"}"
    make_request "POST" "${PROFILE_BASE}/dto" "$dto_data" "" "401" "Create DTO profile without HMAC (should fail)" "false"
    
    make_request "POST" "${PROFILE_BASE}/dispatch" "{}" "" "401" "Dispatch profiles without HMAC (should fail)" "false"
    
    make_request "PUT" "${PROFILE_BASE}/follow/inc/1/${USER_ID}" "" "" "401" "Increase follow count without HMAC (should fail)" "false"
    
    make_request "PUT" "${PROFILE_BASE}/follower/inc/1/${USER_ID}" "" "" "401" "Increase follower count without HMAC (should fail)" "false"
    
    local user_ids="[\"${USER_ID}\"]"
    make_request "POST" "${PROFILE_BASE}/dto/ids" "$user_ids" "" "401" "Get profiles by IDs (HMAC) without HMAC (should fail)" "false"
    
    log_info "--- Positive Tests: With Valid HMAC Authentication ---"
    
    local hmac_headers=$(build_hmac_headers "POST" "/profile/index" "" "")
    make_request "POST" "${PROFILE_BASE}/index" "" "$hmac_headers" "200" "Init index with valid HMAC" "true"
    
    local last_seen_data="{\"userId\":\"${USER_ID}\"}"
    local hmac_headers=$(build_hmac_headers "PUT" "/profile/last-seen" "" "$last_seen_data")
    make_request "PUT" "${PROFILE_BASE}/last-seen" "$last_seen_data" "$hmac_headers" "200" "Update last seen with valid HMAC" "true"
    
    local hmac_headers=$(build_hmac_headers "GET" "/profile/dto/id/${USER_ID}" "" "")
    local response=$(make_request "GET" "${PROFILE_BASE}/dto/id/${USER_ID}" "" "$hmac_headers" "200" "Read DTO profile with valid HMAC" "true")
    validate_profile_response "$response" "DTO profile" || CRITICAL_FAILURE=true
    
    # Generate a truly unique profile ID using timestamp + random to avoid conflicts
    # Keep social_name under 50 chars (validation limit)
    local short_suffix="${TIMESTAMP}${RANDOM}"
    local new_profile_id=$(generate_uuid)
    local unique_email="hmactest-${short_suffix}@example.com"
    local unique_social_name="hmac${short_suffix:0:46}"  # Max 50 chars: "hmac" + up to 46 chars
    local dto_create_data="{\"objectId\":\"${new_profile_id}\",\"fullName\":\"HMAC Test User\",\"email\":\"${unique_email}\",\"socialName\":\"${unique_social_name}\"}"
    local hmac_headers=$(build_hmac_headers "POST" "/profile/dto" "" "$dto_create_data")
    make_request "POST" "${PROFILE_BASE}/dto" "$dto_create_data" "$hmac_headers" "201" "Create DTO profile with valid HMAC" "true"
    
    local hmac_headers=$(build_hmac_headers "POST" "/profile/dispatch" "" "{}")
    make_request "POST" "${PROFILE_BASE}/dispatch" "{}" "$hmac_headers" "200" "Dispatch profiles with valid HMAC" "true"
    
    local hmac_headers=$(build_hmac_headers "PUT" "/profile/follow/inc/1/${USER_ID}" "" "")
    make_request "PUT" "${PROFILE_BASE}/follow/inc/1/${USER_ID}" "" "$hmac_headers" "200" "Increase follow count with valid HMAC" "true"
    
    local hmac_headers=$(build_hmac_headers "PUT" "/profile/follower/inc/1/${USER_ID}" "" "")
    make_request "PUT" "${PROFILE_BASE}/follower/inc/1/${USER_ID}" "" "$hmac_headers" "200" "Increase follower count with valid HMAC" "true"
    
    local user_ids="[\"${USER_ID}\"]"
    local hmac_headers=$(build_hmac_headers "POST" "/profile/dto/ids" "" "$user_ids")
    local response=$(make_request "POST" "${PROFILE_BASE}/dto/ids" "$user_ids" "$hmac_headers" "200" "Get profiles by IDs with valid HMAC" "true")
    validate_profile_array_response "$response" || CRITICAL_FAILURE=true
}

test_hmac_validation_errors() {
    log_info "=== Testing HMAC Validation Edge Cases ==="
    
    make_request "POST" "${PROFILE_BASE}/index" "" "X-Telar-Signature: invalid
uid: ${USER_ID}
X-Timestamp: $(date +%s)" "401" "Invalid HMAC signature format" "false"
    
    make_request "POST" "${PROFILE_BASE}/index" "" "X-Telar-Signature: sha256=abc123" "401" "Missing uid header" "false"
    
    make_request "POST" "${PROFILE_BASE}/index" "" "uid: ${USER_ID}
X-Timestamp: $(date +%s)" "401" "Missing signature header" "false"
    
    local old_timestamp=$(($(date +%s) - 360))
    make_request "POST" "${PROFILE_BASE}/index" "" "X-Telar-Signature: sha256=invalid
uid: ${USER_ID}
X-Timestamp: ${old_timestamp}" "401" "Expired HMAC timestamp" "false"
    
    local future_timestamp=$(($(date +%s) + 120))
    make_request "POST" "${PROFILE_BASE}/index" "" "X-Telar-Signature: sha256=invalid
uid: ${USER_ID}
X-Timestamp: ${future_timestamp}" "401" "Future HMAC timestamp" "false"
    
    make_request "POST" "${PROFILE_BASE}/index" "" "X-Telar-Signature: sha256=abc123
uid: not-a-uuid
X-Timestamp: $(date +%s)" "401" "Invalid uid format" "false"
    
    local timestamp=$(date +%s)
    local tampered_sig="sha256=$(printf '%064x' 0)"
    make_request "POST" "${PROFILE_BASE}/index" "" "X-Telar-Signature: ${tampered_sig}
uid: ${USER_ID}
X-Timestamp: ${timestamp}" "401" "Tampered HMAC signature" "false"
}

test_validation_errors() {
    log_info "=== Testing Input Validation ==="
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping validation tests"
        return 0
    fi
    
    log_info "--- UUID Validation Tests ---"
    make_request "GET" "${PROFILE_BASE}/id/invalid-uuid" "" "Authorization: Bearer $JWT_TOKEN" "400" "Invalid UUID format" "false"
    make_request "GET" "${PROFILE_BASE}/id/550e8400-e29b-41d4" "" "Authorization: Bearer $JWT_TOKEN" "400" "Incomplete UUID" "false"
    make_request "GET" "${PROFILE_BASE}/id/" "" "Authorization: Bearer $JWT_TOKEN" "404" "Empty UUID" "false"
    
    log_info "--- Social Name Validation Tests ---"
    make_request "GET" "${PROFILE_BASE}/social/" "" "Authorization: Bearer $JWT_TOKEN" "404" "Empty social name" "false"
    make_request "GET" "${PROFILE_BASE}/social/user%20with%20spaces" "" "Authorization: Bearer $JWT_TOKEN" "400" "Social name with spaces" "false"
    make_request "GET" "${PROFILE_BASE}/social/$(printf 'a%.0s' {1..200})" "" "Authorization: Bearer $JWT_TOKEN" "400" "Very long social name" "false"
    
    log_info "--- Pagination Edge Cases ---"
    make_request "GET" "${PROFILE_BASE}/?page=0&limit=10" "" "Authorization: Bearer $JWT_TOKEN" "200" "page=0 (should default to 1)" "false"
    make_request "GET" "${PROFILE_BASE}/?page=-1&limit=10" "" "Authorization: Bearer $JWT_TOKEN" "200" "page=-1 (should default to 1)" "false"
    make_request "GET" "${PROFILE_BASE}/?page=1&limit=0" "" "Authorization: Bearer $JWT_TOKEN" "200" "limit=0 (should default to 10)" "false"
    make_request "GET" "${PROFILE_BASE}/?page=1&limit=-1" "" "Authorization: Bearer $JWT_TOKEN" "200" "limit=-1 (should default to 10)" "false"
    make_request "GET" "${PROFILE_BASE}/?page=abc&limit=xyz" "" "Authorization: Bearer $JWT_TOKEN" "200" "Non-numeric pagination" "false"
    make_request "GET" "${PROFILE_BASE}/?page=1&limit=10000" "" "Authorization: Bearer $JWT_TOKEN" "200" "Very large limit" "false"
    
    log_info "--- Update Validation Tests ---"
    make_request "PUT" "${PROFILE_BASE}/" "{}" "Authorization: Bearer $JWT_TOKEN" "200" "Empty update (should succeed)" "false"
    
    log_info "--- Batch Operations Validation ---"
    make_request "POST" "${PROFILE_BASE}/ids" "[]" "Authorization: Bearer $JWT_TOKEN" "200" "Empty UUID array" "false"
    make_request "POST" "${PROFILE_BASE}/ids" "[\"not-a-uuid\"]" "Authorization: Bearer $JWT_TOKEN" "400" "Invalid UUID in array" "false"
    make_request "POST" "${PROFILE_BASE}/ids" "[\"${USER_ID}\",\"invalid\"]" "Authorization: Bearer $JWT_TOKEN" "400" "Mixed valid/invalid UUIDs" "false"
    
    local large_batch="["
    for i in {1..100}; do
        large_batch+="\"$(generate_uuid)\""
        [[ $i -lt 100 ]] && large_batch+=","
    done
    large_batch+="]"
    make_request "POST" "${PROFILE_BASE}/ids" "$large_batch" "Authorization: Bearer $JWT_TOKEN" "200" "Large batch (100 UUIDs)" "false"
    
    log_info "--- JSON Parsing Validation ---"
    make_request "POST" "${PROFILE_BASE}/ids" "{invalid json}" "Authorization: Bearer $JWT_TOKEN" "400" "Malformed JSON" "false"
    make_request "PUT" "${PROFILE_BASE}/" "not-json" "Authorization: Bearer $JWT_TOKEN" "400" "Non-JSON body" "false"
}

test_profile_creation_flow() {
    log_info "=== Testing Profile Creation Integration Flow ==="
    
    if [[ -z "$USER_ID" ]]; then
        log_warning "No user ID from signup, skipping integration test"
        return 0
    fi
    
    local response1=$(make_request "GET" "${PROFILE_BASE}/my" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read my profile" "false")
    local object_id1=$(extract_json_field "$response1" "objectId")
    
    local response2=$(make_request "GET" "${PROFILE_BASE}/id/${USER_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read profile by ID" "false")
    local object_id2=$(extract_json_field "$response2" "objectId")
    
    if [[ "$object_id1" == "$object_id2" ]] && [[ "$object_id1" == "$USER_ID" ]]; then
        log_success "✓ Profile consistency verified across endpoints"
    else
        log_error "Profile inconsistency detected: my=$object_id1, byId=$object_id2, expected=$USER_ID"
        return 1
    fi
}

test_profile_update_propagation() {
    log_info "=== Testing Profile Update Propagation ==="
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping update propagation test"
        return 0
    fi
    
    local new_tagline="Integration Test Tag ${TIMESTAMP}"
    local update_data="{\"tagLine\":\"${new_tagline}\"}"
    make_request "PUT" "${PROFILE_BASE}/" "$update_data" "Authorization: Bearer $JWT_TOKEN" "200" "Update profile tagline" "false"
    
    sleep 1
    
    local response1=$(make_request "GET" "${PROFILE_BASE}/my" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read my profile after update" "false")
    local tagline1=$(extract_json_field "$response1" "tagLine")
    
    local response2=$(make_request "GET" "${PROFILE_BASE}/id/${USER_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read by ID after update" "false")
    local tagline2=$(extract_json_field "$response2" "tagLine")
    
    local hmac_headers=$(build_hmac_headers "GET" "/profile/dto/id/${USER_ID}" "" "")
    local response3=$(make_request "GET" "${PROFILE_BASE}/dto/id/${USER_ID}" "" "$hmac_headers" "200" "Read DTO after update" "false")
    local tagline3=$(extract_json_field "$response3" "tagLine")
    
    if [[ "$tagline1" == "$new_tagline" ]] && [[ "$tagline2" == "$new_tagline" ]] && [[ "$tagline3" == "$new_tagline" ]]; then
        log_success "✓ Profile update propagated across all endpoints"
    else
        log_error "Update propagation failed: my='$tagline1', byId='$tagline2', dto='$tagline3', expected='$new_tagline'"
        return 1
    fi
}

test_last_seen_update_flow() {
    log_info "=== Testing Last Seen Update Flow ==="
    
    local last_seen_data="{\"userId\":\"${USER_ID}\"}"
    local hmac_headers=$(build_hmac_headers "PUT" "/profile/last-seen" "" "$last_seen_data")
    make_request "PUT" "${PROFILE_BASE}/last-seen" "$last_seen_data" "$hmac_headers" "200" "Update last seen" "false"
    
    sleep 1
    
    local response=$(make_request "GET" "${PROFILE_BASE}/my" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read profile after last-seen update" "false")
    local last_seen=$(extract_json_field "$response" "lastSeen")
    
    if [[ -n "$last_seen" ]] && [[ "$last_seen" =~ ^[0-9]+$ ]]; then
        log_success "✓ Last seen updated successfully: $last_seen"
    else
        log_warning "Last seen field not found or invalid format: '$last_seen'"
    fi
}

test_follow_follower_consistency() {
    log_info "=== Testing Follow/Follower Count Consistency ==="
    
    local response1=$(make_request "GET" "${PROFILE_BASE}/my" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read initial counts" "false")
    local initial_follow=$(extract_json_field "$response1" "followCount")
    local initial_follower=$(extract_json_field "$response1" "followerCount")
    initial_follow=${initial_follow:-0}
    initial_follower=${initial_follower:-0}
    
    local hmac_headers=$(build_hmac_headers "PUT" "/profile/follow/inc/5/${USER_ID}" "" "")
    make_request "PUT" "${PROFILE_BASE}/follow/inc/5/${USER_ID}" "" "$hmac_headers" "200" "Increment follow count by 5" "false"
    
    local hmac_headers=$(build_hmac_headers "PUT" "/profile/follower/inc/3/${USER_ID}" "" "")
    make_request "PUT" "${PROFILE_BASE}/follower/inc/3/${USER_ID}" "" "$hmac_headers" "200" "Increment follower count by 3" "false"
    
    sleep 1
    
    local response2=$(make_request "GET" "${PROFILE_BASE}/my" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read updated counts" "false")
    local new_follow=$(extract_json_field "$response2" "followCount")
    local new_follower=$(extract_json_field "$response2" "followerCount")
    new_follow=${new_follow:-0}
    new_follower=${new_follower:-0}
    
    local expected_follow=$((initial_follow + 5))
    local expected_follower=$((initial_follower + 3))
    
    if [[ "$new_follow" -eq "$expected_follow" ]] && [[ "$new_follower" -eq "$expected_follower" ]]; then
        log_success "✓ Follow/follower counts consistent: follow=$new_follow (expected $expected_follow), follower=$new_follower (expected $expected_follower)"
    else
        log_error "Count mismatch: follow=$new_follow (expected $expected_follow), follower=$new_follower (expected $expected_follower)"
        return 1
    fi
}

test_batch_operations_consistency() {
    log_info "=== Testing Batch Operations Consistency ==="
    
    local user_ids="[\"${USER_ID}\",\"${TEST_PROFILE_IDS[0]}\",\"${TEST_PROFILE_IDS[1]}\"]"
    local response1=$(make_request "POST" "${PROFILE_BASE}/ids" "$user_ids" "Authorization: Bearer $JWT_TOKEN" "200" "Get profiles via JWT" "false")
    
    local hmac_headers=$(build_hmac_headers "POST" "/profile/dto/ids" "" "$user_ids")
    local response2=$(make_request "POST" "${PROFILE_BASE}/dto/ids" "$user_ids" "$hmac_headers" "200" "Get profiles via HMAC" "false")
    
    validate_profile_array_response "$response1" || return 1
    validate_profile_array_response "$response2" || return 1
    
    log_success "✓ Batch operations consistent across authentication methods"
}

test_dto_profile_synchronization() {
    log_info "=== Testing DTO Profile Synchronization ==="
    
    local dto_profile_id=$(generate_uuid)
    local dto_data="{\"objectId\":\"${dto_profile_id}\",\"fullName\":\"DTO Sync Test\",\"email\":\"dtosync-${TIMESTAMP}@example.com\",\"socialName\":\"dtosync${TIMESTAMP}\"}"
    local hmac_headers=$(build_hmac_headers "POST" "/profile/dto" "" "$dto_data")
    make_request "POST" "${PROFILE_BASE}/dto" "$dto_data" "$hmac_headers" "201" "Create profile via DTO" "false"
    
    sleep 2
    
    local response=$(make_request "GET" "${PROFILE_BASE}/id/${dto_profile_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read DTO-created profile via user endpoint" "false")
    
    if [[ $? -eq 0 ]]; then
        validate_profile_response "$response" "DTO-synchronized profile" || return 1
        log_success "✓ DTO profile synchronized and accessible"
    else
        log_warning "DTO profile not accessible via user endpoints (may be expected behavior)"
    fi
}

main() {
    log_info "========================================"
    log_info "Profile Microservice E2E Testing Suite"
    log_info "========================================"
    log_info "Profile Service URL: $BASE_URL"
    log_info "Auth Service URL: $AUTH_URL"
    log_info "Debug Mode: $DEBUG_MODE"
    log_info "Fail Fast: $FAIL_FAST"
    log_info "CI Report: $GENERATE_CI_REPORT"
    echo
    
    detect_database_type
    verify_database_connection
    
    test_server_health
    echo
    
    log_info "=== PHASE 1: Test Setup ==="
    if ! curl -s "$AUTH_URL" > /dev/null 2>&1; then
        log_warning "Auth service not running at $AUTH_URL"
        log_warning "Some tests will be skipped"
    else
        setup_test_user
        echo
        create_test_profiles
        echo
    fi
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_error "CRITICAL: Failed to obtain JWT token - cannot proceed with authenticated tests"
        CRITICAL_FAILURE=true
        cleanup_and_exit 1
    fi
    
    log_info "=== PHASE 2: User-Facing Endpoints (JWT/Cookie Auth) ==="
    test_read_my_profile
    echo
    test_query_profiles
    echo
    test_read_profile_by_id
    echo
    test_get_by_social_name
    echo
    test_get_profiles_by_ids
    echo
    test_update_profile
    echo
    
    log_info "=== PHASE 3: Service-to-Service Endpoints (HMAC Auth) ==="
    test_hmac_protected_endpoints
    echo
    test_hmac_validation_errors
    echo
    
    log_info "=== PHASE 4: Input Validation & Edge Cases ==="
    test_validation_errors
    echo
    
    log_info "=== PHASE 5: Integration Tests ==="
    test_profile_creation_flow
    echo
    test_profile_update_propagation
    echo
    test_last_seen_update_flow
    echo
    test_follow_follower_consistency
    echo
    test_batch_operations_consistency
    echo
    test_dto_profile_synchronization
    echo
    
    local end_time=$(date +%s)
    local duration=$((end_time - TEST_START_TIME))
    
    log_info "========================================"
    log_success "All Profile Microservice tests completed!"
    log_info "========================================"
    log_info "Test Summary:"
    log_info "  Total Tests:    $TEST_COUNT_TOTAL"
    if [[ $TEST_COUNT_TOTAL -gt 0 ]]; then
        log_info "  Passed:         $TEST_COUNT_PASSED ($(awk "BEGIN {printf \"%.1f\", ($TEST_COUNT_PASSED/$TEST_COUNT_TOTAL)*100}")%)"
    else
        log_info "  Passed:         $TEST_COUNT_PASSED (0.0%)"
    fi
    log_info "  Failed:         $TEST_COUNT_FAILED"
    log_info "  Skipped:        $TEST_COUNT_SKIPPED"
    log_info "  Duration:       ${duration}s"
    log_info "  Critical Issues: $([ "$CRITICAL_FAILURE" == "true" ] && echo "YES" || echo "NO")"
    echo
    
    if [[ "$CRITICAL_FAILURE" == "true" ]]; then
        log_error "❌ Test suite FAILED with critical issues"
        exit 1
    elif [[ $TEST_COUNT_FAILED -gt 0 ]]; then
        log_warning "⚠️  Test suite completed with $TEST_COUNT_FAILED failures"
        exit 1
    else
    log_success "🎉 Profile Microservice is fully functional!"
        exit 0
    fi
}

main "$@"
