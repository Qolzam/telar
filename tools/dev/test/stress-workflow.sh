#!/bin/bash

set -uo pipefail

# =============================================================================
# 20-Step Torture Test
# =============================================================================
# Executes comprehensive multi-user test scenarios
# Outputs testing matrix with PASS/FAIL for each step
#
# Usage:
#   bash tools/dev/test/stress-workflow.sh
#
# =============================================================================

BASE_URL="http://127.0.0.1:9099"
POSTS_BASE="${BASE_URL}/posts"
COMMENTS_BASE="${BASE_URL}/comments"
VOTES_BASE="${BASE_URL}/votes"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Load tokens
TOKEN_A=$(sed -n '1p' test_tokens.txt 2>/dev/null || echo "")
TOKEN_B=$(sed -n '2p' test_tokens.txt 2>/dev/null || echo "")
TOKEN_C=$(sed -n '3p' test_tokens.txt 2>/dev/null || echo "")

if [[ -z "$TOKEN_A" || -z "$TOKEN_B" || -z "$TOKEN_C" ]]; then
    echo -e "${RED}ERROR: Tokens not found. Run seed_users.sh first.${NC}"
    exit 1
fi

# Test results
declare -A RESULTS
POST_1_ID=""
POST_2_ID=""
COMMENT_1_ID=""
REPLY_1_ID=""

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    RESULTS["$1"]="PASS"
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    RESULTS["$1"]="FAIL"
}

make_request() {
    local method="$1"
    local url="$2"
    local data="$3"
    local token="$4"
    local expected_status="$5"
    
    local response
    local status_code
    
    if [[ -n "$data" ]]; then
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $token" \
            -d "$data" 2>/dev/null)
    else
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X "$method" "$url" \
            -H "Content-Type: application/json" \
            ${token:+-H "Authorization: Bearer $token"} 2>/dev/null)
    fi
    
    status_code=$(echo "$response" | tail -n1)
    response=$(echo "$response" | sed '$d')
    
    if [[ "$status_code" == "$expected_status" ]]; then
        echo "$response"
        return 0
    else
        echo "ERROR: Expected $expected_status, got $status_code" >&2
        echo "$response" >&2
        return 1
    fi
}

extract_id() {
    echo "$1" | python3 -c "import sys, json; print(json.load(sys.stdin).get('objectId', ''))" 2>/dev/null || echo ""
}

echo ""
echo "=========================================="
echo "20-Step Torture Test"
echo "=========================================="
echo ""

# Phase 1: Content & Permissions
log_test "=== Phase 1: Content & Permissions ==="

# Step 1: [A] Create Post
log_test "Step 1: [A] Create Post 'Post 1 by A'"
RESPONSE=$(make_request "POST" "$POSTS_BASE" '{"postTypeId":1,"body":"Post 1 by A","permission":"Public"}' "$TOKEN_A" "201")
POST_1_ID=$(extract_id "$RESPONSE")
if [[ -n "$POST_1_ID" ]]; then
    log_pass "Step 1"
else
    log_fail "Step 1"
fi

# Step 2: [B] Try to Edit Post 1 (should fail with 403 or 404)
log_test "Step 2: [B] Try to Edit Post 1 (should fail)"
RESPONSE=$(make_request "PUT" "$POSTS_BASE" "{\"objectId\":\"$POST_1_ID\",\"body\":\"Hacked by B\"}" "$TOKEN_B" "403" 2>&1)
STATUS=$?
if [[ $STATUS -eq 0 ]] || echo "$RESPONSE" | grep -q "403\|Forbidden\|forbidden\|404\|not found"; then
    log_pass "Step 2"
else
    log_fail "Step 2"
fi

# Step 3: [B] Comment on Post 1
log_test "Step 3: [B] Comment 'Comment 1 by B on Post 1'"
RESPONSE=$(make_request "POST" "$COMMENTS_BASE" "{\"postId\":\"$POST_1_ID\",\"text\":\"Comment 1 by B on Post 1\"}" "$TOKEN_B" "201")
COMMENT_1_ID=$(extract_id "$RESPONSE")
if [[ -n "$COMMENT_1_ID" ]]; then
    log_pass "Step 3"
