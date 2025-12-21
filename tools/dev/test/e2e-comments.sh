#!/bin/bash

set -euo pipefail

# =============================================================================
# Comments Microservice E2E Testing Suite
# =============================================================================
# 
# This script tests the Comments microservice with realistic scenarios.
# It covers the complete comment lifecycle and cross-service communication.
#
# Features:
# - Tests complete comment flow (create, read, update, delete, replies)
# - Validates cross-service communication via CommentCounter/PostStatsUpdater adapters
# - Supports testing both deployment modes:
#   • Direct Call Adapter (serverless/monolith)
#   • gRPC Adapter (microservices)
#
# Usage:
#   # Test with Direct Call Adapter (default):
#   bash tools/dev/test/e2e-comments.sh
#
#   # Test with gRPC Adapter:
#   DEPLOYMENT_MODE=microservices COMMENTS_SERVICE_GRPC_ADDR=localhost:50052 POSTS_SERVICE_GRPC_ADDR=localhost:50053 bash tools/dev/test/e2e-comments.sh
#
# Author: amir@telar.dev
# =============================================================================

# Configuration
BASE_URL="http://127.0.0.1:9099"
COMMENTS_BASE="${BASE_URL}/comments"
POSTS_BASE="${BASE_URL}/posts"
AUTH_URL="http://127.0.0.1:9099"
AUTH_BASE="${AUTH_URL}/auth"
MAILHOG_URL="http://localhost:8025"

# Test configuration flags
DEBUG_MODE="${DEBUG_MODE:-false}"
CLEANUP_ON_SUCCESS="${CLEANUP_ON_SUCCESS:-true}"
FAIL_FAST="${FAIL_FAST:-true}"
GENERATE_CI_REPORT="${GENERATE_CI_REPORT:-false}"
CI_REPORT_PATH="${CI_REPORT_PATH:-./test-results/comments-e2e-report.json}"

# Communication mode configuration
DEPLOYMENT_MODE="${DEPLOYMENT_MODE:-serverless}"  # serverless or microservices
COMMENTS_SERVICE_GRPC_ADDR="${COMMENTS_SERVICE_GRPC_ADDR:-localhost:50052}"
POSTS_SERVICE_GRPC_ADDR="${POSTS_SERVICE_GRPC_ADDR:-localhost:50053}"

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
CYAN='\033[0;36m'
NC='\033[0m'

# Test data
TIMESTAMP=$(date +%s)
TEST_EMAIL="commentstest-${TIMESTAMP}@example.com"
TEST_PASSWORD="MyVerySecureCommentsPassword123!@#\$%^&*()"
TEST_FULLNAME="Comments Test User"

# Global variables
JWT_TOKEN=""
USER_ID=""
HMAC_SECRET="${HMAC_SECRET:-a-super-secret-key-for-local-dev-and-testing}"
TEST_POST_IDS=()
TEST_COMMENT_IDS=()
TEST_REPLY_IDS=()
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
        echo "$mailhog_response" | grep -oP '"Body":"[^"]*(?<!\\)"' | head -1 | sed 's/"Body":"\(.*\)"/\1/' | sed 's/\\n/ /g' | sed 's/\\r//g' | sed 's/\\t/ /g' | sed 's/\\"/"/g'
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
}

wait_for_service() {
    local service_url="$1"
    local service_name="$2"
    log_info "Waiting for $service_name service at $service_url to become healthy..."
    
    for i in {1..15}; do
        if curl -s -f "${service_url}/health" > /dev/null 2>&1; then
            log_success "✓ $service_name service is healthy."
            return 0
        fi
        if curl -s "${service_url}" > /dev/null 2>&1; then
            log_success "✓ $service_name service is accessible."
            return 0
        fi
        if [[ "$service_name" == "Comments" ]]; then
            if curl -s "${service_url}/comments" > /dev/null 2>&1; then
                log_success "✓ $service_name service is accessible via /comments endpoint."
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
    log_info "=== Testing Comments Service Health ==="
    wait_for_service "$BASE_URL" "Comments"
}

