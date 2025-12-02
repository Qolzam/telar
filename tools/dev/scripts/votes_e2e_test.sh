#!/bin/bash

set -euo pipefail

# Configuration
BASE_URL="http://127.0.0.1:8080"
VOTES_BASE="${BASE_URL}/votes"
POSTS_BASE="${BASE_URL}/posts"
AUTH_URL="http://127.0.0.1:8080"
AUTH_BASE="${AUTH_URL}/auth"
MAILHOG_URL="http://localhost:8025"

# Test configuration flags
DEBUG_MODE="${DEBUG_MODE:-false}"
CLEANUP_ON_SUCCESS="${CLEANUP_ON_SUCCESS:-true}"
FAIL_FAST="${FAIL_FAST:-true}"

# Test metrics tracking
TEST_COUNT_TOTAL=0
TEST_COUNT_PASSED=0
TEST_COUNT_FAILED=0
TEST_START_TIME=$(date +%s)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Test data
TIMESTAMP=$(date +%s)
TEST_EMAIL="votestest-${TIMESTAMP}@example.com"
TEST_PASSWORD="MyVerySecureVotesPassword123!@#\$%^&*()"
TEST_FULLNAME="Votes Test User"

# Global variables
JWT_TOKEN=""
USER_ID=""
TEST_POST_ID=""
TEST_POST_SCORE=0

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
    local content_type="${7:-application/json}"
    
    TEST_COUNT_TOTAL=$((TEST_COUNT_TOTAL + 1))
    
    log_test "$description" >&2
    echo "  → $method $url" >&2
    
    if [[ -n "$data" && "$DEBUG_MODE" == "true" ]]; then
        echo "  → Data: $data" >&2
    fi
    
    if [[ -n "$headers" && "$DEBUG_MODE" == "true" ]]; then
        echo "  → Headers: $headers" >&2
    fi
    
    local response
    local status_code
    
    if [[ -n "$data" && -n "$headers" ]]; then
        # Use explicit Cookie header (more reliable than -b flag)
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
            -H "Content-Type: $content_type" \
            -H "$headers" \
            -d "$data" 2>/dev/null)
    elif [[ -n "$headers" ]]; then
        # Use explicit Cookie header (more reliable than -b flag)
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
            -H "$headers" 2>/dev/null)
    elif [[ -n "$data" ]]; then
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
            -H "Content-Type: $content_type" \
            -d "$data" 2>/dev/null)
    else
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" 2>/dev/null)
    fi
    
    status_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | sed '$d')
    
    if [[ "$status_code" == "$expected_status" ]]; then
        log_success "✓ Status: $status_code (expected $expected_status)" >&2
        TEST_COUNT_PASSED=$((TEST_COUNT_PASSED + 1))
        echo "$response_body"
        return 0
    else
        log_error "✗ Status: $status_code (expected $expected_status)" >&2
        echo "Response: $response_body" >&2
        TEST_COUNT_FAILED=$((TEST_COUNT_FAILED + 1))
        if [[ "$FAIL_FAST" == "true" ]]; then
            exit 1
        fi
        return 1
    fi
}

extract_json_field() {
    local json="$1"
    local field="$2"
    
    if command -v jq >/dev/null 2>&1; then
        echo "$json" | jq -r ".$field // empty" 2>/dev/null || echo ""
    elif command -v python3 >/dev/null 2>&1; then
        # Support nested fields like "user.objectId"
        if [[ "$field" == *"."* ]]; then
            local parts=$(echo "$field" | tr '.' ' ')
            echo "$json" | python3 -c "
import sys, json
data = json.load(sys.stdin)
parts = '$field'.split('.')
result = data
for part in parts:
    if isinstance(result, dict):
        result = result.get(part, '')
    else:
        result = ''
        break
print(result if result else '')
" 2>/dev/null || echo ""
        else
            echo "$json" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('$field', ''))" 2>/dev/null || echo ""
        fi
    else
        echo "$json" | grep -o "\"$field\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | sed "s/\"$field\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\"/\1/" | head -1
    fi
}

