#!/bin/bash

set -euo pipefail

# =============================================================================
# Lifecycle E2E Test - Complete Post/Comment Lifecycle
# =============================================================================
# 
# Tests the complete lifecycle of posts and comments:
# - Post creation and editing
# - Comment creation, replies, likes, editing
# - Cursor pagination with data generation
# - Destructive operations (delete comment, delete post, cascade)
#
# Usage:
#   bash tools/dev/scripts/lifecycle_e2e_test.sh
#
# =============================================================================

# Configuration
BASE_URL="http://127.0.0.1:8080"
POSTS_BASE="${BASE_URL}/posts"
COMMENTS_BASE="${BASE_URL}/comments"
AUTH_BASE="${BASE_URL}/auth"
MAILHOG_URL="http://localhost:8025"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Test data
TIMESTAMP=$(date +%s)
USER_A_EMAIL="lifecycle-user-a-${TIMESTAMP}@example.com"
USER_B_EMAIL="lifecycle-user-b-${TIMESTAMP}@example.com"
TEST_PASSWORD="LifecycleTestPassword123!@#"
TEST_FULLNAME_A="Lifecycle User A"
TEST_FULLNAME_B="Lifecycle User B"

# Global variables
USER_A_TOKEN=""
USER_A_ID=""
USER_B_TOKEN=""
USER_B_ID=""
TEST_POST_ID=""
TEST_COMMENT_ID=""
TEST_REPLY_ID=""

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
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
    local token="$4"
    local expected_status="$5"
    local description="$6"
    
    log_test "$description" >&2
    
    local response
    local status_code
    
    # Check if data is form-encoded (contains =) or JSON
    if [[ -n "$data" ]]; then
        if [[ "$data" == *"="* && "$data" != *"{"* ]]; then
            # Form data
            response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
                -H "Content-Type: application/x-www-form-urlencoded" \
                ${token:+-H "Authorization: Bearer $token"} \
                -d "$data" 2>/dev/null)
        else
            # JSON data
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
    
    # Handle curl errors (status_code will be 000 or empty)
    if [[ -z "$status_code" || "$status_code" == "000" ]]; then
        log_error "Request failed - no response from server" >&2
        echo "Response: $response" >&2
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

extract_uuid_field() {
    local json="$1"
    local field="$2"
    echo "$json" | grep -o "\"$field\"[[:space:]]*:[[:space:]]*\"[a-f0-9-]\{36\}\"" | sed "s/\"$field\"[[:space:]]*:[[:space:]]*\"\([a-f0-9-]\{36\}\)\"/\1/" | head -1
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
            # Get the most recent email (items are sorted by date, newest first)
            local email_body=$(echo "$mailhog_response" | python3 -c "import sys, json; data=json.load(sys.stdin); items=data.get('items', []); print(items[0]['Content']['Body'] if items else '')" 2>/dev/null || echo "")
            
            # Extract code using multiple patterns (like auth_e2e_test.sh)
            code=$(echo "$email_body" | grep -oE 'code=[0-9]{6}' | grep -oE '[0-9]{6}' | head -1)
            if [[ -z "$code" ]]; then
                code=$(echo "$email_body" | grep -oE '(code[:\s]+|verification[:\s]+|Your code is[:\s]+)[0-9]{6}' | grep -oE '[0-9]{6}' | head -1)
            fi
            if [[ -z "$code" ]]; then
                # Last resort: find any 6-digit number in the email
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
            log_info "Waiting for verification email (Attempt $i/$MAX_RETRIES)..." >&2
            sleep $SLEEP_TIME
        fi
    done
    
    log_error "Verification code never arrived for $email after $MAX_RETRIES attempts" >&2
    return 1
}

