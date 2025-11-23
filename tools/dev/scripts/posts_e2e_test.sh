#!/bin/bash

set -euo pipefail

# Configuration
BASE_URL="http://127.0.0.1:8080"
POSTS_BASE="${BASE_URL}/posts"
COMMENTS_BASE="${BASE_URL}/comments"
AUTH_URL="http://127.0.0.1:8080"
AUTH_BASE="${AUTH_URL}/auth"
MAILHOG_URL="http://localhost:8025"

# Test configuration flags
DEBUG_MODE="${DEBUG_MODE:-false}"
CLEANUP_ON_SUCCESS="${CLEANUP_ON_SUCCESS:-true}"
FAIL_FAST="${FAIL_FAST:-true}"
GENERATE_CI_REPORT="${GENERATE_CI_REPORT:-false}"
CI_REPORT_PATH="${CI_REPORT_PATH:-./test-results/posts-e2e-report.json}"

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

# Communication mode configuration
DEPLOYMENT_MODE="${DEPLOYMENT_MODE:-serverless}"  # serverless or microservices
COMMENTS_SERVICE_GRPC_ADDR="${COMMENTS_SERVICE_GRPC_ADDR:-localhost:50052}"
POSTS_SERVICE_GRPC_ADDR="${POSTS_SERVICE_GRPC_ADDR:-localhost:50053}"

# Test data
TIMESTAMP=$(date +%s)
TEST_EMAIL="poststest-${TIMESTAMP}@example.com"
TEST_PASSWORD="MyVerySecurePostsPassword123!@#\$%^&*()"
TEST_FULLNAME="Posts Test User"

# Global variables
JWT_TOKEN=""
USER_ID=""
HMAC_SECRET="${HMAC_SECRET:-a-super-secret-key-for-local-dev-and-testing}"
TEST_POST_IDS=()
TEST_URL_KEYS=()
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