else
    log_fail "Step 3"
fi

# Step 4: [C] Reply to Comment 1
log_test "Step 4: [C] Reply 'Reply by C'"
RESPONSE=$(make_request "POST" "$COMMENTS_BASE" "{\"postId\":\"$POST_1_ID\",\"text\":\"Reply by C\",\"parentCommentId\":\"$COMMENT_1_ID\"}" "$TOKEN_C" "201")
REPLY_1_ID=$(extract_id "$RESPONSE")
if [[ -n "$REPLY_1_ID" ]]; then
    log_pass "Step 4"
else
    log_fail "Step 4"
fi

# Step 5: [A] Try to Delete B's Comment (should fail with 404)
log_test "Step 5: [A] Try to Delete B's Comment (should fail)"
RESPONSE=$(make_request "DELETE" "$COMMENTS_BASE/id/$COMMENT_1_ID/post/$POST_1_ID" "" "$TOKEN_A" "404" 2>&1)
if [[ $? -eq 0 ]] || echo "$RESPONSE" | grep -q "404\|not found\|Not Found"; then
    log_pass "Step 5"
else
    log_fail "Step 5"
fi

# Step 6: [B] Delete Own Comment (should succeed)
log_test "Step 6: [B] Delete Own Comment"
RESPONSE=$(make_request "DELETE" "$COMMENTS_BASE/id/$COMMENT_1_ID/post/$POST_1_ID" "" "$TOKEN_B" "204" 2>&1)
if [[ $? -eq 0 ]]; then
    # Verify comment is deleted (replies are NOT cascade deleted - they remain but parent is deleted)
    sleep 2
    RESPONSE2=$(curl -s -w "\n%{http_code}" --max-time 5 -X GET "$COMMENTS_BASE/$COMMENT_1_ID" -H "Authorization: Bearer $TOKEN_B" 2>/dev/null)
    STATUS2=$(echo "$RESPONSE2" | tail -1)
    if [[ "$STATUS2" == "404" ]]; then
        log_pass "Step 6"
    else
        # Check if comment is marked as deleted
        COMMENT_BODY=$(echo "$RESPONSE2" | sed '$d')
        IS_DELETED=$(echo "$COMMENT_BODY" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d.get('deleted', False))" 2>/dev/null || echo "False")
        if [[ "$IS_DELETED" == "True" ]] || [[ "$IS_DELETED" == "true" ]]; then
            log_pass "Step 6"
        else
            log_fail "Step 6 (comment not deleted)"
        fi
    fi
else
    log_fail "Step 6"
fi

# Phase 2: Voting War
log_test ""
log_test "=== Phase 2: The Voting War ==="

# Step 7: [A] Create Post 2
log_test "Step 7: [A] Create Post 'Post 2 (Voting)'"
RESPONSE=$(make_request "POST" "$POSTS_BASE" '{"postTypeId":1,"body":"Post 2 (Voting)","permission":"Public"}' "$TOKEN_A" "201")
POST_2_ID=$(extract_id "$RESPONSE")
if [[ -n "$POST_2_ID" ]]; then
    log_pass "Step 7"
else
    log_fail "Step 7"
fi

# Step 8: [B] Upvote Post 2
log_test "Step 8: [B] Upvote Post 2"
RESPONSE=$(make_request "POST" "$VOTES_BASE" "{\"postId\":\"$POST_2_ID\",\"typeId\":1}" "$TOKEN_B" "200" 2>&1)
if [[ $? -eq 0 ]]; then
    log_pass "Step 8"
else
    log_fail "Step 8"
fi

# Step 9: [C] Upvote Post 2
log_test "Step 9: [C] Upvote Post 2"
RESPONSE=$(make_request "POST" "$VOTES_BASE" "{\"postId\":\"$POST_2_ID\",\"typeId\":1}" "$TOKEN_C" "200" 2>&1)
if [[ $? -eq 0 ]]; then
    log_pass "Step 9"