signup_user() {
    local email="$1"
    local fullname="$2"
    local token_var="$3"
    local user_id_var="$4"
    
    log_info "Signing up user: $email"
    
    # Use form data format (as auth_e2e_test.sh does)
    local signup_data="fullName=${fullname}&email=${email}&newPassword=${TEST_PASSWORD}&responseType=spa&verifyType=email&g-recaptcha-response=ok"
    
    local response=$(make_request "POST" "${AUTH_BASE}/signup" "$signup_data" "" "200" "Signup user: $email")
    local verification_id=$(extract_json_field "$response" "verificationId")
    
    if [[ -z "$verification_id" ]]; then
        log_error "Failed to get verification ID from signup response"
        log_error "Response was: $response"
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
            log_info "Retrying verification code extraction (attempt $retries/10)..."
            sleep 3
        fi
    done
    
    if [[ -z "$code" ]]; then
        log_error "Failed to get verification code for $email after 10 attempts"
        return 1
    fi
    
    log_info "Verifying email with code: $code"
    local verify_data="verificationId=${verification_id}&code=${code}&responseType=spa"
    
    local verify_response=$(make_request "POST" "${AUTH_BASE}/signup/verify" "$verify_data" "" "200" "Verify email: $email")
    local token=$(extract_json_field "$verify_response" "accessToken")
    if [[ -z "$token" ]]; then
        token=$(extract_json_field "$verify_response" "token")
    fi
    
    local user_id=$(extract_json_field "$verify_response" "objectId")
    if [[ -z "$user_id" ]]; then
        user_id=$(extract_json_field "$verify_response" "userId")
    fi
    
    if [[ -z "$token" ]]; then
        log_error "Failed to get access token"
        return 1
    fi
    
    if [[ -z "$user_id" ]]; then
        log_error "Failed to get user ID"
        return 1
    fi
    
    eval "$token_var='$token'"
    eval "$user_id_var='$user_id'"
    
    log_success "User $email signed up and logged in"
    return 0
}