validate_post_response() {
    local response="$1"
    local context="${2:-post}"
    
    local object_id=$(extract_json_field "$response" "objectId")
    local body=$(extract_json_field "$response" "body")
    local created_date=$(extract_json_field "$response" "createdDate")
    
    if [[ -z "$object_id" ]]; then
        log_error "Missing objectId in $context response"
        return 1
    fi
    
    if [[ -z "$body" ]]; then
        log_error "Missing body in $context response"
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

validate_posts_array_response() {
    local response="$1"
    
    if [[ ! "$response" =~ ^\{ ]]; then
        log_error "Expected object response with posts array, got: ${response:0:50}"
        return 1
    fi
    
    log_success "✓ Posts array response validation passed"
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
    # Try using python to parse JSON properly (MailHog v2 has nested structure)
    if command -v python3 >/dev/null 2>&1; then
        echo "$mailhog_response" | python3 -c "import sys, json; data=json.load(sys.stdin); items=data.get('items', []); print(items[0]['Content']['Body'] if items else '')" 2>/dev/null || echo ""
    else
        # Fallback: try to extract from nested JSON structure
        echo "$mailhog_response" | grep -oP '"Body":"[^"]*(?<!\\)"' | head -1 | sed 's/"Body":"\(.*\)"/\1/' | sed 's/\\n/ /g' | sed 's/\\r//g' | sed 's/\\t/ /g' | sed 's/\\"/"/g'
    fi
}

extract_verification_code() {
    local email_body="$1"
    # First, try to extract from URL parameter: code=123456
    local code=$(echo "$email_body" | grep -oE 'code=[0-9]{6}' | grep -oE '[0-9]{6}' | head -1)
    if [[ -z "$code" ]]; then
        # Try to find verification code in common patterns
        # Look for "code:" or "verification" followed by 6 digits
        code=$(echo "$email_body" | grep -oE '(code[:\s]+|verification[:\s]+|Your code is[:\s]+)[0-9]{6}' | grep -oE '[0-9]{6}' | head -1)
    fi
    if [[ -z "$code" ]]; then
        # Fallback: find 6-digit codes that are likely verification codes (not dates/IDs)
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
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local env_file="${script_dir}/../../../apps/api/.env"
    
    if [[ -f "$env_file" ]]; then
        DB_TYPE=$(grep "^DB_TYPE=" "$env_file" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "unknown")
    else
        DB_TYPE="unknown (.env not found)"
    fi
    
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
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token available for cleanup"
        return 0
    fi
    
    for post_id in "${TEST_POST_IDS[@]}"; do
        if [[ -n "$post_id" ]]; then
            log_info "Soft deleting test post: $post_id"
            # TEMPORARY: Remove output redirection to see actual responses for debugging
            # TODO: Re-enable redirection after root cause is identified
            if [[ "${DEBUG_CLEANUP:-false}" == "true" ]]; then
                make_request "DELETE" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "204" "Cleanup: Delete test post" "false" || log_warning "Failed to delete post $post_id"
            else
                make_request "DELETE" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "204" "Cleanup: Delete test post" "false" > /dev/null 2>&1 || log_warning "Failed to delete post $post_id"
            fi
        fi
    done
    
    log_success "Test data cleanup completed"
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
  "service": "posts-microservice",
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

# Waits for a service to be healthy and optionally checks its version/build hash.
wait_for_service() {
    local service_url="$1"
    local service_name="$2"
    log_info "Waiting for $service_name service at $service_url to become healthy..."
    
    for i in {1..15}; do
        # Try /health endpoint first
        if curl -s -f "${service_url}/health" > /dev/null 2>&1; then
            log_success "✓ $service_name service is healthy."
            return 0
        fi
        # Try root path - any HTTP response (even 404) means server is running
        if curl -s "${service_url}" > /dev/null 2>&1; then
            log_success "✓ $service_name service is accessible (no /health endpoint)."
            return 0
        fi
        # For Posts service, try /posts endpoint
        if [[ "$service_name" == "Posts" ]]; then
            if curl -s "${service_url}/posts" > /dev/null 2>&1; then
                log_success "✓ $service_name service is accessible via /posts endpoint."
                return 0
            fi
        fi
        # For Auth service, try /auth endpoint
        if [[ "$service_name" == "Auth" ]]; then
            if curl -s "${service_url}/auth" > /dev/null 2>&1; then
                log_success "✓ $service_name service is accessible via /auth endpoint."
                return 0
            fi
        fi
        log_info "  ... waiting ($i/15)"
        sleep 2
    done
    
    log_error "CRITICAL: $service_name service at $service_url did not become healthy after 30 seconds."
    exit 1
}

test_server_health() {
    log_info "=== Testing Posts Service Health ==="
    wait_for_service "$BASE_URL" "Posts"
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
    
    log_info "Polling MailHog for verification email to ${TEST_EMAIL}..."
    local verification_code=""
    local mailhog_response=""
    local email_body=""
    
    # Use exponential backoff (1s, 2s, 4s, 8s, 16s) for a total of 31s.
    for i in {1..5}; do
        delay=$((1 << (i-1)))
        log_info "  ... waiting ${delay}s (attempt $i/5)"
        sleep $delay
        
        mailhog_response=$(get_latest_email_for_recipient "$TEST_EMAIL")
        email_body=$(extract_email_body "$mailhog_response")
        verification_code=$(extract_verification_code "$email_body")
        
        if [[ -n "$verification_code" ]]; then
            log_success "✓ Verification code found: ${verification_code}"
            break
        fi
    done
    
    if [[ -z "$verification_code" ]]; then
        log_error "CRITICAL: Could not find verification email after multiple retries."
        log_error "--- MAILHOG DEBUG INFO ---"
        log_error "MailHog URL: ${MAILHOG_URL}"
        log_error "Recipient: ${TEST_EMAIL}"
        log_error "Raw MailHog Response: ${mailhog_response}"
        log_error "--- END DEBUG INFO ---"
        CRITICAL_FAILURE=true
        return 1
    fi
    
    local verification_data="verificationId=${VERIFICATION_ID}&code=${verification_code}&responseType=spa"
    
    local verify_response
    verify_response=$(curl -s -X POST "${AUTH_BASE}/signup/verify" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$verification_data")
    
    if [[ "$DEBUG_MODE" == "true" ]]; then
        log_info "DEBUG: Verify response: $verify_response"
    fi
    
    JWT_TOKEN=$(extract_json_field "$verify_response" "accessToken")
    if [[ -z "$JWT_TOKEN" ]]; then
        JWT_TOKEN=$(extract_json_field "$verify_response" "token")
    fi
    if [[ -z "$JWT_TOKEN" ]]; then
        JWT_TOKEN=$(extract_json_field "$verify_response" "access_token")
    fi
    
    USER_ID=$(extract_json_field "$verify_response" "objectId")
    if [[ -z "$USER_ID" ]]; then
        USER_ID=$(extract_json_field "$verify_response" "userId")
    fi
    if [[ -z "$USER_ID" ]]; then
        USER_ID=$(extract_json_field "$verify_response" "user_id")
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

create_test_posts() {
    log_info "=== Creating Test Posts ==="
    
    TEST_POST_IDS=()
    
    local post_bodies=(
        "This is my first test post about Go programming"
        "Another post with some tags #golang #backend #api"
        "A post with an image URL: https://example.com/image.jpg"
        "Post with album structure for multiple images"
        "Simple text post for basic operations"
    )
    
    local post_type_ids=(1 1 1 2 1)
    
    for i in "${!post_bodies[@]}"; do
        local post_body="${post_bodies[$i]}"
        local post_type_id="${post_type_ids[$i]}"
        
        local post_data="{\"postTypeId\":${post_type_id},\"body\":\"${post_body}\",\"tags\":[\"test\",\"e2e\"]}"
        
        local response
        if response=$(make_request "POST" "${POSTS_BASE}/" "$post_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create test post $((i+1))" "false"); then
            local post_id=$(extract_json_field "$response" "objectId")
            if [[ -n "$post_id" ]]; then
                TEST_POST_IDS+=("$post_id")
                log_info "Created post $((i+1)) with ID: $post_id"
            fi
        fi
        
        sleep 0.5
    done
    
    log_success "Created ${#TEST_POST_IDS[@]} test posts"
}

test_create_post() {
    log_info "=== Testing Create Post ==="
    
    make_request "POST" "${POSTS_BASE}/" "{\"postTypeId\":1,\"body\":\"Test post\"}" "" "401" "Create post without auth (should fail)" "false"
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping authenticated create post test"
        return 0
    fi
    
    local post_data="{\"postTypeId\":1,\"body\":\"This is a comprehensive test post created via E2E test at ${TIMESTAMP}\",\"tags\":[\"e2e\",\"test\",\"automation\"]}"
    local response=$(make_request "POST" "${POSTS_BASE}/" "$post_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create post with JWT" "true")
    
    local post_id=$(extract_json_field "$response" "objectId")
    if [[ -n "$post_id" ]]; then
        TEST_POST_IDS+=("$post_id")
        validate_uuid_format "$post_id" "postId" || CRITICAL_FAILURE=true
    fi
}

test_create_post_with_media() {
    log_info "=== Testing Create Post with Media ==="
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping media post test"
        return 0
    fi
    
    local media_post_data="{\"postTypeId\":1,\"body\":\"Post with image and video\",\"image\":\"https://example.com/test-image.jpg\",\"imageFullPath\":\"https://cdn.example.com/test-image.jpg\",\"video\":\"https://example.com/test-video.mp4\",\"thumbnail\":\"https://example.com/thumb.jpg\"}"
    local response=$(make_request "POST" "${POSTS_BASE}/" "$media_post_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create post with media" "false")
    
    local post_id=$(extract_json_field "$response" "objectId")
    if [[ -n "$post_id" ]]; then
        TEST_POST_IDS+=("$post_id")
    fi
}

test_create_post_with_album() {
    log_info "=== Testing Create Post with Album ==="
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping album post test"
        return 0
    fi
    
    local album_post_data="{\"postTypeId\":2,\"body\":\"Post with photo album\",\"album\":{\"count\":3,\"cover\":\"https://example.com/cover.jpg\",\"photos\":[\"https://example.com/photo1.jpg\",\"https://example.com/photo2.jpg\",\"https://example.com/photo3.jpg\"],\"title\":\"Test Album\"}}"
    local response=$(make_request "POST" "${POSTS_BASE}/" "$album_post_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create post with album" "false")
    
    local post_id=$(extract_json_field "$response" "objectId")
    if [[ -n "$post_id" ]]; then
        TEST_POST_IDS+=("$post_id")
    fi
}

test_get_post() {
    log_info "=== Testing Get Post by ID ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping get post test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    make_request "GET" "${POSTS_BASE}/${post_id}" "" "" "401" "Get post without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "GET" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post by ID with JWT" "true")
        validate_post_response "$response" "post by ID" || CRITICAL_FAILURE=true
    fi
}

test_get_post_by_urlkey() {
    log_info "=== Testing Get Post by URL Key ==="
    
    if [[ ${#TEST_URL_KEYS[@]} -eq 0 ]]; then
        log_info "No URL keys available yet, will test after generating keys"
        return 0
    fi
    
    local url_key="${TEST_URL_KEYS[0]}"
    
    if [[ -z "$url_key" ]]; then
        log_info "URL key not available, skipping URL key lookup test"
        return 0
    fi
    
    make_request "GET" "${POSTS_BASE}/urlkey/${url_key}" "" "" "401" "Get post by URL key without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "GET" "${POSTS_BASE}/urlkey/${url_key}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post by URL key with JWT" "false")
        validate_post_response "$response" "post by URL key"
    fi
}

test_query_posts() {
    log_info "=== Testing Query Posts ==="
    
    make_request "GET" "${POSTS_BASE}/" "" "" "401" "Query posts without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "GET" "${POSTS_BASE}/?limit=10" "" "Authorization: Bearer $JWT_TOKEN" "200" "Query posts with JWT" "true")
        validate_posts_array_response "$response" || CRITICAL_FAILURE=true
        
        response=$(make_request "GET" "${POSTS_BASE}/?limit=5&owner=${USER_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Query posts by owner" "true")
        validate_posts_array_response "$response" || CRITICAL_FAILURE=true
    fi
}

test_query_posts_with_cursor() {
    log_info "=== Testing Query Posts with Cursor ==="
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "GET" "${POSTS_BASE}/queries/cursor?limit=5" "" "Authorization: Bearer $JWT_TOKEN" "200" "Query posts with cursor pagination" "true")
        validate_posts_array_response "$response" || CRITICAL_FAILURE=true
        
        local next_cursor=$(extract_json_field "$response" "nextCursor")
        if [[ -n "$next_cursor" ]]; then
            response=$(make_request "GET" "${POSTS_BASE}/queries/cursor?limit=5&cursor=${next_cursor}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Query posts with next cursor" "false")
            validate_posts_array_response "$response" || CRITICAL_FAILURE=true
        fi
    fi
}

test_search_posts_with_cursor() {
    log_info "=== Testing Search Posts with Cursor ==="
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "GET" "${POSTS_BASE}/queries/search/cursor?q=test&limit=5" "" "Authorization: Bearer $JWT_TOKEN" "200" "Search posts with cursor" "true")
        validate_posts_array_response "$response" || CRITICAL_FAILURE=true
    fi
}

test_get_cursor_info() {
    log_info "=== Testing Get Cursor Info ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping cursor info test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        make_request "GET" "${POSTS_BASE}/cursor/info/${post_id}?sortBy=createdDate&sortOrder=desc" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get cursor info for post" "false"
    fi
}

test_update_post() {
    log_info "=== Testing Update Post ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping update test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    local update_data="{\"objectId\":\"${post_id}\",\"body\":\"Updated post content at ${TIMESTAMP}\",\"tags\":[\"updated\",\"e2e\"]}"
    
    make_request "PUT" "${POSTS_BASE}/" "$update_data" "" "401" "Update post without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        make_request "PUT" "${POSTS_BASE}/" "$update_data" "Authorization: Bearer $JWT_TOKEN" "200" "Update post with JWT" "true"
        
        sleep 1
        
        local response=$(make_request "GET" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Verify post update" "true")
        local updated_body=$(extract_json_field "$response" "body")
        if [[ "$updated_body" == "Updated post content at ${TIMESTAMP}" ]]; then
            log_success "✓ Post update verified successfully"
        else
            log_error "Post update verification failed: expected 'Updated post content at ${TIMESTAMP}', got '$updated_body'"
            CRITICAL_FAILURE=true
        fi
    fi
}

test_update_post_profile() {
    log_info "=== Testing Update Post Profile ==="
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping update post profile test"
        return 0
    fi
    
    local profile_data="{\"ownerUserId\":\"${USER_ID}\",\"ownerDisplayName\":\"Updated Display Name\",\"ownerAvatar\":\"https://example.com/avatar.jpg\"}"
    make_request "PUT" "${POSTS_BASE}/profile" "$profile_data" "Authorization: Bearer $JWT_TOKEN" "200" "Update post profile" "false"
}

test_disable_comment() {
    log_info "=== Testing Disable Comment ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping disable comment test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    local disable_data="{\"objectId\":\"${post_id}\",\"disable\":true}"
    
    make_request "PUT" "${POSTS_BASE}/comment/disable" "$disable_data" "" "401" "Disable comment without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        make_request "PUT" "${POSTS_BASE}/comment/disable" "$disable_data" "Authorization: Bearer $JWT_TOKEN" "200" "Disable comment with JWT" "false"
    fi
}

test_disable_sharing() {
    log_info "=== Testing Disable Sharing ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping disable sharing test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    local disable_data="{\"objectId\":\"${post_id}\",\"disable\":true}"
    
    make_request "PUT" "${POSTS_BASE}/share/disable" "$disable_data" "" "401" "Disable sharing without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        make_request "PUT" "${POSTS_BASE}/share/disable" "$disable_data" "Authorization: Bearer $JWT_TOKEN" "200" "Disable sharing with JWT" "false"
    fi
}

test_generate_post_urlkey() {
    log_info "=== Testing Generate Post URL Key ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping URL key generation test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    make_request "PUT" "${POSTS_BASE}/urlkey/${post_id}" "" "" "401" "Generate URL key without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        local response=$(make_request "PUT" "${POSTS_BASE}/urlkey/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Generate URL key with JWT" "false")
        local url_key=$(extract_json_field "$response" "urlKey")
        if [[ -n "$url_key" ]]; then
            TEST_URL_KEYS+=("$url_key")
            log_success "Generated URL key: $url_key"
        fi
    fi
}

test_delete_post() {
    log_info "=== Testing Delete Post ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping delete test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[-1]}"
    
    make_request "DELETE" "${POSTS_BASE}/${post_id}" "" "" "401" "Delete post without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        make_request "DELETE" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "204" "Delete post with JWT" "false"
        
        sleep 1
        
        make_request "GET" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "404" "Verify post deleted (should return 404)" "false"
        
        unset 'TEST_POST_IDS[-1]'
    fi
}

test_communication_mode() {
    log_info "=== Testing Communication Mode Configuration ==="
    
    log_info "Current deployment mode: ${DEPLOYMENT_MODE}"
    
    if [[ "$DEPLOYMENT_MODE" == "microservices" ]]; then
        log_success "✅ gRPC Adapter Mode"
        log_info "  - CommentCounter uses gRPC network calls to Comments service"
        log_info "  - PostStatsUpdater uses gRPC network calls to Posts service"
        log_info "  - Enables independent service scaling"
        log_info "  - Optimal for Kubernetes microservices deployment"
        log_info "  - Comments gRPC Address: ${COMMENTS_SERVICE_GRPC_ADDR}"
        log_info "  - Posts gRPC Address: ${POSTS_SERVICE_GRPC_ADDR}"
    else
        log_success "✅ Direct Call Adapter Mode"
        log_info "  - CommentCounter uses in-process calls"
        log_info "  - PostStatsUpdater uses in-process calls"
        log_info "  - Zero network overhead"
        log_info "  - Optimal for serverless/monolith deployment"
    fi
    
    log_info ""
    log_info "To test the other mode:"
    if [[ "$DEPLOYMENT_MODE" == "microservices" ]]; then
        log_info "  1. Start combined server: cd apps/api && DEPLOYMENT_MODE=serverless go run cmd/server/main.go"
        log_info "  2. Run tests: DEPLOYMENT_MODE=serverless bash tools/dev/scripts/posts_e2e_test.sh"
    else
        log_info "  1. Start Comments service: cd apps/api && START_GRPC_SERVER=true GRPC_PORT=50052 go run cmd/services/comments/main.go"
        log_info "  2. Start Posts service: cd apps/api && START_GRPC_SERVER=true GRPC_PORT=50053 go run cmd/services/posts/main.go"
        log_info "  3. Start main server: cd apps/api && DEPLOYMENT_MODE=microservices COMMENTS_SERVICE_GRPC_ADDR=localhost:50052 POSTS_SERVICE_GRPC_ADDR=localhost:50053 go run cmd/server/main.go"
        log_info "  4. Run tests: DEPLOYMENT_MODE=microservices COMMENTS_SERVICE_GRPC_ADDR=localhost:50052 POSTS_SERVICE_GRPC_ADDR=localhost:50053 bash tools/dev/scripts/posts_e2e_test.sh"
    fi
}

test_cross_service_comment_count() {
    log_info "=== Testing Cross-Service Comment Count (CommentCounter & PostStatsUpdater) ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping comment count test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping comment count test"
        return 0
    fi
    
    log_info "Step 1: Getting initial comment count..."
    local post_response=$(make_request "GET" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post to check initial comment count" "true")
    local initial_count=$(extract_json_field "$post_response" "commentCounter")
    initial_count="${initial_count:-0}"
    log_info "  Initial comment count: ${initial_count}"
    
    log_info "Step 2: Creating multiple comments to test PostStatsUpdater.IncrementCommentCountForService()..."
    
    local comments_created=0
    for i in {1..3}; do
        local comment_data="{\"postId\":\"${post_id}\",\"text\":\"Test comment ${i} for PostStatsUpdater verification at ${TIMESTAMP}\"}"
        local response=$(make_request "POST" "${COMMENTS_BASE}/" "$comment_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create comment ${i} to test counter increment" "false")
        local comment_id=$(extract_json_field "$response" "objectId")
        if [[ -n "$comment_id" ]]; then
            comments_created=$((comments_created + 1))
            log_info "  Created comment ${i}: ${comment_id}"
        fi
        sleep 0.5
    done
    
    log_info "Step 3: Waiting for commentCounter to update (PostStatsUpdater may be asynchronous)..."
    sleep 2
    
    log_info "Step 4: Verifying post comment count was updated via PostStatsUpdater adapter..."
    post_response=$(make_request "GET" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post to verify comment count increment" "true")
    local updated_count=$(extract_json_field "$post_response" "commentCounter")
    updated_count="${updated_count:-0}"
    
    log_info "  Initial count: ${initial_count}"
    log_info "  Comments created: ${comments_created}"
    log_info "  Updated count: ${updated_count}"
    log_info "  Expected count: $((initial_count + comments_created))"
    
    if [[ -n "$updated_count" ]] && [[ "$updated_count" -ge $((initial_count + comments_created)) ]]; then
        log_success "✓ Comment counter incremented correctly!"
        log_success "  Initial: ${initial_count} → Final: ${updated_count} (increment: +${comments_created})"
        log_info "  This verifies PostStatsUpdater.IncrementCommentCountForService() is working (${DEPLOYMENT_MODE} mode)"
        log_info "  Flow: Comment creation → updatePostCommentCounter() → PostStatsUpdater → incrementCommentCountForService() → IncrementFields()"
    elif [[ -n "$updated_count" ]] && [[ "$updated_count" -gt "$initial_count" ]]; then
        log_warning "⚠️  Comment counter updated but may be incomplete"
        log_warning "  Initial: ${initial_count} → Final: ${updated_count} (expected: $((initial_count + comments_created)))"
        log_info "  This may indicate partial PostStatsUpdater updates or async timing issues"
    else
        log_error "❌ Comment counter did NOT increment!"
        log_error "  Initial: ${initial_count} → Final: ${updated_count} (expected: $((initial_count + comments_created)))"
        log_error "  This indicates PostStatsUpdater.IncrementCommentCountForService() is NOT working correctly!"
        log_error "  Possible causes:"
        log_error "    1. PostStatsUpdater adapter not wired correctly"
        log_error "    2. updatePostCommentCounter() not being called on comment creation"
        log_error "    3. IncrementFields() failing silently"
        log_error "    4. Database transaction not committing"
        CRITICAL_FAILURE=true
    fi
    
    log_info "Step 5: Verifying CommentCounter.GetRootCommentCount() is working..."
    post_response=$(make_request "GET" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post to verify CommentCounter adapter" "true")
    local read_count=$(extract_json_field "$post_response" "commentCounter")
    
    if [[ -n "$read_count" ]]; then
        log_success "✓ CommentCounter adapter accessible (${DEPLOYMENT_MODE} mode)"
        log_info "  Count is populated via CommentCounter.GetRootCommentCount()"
    else
        log_warning "CommentCounter field not found in response"
    fi
}

test_hmac_protected_endpoints() {
    log_info "=== Testing HMAC-Protected Endpoints ==="
    
    log_info "--- Negative Tests: Without HMAC Authentication ---"
    
    local index_data="{\"body\":\"text\",\"objectId\":1}"
    make_request "POST" "${POSTS_BASE}/actions/index" "$index_data" "" "401" "Create index without HMAC (should fail)" "false"
    
    local score_data="{\"postId\":\"${TEST_POST_IDS[0]:-00000000-0000-0000-0000-000000000000}\",\"delta\":1}"
    make_request "PUT" "${POSTS_BASE}/actions/score" "$score_data" "" "401" "Increment score without HMAC (should fail)" "false"
    
    local comment_count_data="{\"postId\":\"${TEST_POST_IDS[0]:-00000000-0000-0000-0000-000000000000}\",\"count\":1}"
    make_request "PUT" "${POSTS_BASE}/actions/comment/count" "$comment_count_data" "" "401" "Increment comment count without HMAC (should fail)" "false"
    
    log_info "--- Positive Tests: With Valid HMAC Authentication ---"
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping HMAC positive tests"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    local hmac_headers=$(build_hmac_headers "POST" "/posts/actions/index" "" "")
    make_request "POST" "${POSTS_BASE}/actions/index" "" "$hmac_headers" "201" "Create index with valid HMAC" "false"
    
    local score_data="{\"postId\":\"${post_id}\",\"delta\":5}"
    local hmac_headers=$(build_hmac_headers "PUT" "/posts/actions/score" "" "$score_data")
    make_request "PUT" "${POSTS_BASE}/actions/score" "$score_data" "$hmac_headers" "200" "Increment score with valid HMAC" "false"
    
    local comment_count_data="{\"postId\":\"${post_id}\",\"count\":3}"
    local hmac_headers=$(build_hmac_headers "PUT" "/posts/actions/comment/count" "" "$comment_count_data")
    make_request "PUT" "${POSTS_BASE}/actions/comment/count" "$comment_count_data" "$hmac_headers" "200" "Increment comment count with valid HMAC" "false"
    log_info "  Note: This tests PostStatsUpdater.IncrementCommentCountForService() (${DEPLOYMENT_MODE} mode)"
}

test_validation_errors() {
    log_info "=== Testing Input Validation ==="
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping validation tests"
        return 0
    fi
    
    log_info "--- UUID Validation Tests ---"
    # Note: With route constraints, invalid UUIDs now return 404 (route doesn't match) instead of 400
    # This is more RESTful - invalid route parameters don't match any route
    make_request "GET" "${POSTS_BASE}/invalid-uuid" "" "Authorization: Bearer $JWT_TOKEN" "404" "Invalid UUID format (should return 404 with route constraint)" "false"
    make_request "GET" "${POSTS_BASE}/550e8400-e29b-41d4" "" "Authorization: Bearer $JWT_TOKEN" "404" "Incomplete UUID (route constraint)" "false"
    
    log_info "--- Create Post Validation Tests ---"
    make_request "POST" "${POSTS_BASE}/" "{\"body\":\"Test\"}" "Authorization: Bearer $JWT_TOKEN" "400" "Create post without postTypeId" "false"
    make_request "POST" "${POSTS_BASE}/" "{\"postTypeId\":1}" "Authorization: Bearer $JWT_TOKEN" "400" "Create post without body" "false"
    make_request "POST" "${POSTS_BASE}/" "{\"postTypeId\":1,\"body\":\"\"}" "Authorization: Bearer $JWT_TOKEN" "400" "Create post with empty body" "false"
    
    log_info "--- Update Post Validation Tests ---"
    if [[ ${#TEST_POST_IDS[@]} -gt 0 ]]; then
        make_request "PUT" "${POSTS_BASE}/" "{\"body\":\"Updated\"}" "Authorization: Bearer $JWT_TOKEN" "400" "Update post without objectId" "false"
        make_request "PUT" "${POSTS_BASE}/" "{\"objectId\":\"invalid-uuid\",\"body\":\"Updated\"}" "Authorization: Bearer $JWT_TOKEN" "400" "Update post with invalid objectId" "false"
    fi
    
    log_info "--- Pagination Edge Cases ---"
    make_request "GET" "${POSTS_BASE}/?limit=0" "" "Authorization: Bearer $JWT_TOKEN" "200" "limit=0 (should default)" "false"
    make_request "GET" "${POSTS_BASE}/?limit=-1" "" "Authorization: Bearer $JWT_TOKEN" "200" "limit=-1 (should default)" "false"
    make_request "GET" "${POSTS_BASE}/?limit=1000" "" "Authorization: Bearer $JWT_TOKEN" "200" "Very large limit (should cap)" "false"
    
    log_info "--- JSON Parsing Validation ---"
    make_request "POST" "${POSTS_BASE}/" "{invalid json}" "Authorization: Bearer $JWT_TOKEN" "400" "Malformed JSON" "false"
    make_request "PUT" "${POSTS_BASE}/" "not-json" "Authorization: Bearer $JWT_TOKEN" "400" "Non-JSON body" "false"
}

test_integration_flow() {
    log_info "=== Testing Complete Integration Flow ==="
    
    if [[ -z "$JWT_TOKEN" ]] || [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "Insufficient test data for integration flow"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    log_info "--- Step 1: Create Post ---"
    local create_data="{\"postTypeId\":1,\"body\":\"Integration test post ${TIMESTAMP}\",\"tags\":[\"integration\"]}"
    local create_response=$(make_request "POST" "${POSTS_BASE}/" "$create_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create post for integration test" "false")
    local new_post_id=$(extract_json_field "$create_response" "objectId")
    
    if [[ -z "$new_post_id" ]]; then
        log_warning "Failed to create post for integration test"
        return 0
    fi
    
    sleep 1
    
    log_info "--- Step 2: Read Post ---"
    local read_response=$(make_request "GET" "${POSTS_BASE}/${new_post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read created post" "false")
    validate_post_response "$read_response" "integration post"
    
    log_info "--- Step 3: Update Post ---"
    local update_data="{\"objectId\":\"${new_post_id}\",\"body\":\"Updated integration post\"}"
    make_request "PUT" "${POSTS_BASE}/" "$update_data" "Authorization: Bearer $JWT_TOKEN" "200" "Update post" "false"
    
    sleep 1
    
    log_info "--- Step 4: Verify Update ---"
    local verify_response=$(make_request "GET" "${POSTS_BASE}/${new_post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Verify update" "false")
    local updated_body=$(extract_json_field "$verify_response" "body")
    if [[ "$updated_body" == "Updated integration post" ]]; then
        log_success "✓ Integration flow verified successfully"
    else
        log_error "Integration flow failed: body mismatch"
        return 1
    fi
    
    log_info "--- Step 5: Generate URL Key ---"
    local urlkey_response=$(make_request "PUT" "${POSTS_BASE}/urlkey/${new_post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Generate URL key" "false")
    local url_key=$(extract_json_field "$urlkey_response" "urlKey")
    
    if [[ -n "$url_key" ]]; then
        log_info "--- Step 6: Access via URL Key ---"
        make_request "GET" "${POSTS_BASE}/urlkey/${url_key}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Access post via URL key" "false"
    fi
    
    log_info "--- Step 7: Query Posts ---"
    local query_response=$(make_request "GET" "${POSTS_BASE}/?owner=${USER_ID}&limit=10" "" "Authorization: Bearer $JWT_TOKEN" "200" "Query posts by owner" "false")
    validate_posts_array_response "$query_response"
    
    log_info "--- Step 8: Delete Post ---"
    make_request "DELETE" "${POSTS_BASE}/${new_post_id}" "" "Authorization: Bearer $JWT_TOKEN" "204" "Delete post" "false"
    
    log_success "✓ Complete integration flow executed successfully"
}

main() {
    log_info "========================================"
    log_info "Posts Microservice E2E Testing Suite"
    log_info "========================================"
    log_info "Posts Service URL: $BASE_URL"
    log_info "Auth Service URL: $AUTH_URL"
    log_info "Deployment Mode: $DEPLOYMENT_MODE"
    log_info "Debug Mode: $DEBUG_MODE"
    log_info "Fail Fast: $FAIL_FAST"
    log_info "CI Report: $GENERATE_CI_REPORT"
    echo
    
    detect_database_type
    verify_database_connection
    
    wait_for_service "$BASE_URL" "Posts"
    wait_for_service "$AUTH_URL" "Auth"
    echo
    
    log_info "=== PHASE 0: Communication Mode Verification ==="
    test_communication_mode
    echo
    
    log_info "=== PHASE 1: Test Setup ==="
    setup_test_user
    echo
    create_test_posts
    echo
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_error "CRITICAL: Failed to obtain JWT token - cannot proceed with authenticated tests"
        CRITICAL_FAILURE=true
        cleanup_and_exit 1
    fi
    
    log_info "=== PHASE 2: User-Facing Endpoints (JWT/Cookie Auth) ==="
    test_create_post
    echo
    test_create_post_with_media
    echo
    test_create_post_with_album
    echo
    test_get_post
    echo
    test_query_posts
    echo
    test_query_posts_with_cursor
    echo
    test_search_posts_with_cursor
    echo
    test_get_cursor_info
    echo
    test_update_post
    echo
    test_update_post_profile
    echo
    test_disable_comment
    echo
    test_disable_sharing
    echo
    test_generate_post_urlkey
    echo
    test_get_post_by_urlkey
    echo
    
    log_info "=== PHASE 3: Cross-Service Communication Tests ==="
    test_cross_service_comment_count
    echo
    
    log_info "=== PHASE 4: Service-to-Service Endpoints (HMAC Auth) ==="
    test_hmac_protected_endpoints
    echo
    
    log_info "=== PHASE 5: Input Validation & Edge Cases ==="
    test_validation_errors
    echo
    
    log_info "=== PHASE 6: Integration Tests ==="
    test_integration_flow
    echo
    
    log_info "=== PHASE 7: Delete Operations ==="
    test_delete_post
    echo
    
    local end_time=$(date +%s)
    local duration=$((end_time - TEST_START_TIME))
    
    log_info "========================================"
    log_success "All Posts Microservice tests completed!"
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
    log_info "  Deployment Mode: $DEPLOYMENT_MODE"
    log_info "  Critical Issues: $([ "$CRITICAL_FAILURE" == "true" ] && echo "YES" || echo "NO")"
    echo
    
    if [[ "$CRITICAL_FAILURE" == "true" ]]; then
        log_error "❌ Test suite FAILED with critical issues"
        exit 1
    elif [[ $TEST_COUNT_FAILED -gt 0 ]]; then
        log_warning "⚠️  Test suite completed with $TEST_COUNT_FAILED failures"
        exit 1
    else
        log_success "🎉 Posts Microservice is fully functional!"
        exit 0
    fi
}

main "$@"