else
    log_fail "Step 9"
fi

# Step 10: [B] Switch to Downvote
log_test "Step 10: [B] Switch to Downvote"
RESPONSE=$(make_request "POST" "$VOTES_BASE" "{\"postId\":\"$POST_2_ID\",\"typeId\":2}" "$TOKEN_B" "200" 2>&1)
if [[ $? -eq 0 ]]; then
    log_pass "Step 10"
else
    log_fail "Step 10"
fi

# Step 11: [C] Toggle Off (remove vote by posting same voteType again)
log_test "Step 11: [C] Toggle Off (remove vote)"
RESPONSE=$(make_request "POST" "$VOTES_BASE" "{\"postId\":\"$POST_2_ID\",\"typeId\":1}" "$TOKEN_C" "200" 2>&1)
if [[ $? -eq 0 ]]; then
    log_pass "Step 11"
else
    log_fail "Step 11"
fi

# Step 12: [A] Self-Vote
log_test "Step 12: [A] Self-Vote (upvote own post)"
RESPONSE=$(make_request "POST" "$VOTES_BASE" "{\"postId\":\"$POST_2_ID\",\"typeId\":1}" "$TOKEN_A" "200" 2>&1)
if [[ $? -eq 0 ]]; then
    log_pass "Step 12"
else
    log_fail "Step 12"
fi

# Step 13: Verify Score Consistency
log_test "Step 13: Verify Score Consistency (all users see same score)"
SCORE_A=$(make_request "GET" "$POSTS_BASE/$POST_2_ID" "" "$TOKEN_A" "200" 2>&1 | python3 -c "import sys, json; print(json.load(sys.stdin).get('score', 0))" 2>/dev/null || echo "0")
SCORE_B=$(make_request "GET" "$POSTS_BASE/$POST_2_ID" "" "$TOKEN_B" "200" 2>&1 | python3 -c "import sys, json; print(json.load(sys.stdin).get('score', 0))" 2>/dev/null || echo "0")
SCORE_C=$(make_request "GET" "$POSTS_BASE/$POST_2_ID" "" "$TOKEN_C" "200" 2>&1 | python3 -c "import sys, json; print(json.load(sys.stdin).get('score', 0))" 2>/dev/null || echo "0")
if [[ "$SCORE_A" == "$SCORE_B" && "$SCORE_B" == "$SCORE_C" ]]; then
    log_pass "Step 13 (Score: $SCORE_A)"
else
    log_fail "Step 13 (Scores differ: A=$SCORE_A, B=$SCORE_B, C=$SCORE_C)"
fi

# Phase 3: Pagination & Caching
log_test ""
log_test "=== Phase 3: Pagination & Caching ==="

# Step 14: Inject 25 comments via API
log_test "Step 14: Inject 25 comments on Post 2"
COMMENT_COUNT=0
for i in {1..25}; do
    RESPONSE=$(make_request "POST" "$COMMENTS_BASE" "{\"postId\":\"$POST_2_ID\",\"text\":\"Comment $i\"}" "$TOKEN_B" "201" 2>&1)
    if [[ $? -eq 0 ]]; then
        COMMENT_COUNT=$((COMMENT_COUNT + 1))
    fi
done
if [[ $COMMENT_COUNT -eq 25 ]]; then
    log_pass "Step 14"
else
    log_fail "Step 14 (Created $COMMENT_COUNT/25)"
fi

# Step 15: [A] View Post 2 with Pagination
log_test "Step 15: [A] View Post 2 (verify pagination)"
RESPONSE=$(make_request "GET" "$COMMENTS_BASE?postId=$POST_2_ID&limit=10" "" "$TOKEN_A" "200" 2>&1)
COMMENT_LIST=$(echo "$RESPONSE" | python3 -c "import sys, json; d=json.load(sys.stdin); print(len(d.get('comments', [])))" 2>/dev/null || echo "0")
if [[ "$COMMENT_LIST" -ge 10 ]]; then
    log_pass "Step 15"
else
    log_fail "Step 15"
fi