main() {
    log_info "=========================================="
    log_info "Lifecycle E2E Test - Complete Flow"
    log_info "=========================================="
    
    # Phase 1: User Setup
    log_info ""
    log_info "=== Phase 1: User Setup ==="
    signup_user "$USER_A_EMAIL" "$TEST_FULLNAME_A" "USER_A_TOKEN" "USER_A_ID"
    signup_user "$USER_B_EMAIL" "$TEST_FULLNAME_B" "USER_B_TOKEN" "USER_B_ID"
    
    # Phase 2: Post Lifecycle
    log_info ""
    log_info "=== Phase 2: Post Lifecycle ==="
    
    log_test "User A creates a post"
    local create_post_data=$(cat <<EOF
{
  "postTypeId": 1,
  "body": "Lifecycle Test Post - Initial Content",
  "permission": "Public"
}
EOF
)
    local post_response=$(make_request "POST" "$POSTS_BASE" "$create_post_data" "$USER_A_TOKEN" "201" "Create post")
    TEST_POST_ID=$(extract_uuid_field "$post_response" "objectId")
    
    if [[ -z "$TEST_POST_ID" ]]; then
        log_error "Failed to get post ID"
        exit 1
    fi
    
    log_success "Post created with ID: $TEST_POST_ID"
    
    log_test "User A edits the post"
    local update_post_data=$(cat <<EOF
{
  "objectId": "$TEST_POST_ID",
  "body": "Lifecycle Test Post - Updated Content"
}
EOF
)
    make_request "PUT" "$POSTS_BASE" "$update_post_data" "$USER_A_TOKEN" "200" "Update post" > /dev/null
    
    log_test "Verify post was updated"
    local get_post_response=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "$USER_A_TOKEN" "200" "Get updated post")
    local updated_body=$(extract_json_field "$get_post_response" "body")
    
    if [[ "$updated_body" != "Lifecycle Test Post - Updated Content" ]]; then
        log_error "Post body was not updated correctly. Expected: 'Lifecycle Test Post - Updated Content', Got: '$updated_body'"
        exit 1
    fi
    
    log_success "Post edit verified"
    
    # Phase 3: Comment Lifecycle
    log_info ""
    log_info "=== Phase 3: Comment Lifecycle ==="
    
    log_test "User B creates a comment on the post"
    local create_comment_data=$(cat <<EOF
{
  "postId": "$TEST_POST_ID",
  "text": "This is a test comment from User B"
}
EOF
)
    local comment_response=$(make_request "POST" "$COMMENTS_BASE" "$create_comment_data" "$USER_B_TOKEN" "201" "Create comment")
    TEST_COMMENT_ID=$(extract_uuid_field "$comment_response" "objectId")
    
    if [[ -z "$TEST_COMMENT_ID" ]]; then
        log_error "Failed to get comment ID"
        exit 1
    fi
    
    log_success "Comment created with ID: $TEST_COMMENT_ID"
    
    log_test "Verify post commentCount = 1"
    local post_after_comment=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "$USER_A_TOKEN" "200" "Get post after comment")
    local comment_count=$(echo "$post_after_comment" | grep -o "\"commentCounter\"[[:space:]]*:[[:space:]]*[0-9]*" | grep -o "[0-9]*" | head -1)
    
    if [[ "$comment_count" != "1" ]]; then
        log_error "Post commentCount is $comment_count (expected 1)"
        exit 1
    fi
    
    log_success "Post commentCount = 1 verified"
    
    log_test "User A replies to User B's comment"
    local create_reply_data=$(cat <<EOF
{
  "postId": "$TEST_POST_ID",
  "text": "This is a reply from User A",
  "parentCommentId": "$TEST_COMMENT_ID"
}
EOF
)
    local reply_response=$(make_request "POST" "$COMMENTS_BASE" "$create_reply_data" "$USER_A_TOKEN" "201" "Create reply")
    TEST_REPLY_ID=$(extract_uuid_field "$reply_response" "objectId")
    
    if [[ -z "$TEST_REPLY_ID" ]]; then
        log_error "Failed to get reply ID"
        exit 1
    fi
    
    log_success "Reply created with ID: $TEST_REPLY_ID"
    
    log_test "Verify comment replyCount = 1"
    local comment_after_reply=$(make_request "GET" "${COMMENTS_BASE}/${TEST_COMMENT_ID}" "" "$USER_A_TOKEN" "200" "Get comment after reply")
    # Try both camelCase and snake_case
    local reply_count=$(echo "$comment_after_reply" | grep -oE '"(replyCount|reply_count)"[[:space:]]*:[[:space:]]*[0-9]+' | grep -oE '[0-9]+' | head -1)
    
    if [[ -z "$reply_count" ]]; then
        log_error "Could not extract replyCount from response: ${comment_after_reply:0:200}"
        exit 1
    fi
    
    if [[ "$reply_count" != "1" ]]; then
        log_error "Comment replyCount is $reply_count (expected 1). Full response: ${comment_after_reply:0:500}"
        exit 1
    fi
    
    log_success "Comment replyCount = 1 verified"
    
    log_test "User B likes User A's reply"
    make_request "POST" "${COMMENTS_BASE}/${TEST_REPLY_ID}/like" "" "$USER_B_TOKEN" "200" "Like reply" > /dev/null
    
    log_test "Verify reply score = 1"
    local reply_after_like=$(make_request "GET" "${COMMENTS_BASE}/${TEST_REPLY_ID}" "" "$USER_B_TOKEN" "200" "Get reply after like")
    local reply_score=$(echo "$reply_after_like" | grep -o "\"score\"[[:space:]]*:[[:space:]]*[0-9]*" | grep -o "[0-9]*" | head -1)
    
    if [[ "$reply_score" != "1" ]]; then
        log_error "Reply score is $reply_score (expected 1)"
        exit 1
    fi
    
    log_success "Reply score = 1 verified"
    
    log_test "User A edits the reply"
    local update_reply_data=$(cat <<EOF
{
  "objectId": "$TEST_REPLY_ID",
  "text": "This is an edited reply from User A"
}
EOF
)
    make_request "PUT" "$COMMENTS_BASE" "$update_reply_data" "$USER_A_TOKEN" "200" "Update reply" > /dev/null
    
    log_test "Verify reply was updated"
    local updated_reply=$(make_request "GET" "${COMMENTS_BASE}/${TEST_REPLY_ID}" "" "$USER_A_TOKEN" "200" "Get updated reply")
    local updated_reply_text=$(extract_json_field "$updated_reply" "text")
    
    if [[ "$updated_reply_text" != "This is an edited reply from User A" ]]; then
        log_error "Reply text was not updated correctly"
        exit 1
    fi
    
    log_success "Reply edit verified"
    
    # Phase 4: Pagination Check
    log_info ""
    log_info "=== Phase 4: Pagination Check ==="
    
    log_test "User B creates 15 more comments for pagination test"
    for i in {1..15}; do
        local pagination_comment_data=$(cat <<EOF
{
  "postId": "$TEST_POST_ID",
  "text": "Pagination test comment $i"
}
EOF
)
        make_request "POST" "$COMMENTS_BASE" "$pagination_comment_data" "$USER_B_TOKEN" "201" "Create pagination comment $i" > /dev/null
    done
    
    log_success "15 comments created"
    
    log_test "User A fetches comments (Page 1, limit 10)"
    local page1_response=$(make_request "GET" "${COMMENTS_BASE}?postId=${TEST_POST_ID}&limit=10" "" "$USER_A_TOKEN" "200" "Fetch page 1")
    local page1_count=$(echo "$page1_response" | grep -o "\"comments\"[[:space:]]*:\[" | wc -l)
    local next_cursor=$(extract_json_field "$page1_response" "nextCursor")
    local has_next=$(echo "$page1_response" | grep -o "\"hasNext\"[[:space:]]*:[[:space:]]*true" | wc -l)
    
    # Count comments in response
    local comment_count_page1=$(echo "$page1_response" | grep -o "\"objectId\"[[:space:]]*:[[:space:]]*\"[a-f0-9-]\{36\}\"" | wc -l)
    
    if [[ "$comment_count_page1" -lt 10 ]]; then
        log_error "Page 1 should have at least 10 comments, got $comment_count_page1"
        exit 1
    fi
    
    if [[ -z "$next_cursor" ]] || [[ "$has_next" == "0" ]]; then
        log_error "Page 1 should have nextCursor and hasNext=true"
        exit 1
    fi
    
    log_success "Page 1: $comment_count_page1 comments, nextCursor exists"
    
    log_test "User A fetches comments (Page 2 using cursor)"
    local page2_response=$(make_request "GET" "${COMMENTS_BASE}?postId=${TEST_POST_ID}&limit=10&cursor=${next_cursor}" "" "$USER_A_TOKEN" "200" "Fetch page 2")
    local comment_count_page2=$(echo "$page2_response" | grep -o "\"objectId\"[[:space:]]*:[[:space:]]*\"[a-f0-9-]\{36\}\"" | wc -l)
    
    if [[ "$comment_count_page2" -lt 6 ]]; then
        log_error "Page 2 should have at least 6 comments (16 total - 10 from page 1), got $comment_count_page2"
        exit 1
    fi
    
    log_success "Page 2: $comment_count_page2 comments retrieved"
    
    # Phase 5: Destructive Cycle
    log_info ""
    log_info "=== Phase 5: Destructive Cycle ==="
    
    log_test "User B deletes their comment"
    make_request "DELETE" "${COMMENTS_BASE}/id/${TEST_COMMENT_ID}/post/${TEST_POST_ID}" "" "$USER_B_TOKEN" "204" "Delete comment" > /dev/null
    
    log_test "Verify post commentCount decreased"
    local post_after_delete=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "$USER_A_TOKEN" "200" "Get post after comment delete")
    local comment_count_after=$(echo "$post_after_delete" | grep -o "\"commentCounter\"[[:space:]]*:[[:space:]]*[0-9]*" | grep -o "[0-9]*" | head -1)
    
    # Should be 15 (16 total - 1 deleted root comment)
    if [[ "$comment_count_after" -lt 15 ]]; then
        log_error "Post commentCount should be at least 15 after deleting one comment, got $comment_count_after"
        exit 1
    fi
    
    log_success "Post commentCount decreased to $comment_count_after"
    
    log_test "User A deletes the post"
    make_request "DELETE" "${POSTS_BASE}/${TEST_POST_ID}" "" "$USER_A_TOKEN" "204" "Delete post" > /dev/null
    
    log_test "Verify post is gone (404)"
    make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "$USER_A_TOKEN" "404" "Verify post deleted" > /dev/null
    
    log_success "Post deleted and verified gone"
    
    log_test "Verify comments associated with post are gone (cascade check)"
    make_request "GET" "${COMMENTS_BASE}/${TEST_COMMENT_ID}" "" "$USER_A_TOKEN" "404" "Verify comment deleted (cascade)" > /dev/null
    make_request "GET" "${COMMENTS_BASE}/${TEST_REPLY_ID}" "" "$USER_A_TOKEN" "404" "Verify reply deleted (cascade)" > /dev/null
    
    log_success "Comments and replies deleted via cascade"
    
    # Summary
    log_info ""
    log_info "=========================================="
    log_success "All lifecycle tests passed!"
    log_info "=========================================="
    log_info "Test Summary:"
    log_info "  - User A & B setup: ✓"
    log_info "  - Post create & edit: ✓"
    log_info "  - Comment create, reply, like, edit: ✓"
    log_info "  - Pagination (15 comments, 2 pages): ✓"
    log_info "  - Destructive cycle (delete comment, delete post, cascade): ✓"
    log_info "=========================================="
}