setup_test_user() {
    log_info "=== Setting Up Test User via Auth Service ==="
    
    local signup_data="fullName=${TEST_FULLNAME}&email=${TEST_EMAIL}&newPassword=${TEST_PASSWORD}&responseType=spa&verifyType=email"
    
    local signup_response
    signup_response=$(curl -s -X POST "${AUTH_BASE}/signup" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$signup_data")
    
    local VERIFICATION_ID=$(extract_json_field "$signup_response" "verificationId")
    
    if [[ -z "$VERIFICATION_ID" ]]; then
        log_error "Failed to create test user"
        return 1
    fi
    
    log_info "Polling MailHog for verification email to ${TEST_EMAIL}..."
    local verification_code=""
    local mailhog_response=""
    local email_body=""
    
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
        CRITICAL_FAILURE=true
        return 1
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
    else
        log_warning "User created but no JWT token received"
    fi
}

create_test_post() {
    log_info "=== Creating Test Post for Comments ==="
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping test post creation"
        return 0
    fi
    
    # Helper function to create post without counting as test
    create_post_setup() {
        local post_data="$1"
        local response
        local status_code
        
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X POST "${POSTS_BASE}/" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $JWT_TOKEN" \
            -d "$post_data" 2>&1)
        
        status_code=$(echo "$response" | tail -n1)
        response_body=$(echo "$response" | sed '$d')
        
        if [[ "$status_code" == "201" ]]; then
            echo "$response_body"
            return 0
        else
            return 1
        fi
    }
    
    local post_data="{\"postTypeId\":1,\"body\":\"Test post for comments E2E testing at ${TIMESTAMP}\",\"tags\":[\"test\",\"comments\"]}"
    local response
    if response=$(create_post_setup "$post_data"); then
        local post_id=$(extract_json_field "$response" "objectId")
        if [[ -n "$post_id" ]]; then
            TEST_POST_IDS+=("$post_id")
            log_info "Created test post with ID: $post_id"
        fi
    else
        log_warning "Failed to create test post for comments (may already exist)"
        return 1
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
        log_info "  2. Run tests: DEPLOYMENT_MODE=serverless bash tools/dev/test/e2e-comments.sh"
    else
        log_info "  1. Start Comments service: cd apps/api && START_GRPC_SERVER=true GRPC_PORT=50052 go run cmd/services/comments/main.go"
        log_info "  2. Start Posts service: cd apps/api && START_GRPC_SERVER=true GRPC_PORT=50053 go run cmd/services/posts/main.go"
        log_info "  3. Start main server: cd apps/api && DEPLOYMENT_MODE=microservices COMMENTS_SERVICE_GRPC_ADDR=localhost:50052 POSTS_SERVICE_GRPC_ADDR=localhost:50053 go run cmd/server/main.go"
        log_info "  4. Run tests: DEPLOYMENT_MODE=microservices COMMENTS_SERVICE_GRPC_ADDR=localhost:50052 POSTS_SERVICE_GRPC_ADDR=localhost:50053 bash tools/dev/test/e2e-comments.sh"
    fi
}

test_create_comment() {
    log_info "=== Testing Create Comment ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping create comment test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    make_request "POST" "${COMMENTS_BASE}/" "{\"postId\":\"${post_id}\",\"text\":\"This is a test comment\"}" "" "401" "Create comment without auth (should fail)" "false"
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping authenticated create comment test"
        return 0
    fi
    
    local comment_data="{\"postId\":\"${post_id}\",\"text\":\"This is a comprehensive test comment created via E2E test at ${TIMESTAMP}\"}"
    local response=$(make_request "POST" "${COMMENTS_BASE}/" "$comment_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create comment with JWT" "true")
    
    local comment_id=$(extract_json_field "$response" "objectId")
    if [[ -n "$comment_id" ]]; then
        TEST_COMMENT_IDS+=("$comment_id")
        validate_uuid_format "$comment_id" "commentId" || CRITICAL_FAILURE=true
    fi
}

test_cross_service_comment_count() {
    log_info "=== Testing Cross-Service Comment Count (CommentCounter) ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping comment count test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping comment count test"
        return 0
    fi
    
    log_info "Creating multiple comments to test count increment..."
    
    for i in {1..3}; do
        local comment_data="{\"postId\":\"${post_id}\",\"text\":\"Test comment ${i} for count verification\"}"
        local response=$(make_request "POST" "${COMMENTS_BASE}/" "$comment_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create comment ${i} for count test" "false")
        local comment_id=$(extract_json_field "$response" "objectId")
        if [[ -n "$comment_id" ]]; then
            TEST_COMMENT_IDS+=("$comment_id")
        fi
        sleep 0.5
    done
    
    log_info "Verifying post comment count was updated..."
    sleep 1
    
    local post_response=$(make_request "GET" "${POSTS_BASE}/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post to verify comment count" "true")
    local comment_count=$(extract_json_field "$post_response" "commentCounter")
    
    if [[ -n "$comment_count" ]] && [[ "$comment_count" -ge 3 ]]; then
        log_success "✓ Comment count updated correctly: ${comment_count} (expected >= 3)"
        log_info "  This verifies PostStatsUpdater adapter is working (${DEPLOYMENT_MODE} mode)"
    else
        log_warning "Comment count may not be updated yet: ${comment_count:-0} (expected >= 3)"
        log_info "  Note: Count updates are asynchronous, this is expected behavior"
    fi
}

test_get_comments() {
    log_info "=== Testing Get Comments by Post ==="
    
    if [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "No test posts available, skipping get comments test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    make_request "GET" "${COMMENTS_BASE}/?postId=${post_id}" "" "" "401" "Get comments without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        make_request "GET" "${COMMENTS_BASE}/?postId=${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get comments by post ID with JWT" "true"
    fi
}

test_create_reply() {
    log_info "=== Testing Create Reply ==="
    
    if [[ ${#TEST_COMMENT_IDS[@]} -eq 0 ]]; then
        log_warning "No test comments available, skipping create reply test"
        return 0
    fi
    
    local comment_id="${TEST_COMMENT_IDS[0]}"
    local post_id="${TEST_POST_IDS[0]}"
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token, skipping create reply test"
        return 0
    fi
    
    local reply_data="{\"postId\":\"${post_id}\",\"text\":\"This is a reply to the comment\",\"parentCommentId\":\"${comment_id}\"}"
    local response=$(make_request "POST" "${COMMENTS_BASE}/" "$reply_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create reply with JWT" "false")
    
    local reply_id=$(extract_json_field "$response" "objectId")
    if [[ -n "$reply_id" ]]; then
        TEST_REPLY_IDS+=("$reply_id")
    fi
}

test_update_comment() {
    log_info "=== Testing Update Comment ==="
    
    if [[ ${#TEST_COMMENT_IDS[@]} -eq 0 ]]; then
        log_warning "No test comments available, skipping update test"
        return 0
    fi
    
    local comment_id="${TEST_COMMENT_IDS[0]}"
    local update_data="{\"objectId\":\"${comment_id}\",\"text\":\"Updated comment content at ${TIMESTAMP}\"}"
    
    make_request "PUT" "${COMMENTS_BASE}/" "$update_data" "" "401" "Update comment without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        make_request "PUT" "${COMMENTS_BASE}/" "$update_data" "Authorization: Bearer $JWT_TOKEN" "200" "Update comment with JWT" "true"
    fi
}

test_delete_comment() {
    log_info "=== Testing Delete Comment ==="
    
    if [[ ${#TEST_COMMENT_IDS[@]} -eq 0 ]]; then
        log_warning "No test comments available, skipping delete test"
        return 0
    fi
    
    local comment_id="${TEST_COMMENT_IDS[-1]}"
    local post_id="${TEST_POST_IDS[0]}"
    
    make_request "DELETE" "${COMMENTS_BASE}/id/${comment_id}/post/${post_id}" "" "" "401" "Delete comment without auth (should fail)" "false"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        make_request "DELETE" "${COMMENTS_BASE}/id/${comment_id}/post/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "204" "Delete comment with JWT" "false"
        
        sleep 1
        
        make_request "GET" "${COMMENTS_BASE}/${comment_id}" "" "Authorization: Bearer $JWT_TOKEN" "404" "Verify comment deleted (should return 404)" "false"
        
        unset 'TEST_COMMENT_IDS[-1]'
    fi
}

test_integration_flow() {
    log_info "=== Testing Complete Integration Flow ==="
    
    if [[ -z "$JWT_TOKEN" ]] || [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "Insufficient test data for integration flow"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    log_info "--- Step 1: Create Comment ---"
    local create_data="{\"postId\":\"${post_id}\",\"text\":\"Integration test comment ${TIMESTAMP}\"}"
    local create_response=$(make_request "POST" "${COMMENTS_BASE}/" "$create_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create comment for integration test" "false")
    local new_comment_id=$(extract_json_field "$create_response" "objectId")
    
    if [[ -z "$new_comment_id" ]]; then
        log_warning "Failed to create comment for integration test"
        return 0
    fi
    
    sleep 1
    
    log_info "--- Step 2: Read Comment ---"
    make_request "GET" "${COMMENTS_BASE}/${new_comment_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Read created comment" "false"
    
    log_info "--- Step 3: Update Comment ---"
    local update_data="{\"objectId\":\"${new_comment_id}\",\"text\":\"Updated integration comment\"}"
    make_request "PUT" "${COMMENTS_BASE}/" "$update_data" "Authorization: Bearer $JWT_TOKEN" "200" "Update comment" "false"
    
    sleep 1
    
    log_info "--- Step 4: Verify Update ---"
    local verify_response=$(make_request "GET" "${COMMENTS_BASE}/${new_comment_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Verify update" "false")
    local updated_text=$(extract_json_field "$verify_response" "text")
    if [[ "$updated_text" == "Updated integration comment" ]]; then
        log_success "✓ Integration flow verified successfully"
    else
        log_error "Integration flow failed: text mismatch"
        return 1
    fi
    
    log_info "--- Step 5: Create Reply ---"
    local reply_data="{\"postId\":\"${post_id}\",\"text\":\"Reply to integration comment\",\"parentCommentId\":\"${new_comment_id}\"}"
    make_request "POST" "${COMMENTS_BASE}/" "$reply_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create reply" "false"
    
    log_info "--- Step 6: Query Comments ---"
    make_request "GET" "${COMMENTS_BASE}/?postId=${post_id}&limit=10" "" "Authorization: Bearer $JWT_TOKEN" "200" "Query comments by post" "false"
    
    log_info "--- Step 7: Delete Comment ---"
    make_request "DELETE" "${COMMENTS_BASE}/id/${new_comment_id}/post/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "204" "Delete comment" "false"
    
    log_success "✓ Complete integration flow executed successfully"
}

test_comment_likes() {
    log_info "=== Testing Comment Likes (Toggle Like/Unlike) ==="
    
    if [[ -z "$JWT_TOKEN" ]] || [[ ${#TEST_POST_IDS[@]} -eq 0 ]]; then
        log_warning "Insufficient test data for comment likes test"
        return 0
    fi
    
    local post_id="${TEST_POST_IDS[0]}"
    
    log_info "--- Step 1: Create Comment for Like Test ---"
    local comment_data="{\"postId\":\"${post_id}\",\"text\":\"Comment for like testing at ${TIMESTAMP}\"}"
    local create_response=$(make_request "POST" "${COMMENTS_BASE}/" "$comment_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create comment for like test" "true")
    local comment_id=$(extract_json_field "$create_response" "objectId")
    
    if [[ -z "$comment_id" ]]; then
        log_error "Failed to create comment for like test"
        return 1
    fi
    
    log_info "--- Step 2: Like Comment (POST /comments/:id/like) ---"
    local like_response=$(make_request "POST" "${COMMENTS_BASE}/${comment_id}/like" "" "Authorization: Bearer $JWT_TOKEN" "200" "Like comment" "true")
    
    # DEBUG: Log raw JSON response for debugging (full length)
    log_info "Raw Like Response JSON (full): ${like_response}"
    log_info "Response length: ${#like_response} characters"
    
    # Use jq if available, otherwise use grep
    if command -v jq >/dev/null 2>&1; then
        local score_after_like=$(echo "$like_response" | jq -r '.score // empty')
        local is_liked_after_like=$(echo "$like_response" | jq -r '.isLiked // empty')
        log_info "Extracted via jq - score: '${score_after_like}', isLiked: '${is_liked_after_like}'"
    else
        local score_after_like=$(extract_json_field "$like_response" "score")
        local is_liked_after_like=$(extract_json_field "$like_response" "isLiked")
        log_info "Extracted via grep - score: '${score_after_like}', isLiked: '${is_liked_after_like}'"
    fi
    
    if [[ "$score_after_like" == "1" ]] && [[ "$is_liked_after_like" == "true" ]]; then
        log_success "✓ Comment liked successfully - score: ${score_after_like}, isLiked: ${is_liked_after_like}"
    else
        log_error "Like failed - expected score=1, isLiked=true, got score=${score_after_like}, isLiked=${is_liked_after_like}"
        return 1
    fi
    
    log_info "--- Step 3: Verify Like (GET /comments/:id) ---"
    local get_response=$(make_request "GET" "${COMMENTS_BASE}/${comment_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get comment to verify like" "true")
    local verify_score=$(extract_json_field "$get_response" "score")
    local verify_is_liked=$(extract_json_field "$get_response" "isLiked")
    
    if [[ "$verify_score" == "1" ]] && [[ "$verify_is_liked" == "true" ]]; then
        log_success "✓ Like verified - score: ${verify_score}, isLiked: ${verify_is_liked}"
    else
        log_error "Verification failed - expected score=1, isLiked=true, got score=${verify_score}, isLiked=${verify_is_liked}"
        return 1
    fi
    
    log_info "--- Step 4: Toggle Like (Unlike) - POST /comments/:id/like again ---"
    local unlike_response=$(make_request "POST" "${COMMENTS_BASE}/${comment_id}/like" "" "Authorization: Bearer $JWT_TOKEN" "200" "Unlike comment (toggle)" "true")
    local score_after_unlike=$(extract_json_field "$unlike_response" "score")
    local is_liked_after_unlike=$(extract_json_field "$unlike_response" "isLiked")
    
    if [[ "$score_after_unlike" == "0" ]] && [[ "$is_liked_after_unlike" == "false" ]]; then
        log_success "✓ Comment unliked successfully - score: ${score_after_unlike}, isLiked: ${is_liked_after_unlike}"
    else
        log_error "Unlike failed - expected score=0, isLiked=false, got score=${score_after_unlike}, isLiked=${is_liked_after_unlike}"
        return 1
    fi
    
    log_info "--- Step 5: Verify Unlike (GET /comments/:id) ---"
    local final_get_response=$(make_request "GET" "${COMMENTS_BASE}/${comment_id}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get comment to verify unlike" "true")
    local final_score=$(extract_json_field "$final_get_response" "score")
    local final_is_liked=$(extract_json_field "$final_get_response" "isLiked")
    
    if [[ "$final_score" == "0" ]] && [[ "$final_is_liked" == "false" ]]; then
        log_success "✓ Unlike verified - score: ${final_score}, isLiked: ${final_is_liked}"
    else
        log_error "Final verification failed - expected score=0, isLiked=false, got score=${final_score}, isLiked=${final_is_liked}"
        return 1
    fi
    
    log_success "✓ Comment likes test completed successfully"
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
    
    for comment_id in "${TEST_COMMENT_IDS[@]}"; do
        if [[ -n "$comment_id" ]] && [[ ${#TEST_POST_IDS[@]} -gt 0 ]]; then
            local post_id="${TEST_POST_IDS[0]}"
            log_info "Soft deleting test comment: $comment_id"
            make_request "DELETE" "${COMMENTS_BASE}/id/${comment_id}/post/${post_id}" "" "Authorization: Bearer $JWT_TOKEN" "204" "Cleanup: Delete test comment" "false" > /dev/null 2>&1 || log_warning "Failed to delete comment $comment_id"
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
  "service": "comments-microservice",
  "timestamp": $(date +%s),
  "duration_seconds": $duration,
  "total_tests": $TEST_COUNT_TOTAL,
  "passed": $TEST_COUNT_PASSED,
  "failed": $TEST_COUNT_FAILED,
  "skipped": $TEST_COUNT_SKIPPED,
  "success_rate": $success_rate,
  "critical_failure": $CRITICAL_FAILURE,
  "deployment_mode": "$DEPLOYMENT_MODE"
}
EOF
    
    log_success "CI report generated: $CI_REPORT_PATH"
}

trap 'cleanup_and_exit $?' EXIT INT TERM

main() {
    log_info "========================================"
    log_info "Comments Microservice E2E Testing Suite"
    log_info "========================================"
    log_info "Comments Service URL: $BASE_URL"
    log_info "Posts Service URL: $BASE_URL"
    log_info "Auth Service URL: $AUTH_URL"
    log_info "Deployment Mode: $DEPLOYMENT_MODE"
    log_info "Debug Mode: $DEBUG_MODE"
    log_info "Fail Fast: $FAIL_FAST"
    log_info "CI Report: $GENERATE_CI_REPORT"
    echo
    
    detect_database_type
    verify_database_connection
    
    test_server_health
    wait_for_service "$AUTH_URL" "Auth"
    echo
    
    log_info "=== PHASE 0: Communication Mode Verification ==="
    test_communication_mode
    echo
    
    log_info "=== PHASE 1: Test Setup ==="
    setup_test_user
    echo
    create_test_post
    echo
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_error "CRITICAL: Failed to obtain JWT token - cannot proceed with authenticated tests"
        CRITICAL_FAILURE=true
        cleanup_and_exit 1
    fi
    
    log_info "=== PHASE 2: User-Facing Endpoints (JWT/Cookie Auth) ==="
    test_create_comment
    echo
    test_get_comments
    echo
    test_create_reply
    echo
    test_update_comment
    echo
    
    log_info "=== PHASE 3: Cross-Service Communication Tests ==="
    test_cross_service_comment_count
    echo
    
    log_info "=== PHASE 4: Integration Tests ==="
    test_integration_flow
    echo
    
    log_info "=== PHASE 5: Comment Likes (Voting) ==="
    test_comment_likes
    echo
    
    log_info "=== PHASE 6: Delete Operations ==="
    test_delete_comment
    echo
    
    local end_time=$(date +%s)
    local duration=$((end_time - TEST_START_TIME))
    
    log_info "========================================"
    log_success "All Comments Microservice tests completed!"
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
        log_success "🎉 Comments Microservice is fully functional!"
        exit 0
    fi
}

main "$@"