validate_uuid_format() {
    local uuid="$1"
    local field_name="${2:-UUID}"
    
    if [[ ! "$uuid" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]; then
        log_error "Invalid $field_name format: $uuid"
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

extract_verification_code() {
    local mailhog_response="$1"
    local email_body=""
    
    if command -v python3 >/dev/null 2>&1; then
        email_body=$(echo "$mailhog_response" | python3 -c "import sys, json; data=json.load(sys.stdin); items=data.get('items', []); print(items[0]['Content']['Body'] if items else '')" 2>/dev/null || echo "")
    else
        email_body=$(echo "$mailhog_response" | grep -oP '"Body":"[^"]*(?<!\\)"' | head -1 | sed 's/"Body":"\(.*\)"/\1/' | sed 's/\\n/ /g')
    fi
    
    # Extract 6-digit verification code
    local code=$(echo "$email_body" | grep -oE 'code=[0-9]{6}' | grep -oE '[0-9]{6}' | head -1)
    if [[ -z "$code" ]]; then
        code=$(echo "$email_body" | grep -oE '[0-9]{6}' | head -1)
    fi
    echo "$code"
}

wait_for_servers() {
    log_info "Waiting for servers to be ready..."
    
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if curl -s --max-time 2 "${BASE_URL}/health" >/dev/null 2>&1; then
            log_success "API server is ready"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    log_error "API server did not become ready after ${max_attempts} seconds"
    return 1
}

cleanup() {
    if [[ "$CLEANUP_ON_SUCCESS" == "true" && "$TEST_COUNT_FAILED" -eq 0 ]]; then
        log_info "Cleaning up test data..."
        # Cleanup would go here if needed
    fi
}

print_summary() {
    local end_time=$(date +%s)
    local duration=$((end_time - TEST_START_TIME))
    
    echo ""
    echo "=========================================="
    echo "  VOTES E2E TEST SUMMARY"
    echo "=========================================="
    echo "Total Tests: $TEST_COUNT_TOTAL"
    echo "Passed: $TEST_COUNT_PASSED"
    echo "Failed: $TEST_COUNT_FAILED"
    echo "Duration: ${duration}s"
    echo "=========================================="
    
    if [[ $TEST_COUNT_FAILED -eq 0 ]]; then
        log_success "All tests passed!"
        return 0
    else
        log_error "Some tests failed"
        return 1
    fi
}

# Main test execution
main() {
    log_info "Starting Votes E2E Test Suite"
    log_info "Test Email: $TEST_EMAIL"
    
    wait_for_servers || exit 1
    
    # Step 1: Signup (form-encoded)
    log_info "Step 1: User Signup"
    local signup_data="fullName=${TEST_FULLNAME}&email=${TEST_EMAIL}&newPassword=${TEST_PASSWORD}&responseType=spa&verifyType=email"
    local signup_response=$(curl -s -X POST "${AUTH_BASE}/signup" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$signup_data")
    
    local verification_id=$(extract_json_field "$signup_response" "verificationId")
    if [[ -z "$verification_id" ]]; then
        log_error "Failed to extract verification ID from signup response"
        echo "Response: $signup_response" >&2
        exit 1
    fi
    log_success "Verification ID: $verification_id"
    
    # Step 2: Get verification code from MailHog
    log_info "Step 2: Retrieving verification code from MailHog"
    sleep 3  # Wait for email to arrive
    local mailhog_response=$(get_latest_email_for_recipient "$TEST_EMAIL")
    local verification_code=$(extract_verification_code "$mailhog_response")
    
    if [[ -z "$verification_code" ]]; then
        log_error "Failed to extract verification code from email"
        echo "MailHog response: $mailhog_response" >&2
        exit 1
    fi
    log_success "Verification code: $verification_code"
    
    # Step 3: Verify email
    log_info "Step 3: Verifying email"
    local verify_data="verificationId=${verification_id}&code=${verification_code}&responseType=spa"
    local verify_response=$(curl -s -X POST "${AUTH_BASE}/signup/verify" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$verify_data")
    
    # Extract JWT token and user ID from verify response
    JWT_TOKEN=$(extract_json_field "$verify_response" "accessToken")
    if [[ -z "$JWT_TOKEN" ]]; then
        JWT_TOKEN=$(extract_json_field "$verify_response" "token")
    fi
    if [[ -z "$JWT_TOKEN" ]]; then
        JWT_TOKEN=$(extract_json_field "$verify_response" "access_token")
    fi
    
    USER_ID=$(extract_json_field "$verify_response" "user.objectId")
    if [[ -z "$USER_ID" ]]; then
        USER_ID=$(extract_json_field "$verify_response" "objectId")
    fi
    if [[ -z "$USER_ID" ]]; then
        USER_ID=$(extract_json_field "$verify_response" "userId")
    fi
    
    if [[ -z "$JWT_TOKEN" ]]; then
        log_warning "No JWT token from verify, will login separately"
    else
        log_success "JWT token obtained from verification (length: ${#JWT_TOKEN})"
    fi
    
    if [[ -z "$USER_ID" ]]; then
        log_error "Failed to extract user ID from verify response"
        echo "Response: $verify_response" >&2
        exit 1
    fi
    validate_uuid_format "$USER_ID" "User ID" || exit 1
    log_success "User verified: $USER_ID"
    
    # Step 4: Login (if we didn't get token from verify)
    if [[ -z "$JWT_TOKEN" ]]; then
        log_info "Step 4: User Login"
        local login_data="{\"username\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}"
        local login_response=$(make_request "POST" "${AUTH_BASE}/login" "$login_data" "" "200" "User login")
        
        JWT_TOKEN=$(extract_json_field "$login_response" "accessToken")
        if [[ -z "$JWT_TOKEN" ]]; then
            log_error "Failed to extract JWT token from login response"
            exit 1
        fi
        log_success "JWT token obtained from login (length: ${#JWT_TOKEN})"
    else
        log_info "Step 4: Skipping login (token obtained from verification)"
    fi
    
    # Step 5: Create Post
    log_info "Step 5: Creating test post"
    local post_data="{\"body\":\"Votes E2E Test Post - ${TIMESTAMP}\",\"postTypeId\":1}"
    local post_response=$(make_request "POST" "${POSTS_BASE}/" "$post_data" "Authorization: Bearer $JWT_TOKEN" "201" "Create post")
    
    TEST_POST_ID=$(extract_json_field "$post_response" "objectId")
    if [[ -z "$TEST_POST_ID" ]]; then
        log_error "Failed to extract post ID from create response"
        if [[ "$DEBUG_MODE" == "true" ]]; then
            echo "Response: $post_response" >&2
        fi
        exit 1
    fi
    validate_uuid_format "$TEST_POST_ID" "Post ID" || exit 1
    log_success "Post created: $TEST_POST_ID"
    
    # Step 6: Get initial post score
    log_info "Step 6: Getting initial post score"
    local get_post_response=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post")
    TEST_POST_SCORE=$(extract_json_field "$get_post_response" "score")
    TEST_POST_SCORE=${TEST_POST_SCORE:-0}
    log_info "Initial post score: $TEST_POST_SCORE"
    
    # Step 7: Test Case A - Vote Up with Authorization Header (Mobile App Style)
    log_info "Step 7: Test Case A - Vote Up with Authorization Header (Mobile App Style)"
    local vote_up_data="{\"postId\":\"$TEST_POST_ID\",\"typeId\":1}"
    make_request "POST" "${VOTES_BASE}/" "$vote_up_data" "Authorization: Bearer $JWT_TOKEN" "200" "Vote Up (Header Auth)" >/dev/null
    
    # Verify score increased
    sleep 1
    local post_after_up=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post after Up vote")
    local score_after_up=$(extract_json_field "$post_after_up" "score")
    score_after_up=${score_after_up:-0}
    
    local expected_score_up=$((TEST_POST_SCORE + 1))
    if [[ "$score_after_up" == "$expected_score_up" ]]; then
        log_success "✓ Post score is $score_after_up (expected $expected_score_up after Up vote)"
    else
        log_error "✗ Post score is $score_after_up (expected $expected_score_up after Up vote)"
        exit 1
    fi
    
    # Step 8: Test Case B - Vote Down with Cookie Auth (Web App Style)
    log_info "Step 8: Test Case B - Vote Down with Cookie Auth (Web App Style)"
    local vote_down_data="{\"postId\":\"$TEST_POST_ID\",\"typeId\":2}"
    # Per blueprint: Cookie name is "access_token"
    make_request "POST" "${VOTES_BASE}/" "$vote_down_data" "Cookie: access_token=$JWT_TOKEN" "200" "Vote Down (Cookie Auth)" >/dev/null
    
    # Verify score decreased (should be -1 from Up vote, so original score)
    sleep 1
    local post_after_down=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post after Down vote")
    local score_after_down=$(extract_json_field "$post_after_down" "score")
    score_after_down=${score_after_down:-0}
    
    # Expected: Up (+1) then Down (-1) = net change of 0, but Down vote switches from Up
    # So: Up (+1) -> score = original + 1, then Down (-1) switches from Up, so delta = -2
    # Final score = original + 1 - 2 = original - 1
    local expected_score_down=$((TEST_POST_SCORE - 1))
    if [[ "$score_after_down" == "$expected_score_down" ]]; then
        log_success "✓ Post score is $score_after_down (expected $expected_score_down after switching Up->Down)"
    else
        log_error "✗ Post score is $score_after_down (expected $expected_score_down after switching Up->Down)"
        exit 1
    fi
    
    # Step 9: Test Case C - Vote Up again with Cookie Auth (should switch from Down to Up)
    log_info "Step 9: Test Case C - Vote Up again with Cookie Auth (should switch from Down to Up)"
    local vote_up_again_data="{\"postId\":\"$TEST_POST_ID\",\"typeId\":1}"
    make_request "POST" "${VOTES_BASE}/" "$vote_up_again_data" "Cookie: access_token=$JWT_TOKEN" "200" "Vote Up again (Cookie Auth - switch from Down)" >/dev/null
    
    # Verify score switched: Down(-1) -> Up(+1) = delta +2, so score = -1 + 2 = 1
    sleep 1
    local post_after_switch=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post after switch")
    local score_after_switch=$(extract_json_field "$post_after_switch" "score")
    score_after_switch=${score_after_switch:-0}
    
    # Expected: After Down vote, score was -1. Switching to Up adds +2 (Up(+1) - Down(-1) = +2)
    # So final score = -1 + 2 = 1
    local expected_score_switch=1
    if [[ "$score_after_switch" == "$expected_score_switch" ]]; then
        log_success "✓ Post score is $score_after_switch (expected $expected_score_switch after switching Down->Up)"
    else
        log_error "✗ Post score is $score_after_switch (expected $expected_score_switch after switching Down->Up)"
        exit 1
    fi
    
    # Step 10: Test Case D - Vote Up again (should toggle off to Neutral)
    # CRITICAL: After Step 9, the vote is Up (typeId: 1). To toggle off, we must send the SAME type (1).
    # Backend toggles off when existing.VoteTypeID == voteType (same type sent again).
    log_info "Step 10: Test Case D - Vote Up again (should toggle off to Neutral)"
    
    # First, verify current state is Up (from Step 9)
    local post_before_toggle=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post before toggle off")
    local vote_type_before=$(extract_json_field "$post_before_toggle" "voteType")
    vote_type_before=${vote_type_before:-0}
    log_info "  Current voteType before toggle: $vote_type_before (should be 1=Up)"
    
    # Send the SAME type to toggle off (backend toggles when existing == new)
    local vote_toggle_data="{\"postId\":\"$TEST_POST_ID\",\"typeId\":1}"
    make_request "POST" "${VOTES_BASE}/" "$vote_toggle_data" "Cookie: access_token=$JWT_TOKEN" "200" "Vote Up again (Cookie Auth - toggle off)" >/dev/null
    
    # Verify score toggled off: Up(+1) -> Neutral(0) = delta -1, so score = 1 - 1 = 0
    sleep 1
    local post_after_toggle=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "Authorization: Bearer $JWT_TOKEN" "200" "Get post after toggle off")
    local score_after_toggle=$(extract_json_field "$post_after_toggle" "score")
    score_after_toggle=${score_after_toggle:-0}
    local vote_type_after=$(extract_json_field "$post_after_toggle" "voteType")
    vote_type_after=${vote_type_after:-0}
    
    # Expected: After Up vote, score was 1. Clicking Up again toggles off (-1 delta), score = 0
    local expected_score_toggle=0
    local expected_vote_type=0
    if [[ "$score_after_toggle" == "$expected_score_toggle" ]] && [[ "$vote_type_after" == "$expected_vote_type" ]]; then
        log_success "✓ Post score is $score_after_toggle (expected $expected_score_toggle after toggling off)"
        log_success "✓ Post voteType is $vote_type_after (expected $expected_vote_type after toggling off)"
    else
        log_error "✗ Post score is $score_after_toggle (expected $expected_score_toggle after toggling off)"
        log_error "✗ Post voteType is $vote_type_after (expected $expected_vote_type after toggling off)"
        log_error "  This indicates the toggle-off logic is broken (BUG: Same type sent again should toggle to Neutral)"
        exit 1
    fi
    
    cleanup
    print_summary
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Run main function
main "$@"