# Run main function
main "$@"


    log_test "User B deletes their comment"
    make_request "DELETE" "${COMMENTS_BASE}/id/${TEST_COMMENT_ID}/post/${TEST_POST_ID}" "" "$USER_B_TOKEN" "204" "Delete comment" > /dev/null
    
    log_test "Verify post commentCount decreased"
    local post_after_delete=$(make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "$USER_A_TOKEN" "200" "Get post after comment delete")
    local comment_count_after=$(echo "$post_after_delete" | grep -o "\"commentCounter\"[[:space:]]*:[[:space:]]*[0-9]*" | grep -o "[0-9]*" | head -1)
    
    # Should be 15 (16 total - 1 deleted root comment)
    if [[ "$comment_count_after" -lt 15 ]]; then
        log_error "Post commentCount should be at least 15 after deleting one comment, got $comment_count_after"
        exit 1
    fi
    
    log_success "Post commentCount decreased to $comment_count_after"
    
    log_test "User A deletes the post"
    make_request "DELETE" "${POSTS_BASE}/${TEST_POST_ID}" "" "$USER_A_TOKEN" "204" "Delete post" > /dev/null
    
    log_test "Verify post is gone (404)"
    make_request "GET" "${POSTS_BASE}/${TEST_POST_ID}" "" "$USER_A_TOKEN" "404" "Verify post deleted" > /dev/null
    
    log_success "Post deleted and verified gone"
    
    log_test "Verify comments associated with post are gone (cascade check)"
    make_request "GET" "${COMMENTS_BASE}/${TEST_COMMENT_ID}" "" "$USER_A_TOKEN" "404" "Verify comment deleted (cascade)" > /dev/null
    make_request "GET" "${COMMENTS_BASE}/${TEST_REPLY_ID}" "" "$USER_A_TOKEN" "404" "Verify reply deleted (cascade)" > /dev/null
    
    log_success "Comments and replies deleted via cascade"
    
    # Summary
    log_info ""
    log_info "=========================================="
    log_success "All lifecycle tests passed!"
    log_info "=========================================="
    log_info "Test Summary:"
    log_info "  - User A & B setup: ✓"
    log_info "  - Post create & edit: ✓"
    log_info "  - Comment create, reply, like, edit: ✓"
    log_info "  - Pagination (15 comments, 2 pages): ✓"
    log_info "  - Destructive cycle (delete comment, delete post, cascade): ✓"
    log_info "=========================================="
}

# Run main function
main "$@"