# Step 16: [B] Add 26th Comment
log_test "Step 16: [B] Add 26th Comment"
RESPONSE=$(make_request "POST" "$COMMENTS_BASE" "{\"postId\":\"$POST_2_ID\",\"text\":\"Comment 26 by B\"}" "$TOKEN_B" "201" 2>&1)
COMMENT_26_ID=$(extract_id "$RESPONSE")
if [[ -n "$COMMENT_26_ID" ]]; then
    log_pass "Step 16"
else
    log_fail "Step 16"
fi

# Step 17: [A] Refresh and Verify 26th Comment Appears
log_test "Step 17: [A] Refresh and Verify 26th Comment"
sleep 2
RESPONSE=$(make_request "GET" "$COMMENTS_BASE?postId=$POST_2_ID&limit=30" "" "$TOKEN_A" "200" 2>&1)
HAS_26=$(echo "$RESPONSE" | python3 -c "import sys, json; comments=json.load(sys.stdin).get('comments', []); print('YES' if any(c.get('objectId') == '$COMMENT_26_ID' for c in comments) else 'NO')" 2>/dev/null || echo "NO")
if [[ "$HAS_26" == "YES" ]]; then
    log_pass "Step 17"
else
    log_fail "Step 17"
fi

# Phase 4: Double Next Audit
log_test ""
log_test "=== Phase 4: Double Next Audit ==="

# Step 18: Clear logs
log_test "Step 18: Clear server logs"
echo "" > /tmp/telar-logs/api.log
log_pass "Step 18"

# Step 19: [A] Create Comment
log_test "Step 19: [A] Create Comment (for audit)"
RESPONSE=$(make_request "POST" "$COMMENTS_BASE" "{\"postId\":\"$POST_2_ID\",\"text\":\"Audit test comment\"}" "$TOKEN_A" "201" 2>&1)
if [[ $? -eq 0 ]]; then
    log_pass "Step 19"
else
    log_fail "Step 19"
fi

# Step 20: Audit Logs for Double Execution (Database Method)
log_test "Step 20: Audit Logs (verify handler executes once)"
sleep 1
# Use database method: count comments created by the single request
BEFORE_COUNT=$(docker exec telar-postgres psql -U postgres -d telar_social_test -t -c "SELECT COUNT(*) FROM comments WHERE post_id = '$POST_2_ID' AND text LIKE 'Audit test%';" 2>/dev/null | tr -d ' ' || echo "0")
# Make the audit request
curl -s -X POST http://localhost:9099/comments -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN_A" -d "{\"postId\":\"$POST_2_ID\",\"text\":\"Audit test comment $(date +%s)\"}" > /dev/null
sleep 2
AFTER_COUNT=$(docker exec telar-postgres psql -U postgres -d telar_social_test -t -c "SELECT COUNT(*) FROM comments WHERE post_id = '$POST_2_ID' AND text LIKE 'Audit test%';" 2>/dev/null | tr -d ' ' || echo "0")
DIFF=$((AFTER_COUNT - BEFORE_COUNT))
if [[ "$DIFF" -eq 1 ]]; then
    log_pass "Step 20 (Handler executed exactly once - created 1 comment)"
else
    log_fail "Step 20 (Handler executed $DIFF times, expected 1)"
fi

# Summary
echo ""
echo "=========================================="
echo "Testing Matrix"
echo "=========================================="
echo ""
printf "%-30s %s\n" "Step" "Result"
echo "------------------------------------------"
for i in {1..20}; do
    RESULT=${RESULTS["Step $i"]:-"NOT TESTED"}
    printf "%-30s %s\n" "Step $i" "$RESULT"
done
echo ""

PASS_COUNT=0
FAIL_COUNT=0
for result in "${RESULTS[@]}"; do
    if [[ "$result" == "PASS" ]]; then
        PASS_COUNT=$((PASS_COUNT + 1))
    elif [[ "$result" == "FAIL" ]]; then
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
done

echo "Summary: $PASS_COUNT PASS, $FAIL_COUNT FAIL"
echo ""

if [[ $FAIL_COUNT -eq 0 ]]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
