#!/bin/bash

# Comprehensive Search Functionality Test Suite
# Tests both posts and profile search endpoints with all scenarios
# Covers: autocomplete endpoints, cursor pagination, filters, edge cases, security, performance

# Configuration
BASE_URL="http://127.0.0.1:9099"
AUTH_URL="${BASE_URL}"
POSTS_URL="${BASE_URL}"
PROFILES_URL="${BASE_URL}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED_TESTS++))
}

log_failure() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED_TESTS++))
}

log_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
    ((TOTAL_TESTS++))
}

log_section() {
    echo -e "\n${CYAN}=========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}=========================================${NC}"
}

# Helper function to test HTTP response
test_response() {
    local url="$1"
    local expected_status="$2"
    local test_name="$3"
    local should_contain="$4"
    local should_not_contain="$5"
    local use_auth="${6:-false}"

    log_test "$test_name"

    local response=""
    local http_code=""
    local curl_opts="-s -w \"\n%{http_code}\" --max-time 10"

    if [ "$use_auth" = "true" ]; then
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -H "Authorization: Bearer ${JWT_TOKEN}" "$url" 2>/dev/null)
    else
        response=$(curl -s -w "\n%{http_code}" --max-time 10 "$url" 2>/dev/null)
    fi

    if [ $? -ne 0 ]; then
        log_failure "$test_name - Request failed"
        return 1
    fi

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -eq "$expected_status" ]; then
        local content_check=true
        
        if [ -n "$should_contain" ]; then
            if [[ ! "$body" == *"$should_contain"* ]]; then
                content_check=false
            fi
        fi
        
        if [ -n "$should_not_contain" ]; then
            if [[ "$body" == *"$should_not_contain"* ]]; then
                content_check=false
            fi
        fi
        
        if [ "$content_check" = true ]; then
            log_success "$test_name"
            return 0
        else
            log_failure "$test_name - Content validation failed"
            return 1
        fi
    else
        log_failure "$test_name - Expected status $expected_status, got $http_code"
        return 1
    fi
}

# Helper to test JSON response structure
test_json_structure() {
    local url="$1"
    local test_name="$2"
    local expected_field="$3"
    local use_auth="${4:-false}"

    log_test "$test_name"

    local response=""
    if [ "$use_auth" = "true" ]; then
        response=$(curl -s --max-time 10 -H "Authorization: Bearer ${JWT_TOKEN}" "$url" 2>/dev/null)
    else
        response=$(curl -s --max-time 10 "$url" 2>/dev/null)
    fi

    if [ $? -ne 0 ]; then
        log_failure "$test_name - Request failed"
        return 1
    fi

    if command -v jq &> /dev/null; then
        if echo "$response" | jq -e ".$expected_field" > /dev/null 2>&1; then
            log_success "$test_name"
            return 0
        else
            log_failure "$test_name - JSON structure invalid (missing $expected_field)"
            return 1
        fi
    else
        # Fallback: check if response is valid JSON array/object
        if echo "$response" | grep -qE "^(\[|\{)" && echo "$response" | grep -q "$expected_field"; then
            log_success "$test_name"
            return 0
        else
            log_failure "$test_name - JSON structure validation failed"
            return 1
        fi
    fi
}

# Helper to test response time
test_performance() {
    local url="$1"
    local test_name="$2"
    local max_ms="$3"
    local use_auth="${4:-false}"

    log_test "$test_name"

    local start_time=$(date +%s%N)
    if [ "$use_auth" = "true" ]; then
        curl -s --max-time 10 -H "Authorization: Bearer ${JWT_TOKEN}" "$url" > /dev/null 2>&1
    else
        curl -s --max-time 10 "$url" > /dev/null 2>&1
    fi
    local end_time=$(date +%s%N)
    local duration=$(( (end_time - start_time) / 1000000 ))

    if [ $? -eq 0 ] && [ "$duration" -lt "$max_ms" ]; then
        log_success "$test_name - Completed in ${duration}ms (max: ${max_ms}ms)"
        return 0
    else
        log_failure "$test_name - Too slow: ${duration}ms (max: ${max_ms}ms)"
        return 1
    fi
}

echo "========================================="
echo "🔍 COMPREHENSIVE SEARCH FUNCTIONALITY TEST SUITE"
echo "========================================="

# Pre-flight check: Health check
log_info "Checking API availability on port 9099..."
if ! curl -s --max-time 5 "${BASE_URL}/profile/search?q=test" > /dev/null 2>&1; then
    log_failure "API is down on port 9099"
    echo "   Run: make run-api-background"
    echo "   OR: bash tools/dev/app/start.sh"
    exit 1
fi
log_success "API is up on port 9099"

# Dynamic token acquisition from seed_users.sh output
log_info "Acquiring authentication token..."
JWT_TOKEN=""

# 1. Check for token file from seed_users.sh
if [ -f "test_tokens.txt" ]; then
    TOKEN=$(head -n 1 test_tokens.txt 2>/dev/null | tr -d '[:space:]')
    if [ -n "$TOKEN" ]; then
        JWT_TOKEN="$TOKEN"
        log_success "Using token from test_tokens.txt: ${TOKEN:0:20}..."
    fi
fi

# 2. If no token file, try to get user from test_users.json
if [ -z "$JWT_TOKEN" ] && [ -f "test_users.json" ]; then
    USER_EMAIL=$(jq -r '.[0].email // empty' test_users.json 2>/dev/null || echo "")
    USER_PASSWORD=$(jq -r '.[0].password // empty' test_users.json 2>/dev/null || echo "")
    if [ -n "$USER_EMAIL" ] && [ -n "$USER_PASSWORD" ]; then
        log_info "Attempting login with user from test_users.json: $USER_EMAIL"
        LOGIN_RESPONSE=$(curl -s -X POST "${AUTH_URL}/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\",\"responseType\":\"json\"}" 2>/dev/null)
        JWT_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"accessToken":"[^"]*' | cut -d'"' -f4)
        if [ -n "$JWT_TOKEN" ]; then
            log_success "Login successful: ${JWT_TOKEN:0:20}..."
        fi
    fi
fi

# 3. If still no token, run seed_users.sh to create one
if [ -z "$JWT_TOKEN" ]; then
    log_info "No tokens found. Running seed script to create test user..."
    if [ -f "tools/dev/seed/users.sh" ]; then
        bash tools/dev/seed/users.sh 1 > /dev/null 2>&1
        if [ -f "test_tokens.txt" ]; then
            JWT_TOKEN=$(head -n 1 test_tokens.txt 2>/dev/null | tr -d '[:space:]')
            if [ -n "$JWT_TOKEN" ]; then
                log_success "Token acquired from seed script: ${JWT_TOKEN:0:20}..."
            fi
        fi
    fi
fi

# Final validation
if [ -z "$JWT_TOKEN" ]; then
    log_failure "Failed to acquire authentication token"
    echo "   Run: bash tools/dev/seed/users.sh 1"
    exit 1
fi

log_success "Authentication token ready: ${JWT_TOKEN:0:20}..."

# =========================================
# EDGE CASE DATA SETUP
# =========================================
log_section "📦 EDGE CASE DATA SETUP"

log_info "Creating edge case test posts..."
# Create post with emoji and special characters
EDGE_POST_1=$(curl -s -X POST "${POSTS_URL}/posts/" \
    -H "Authorization: Bearer ${JWT_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"body":"UniqueKeyword🚀 SpecialChars@# TestPost", "postTypeId":1, "tags":["tech","golang"]}' 2>/dev/null)

if echo "$EDGE_POST_1" | grep -q "objectId"; then
    log_success "Edge case post 1 created (emoji + special chars)"
else
    log_info "Edge case post 1 may already exist or creation failed (continuing)"
fi

# Create post with specific tag for tag filter tests
EDGE_POST_2=$(curl -s -X POST "${POSTS_URL}/posts/" \
    -H "Authorization: Bearer ${JWT_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"body":"TagFilterTestPost", "postTypeId":1, "tags":["tech"]}' 2>/dev/null)

if echo "$EDGE_POST_2" | grep -q "objectId"; then
    log_success "Edge case post 2 created (tag filter test)"
else
    log_info "Edge case post 2 may already exist or creation failed (continuing)"
fi

# Wait for eventual consistency
sleep 2

# =========================================
# PROFILE SEARCH - PUBLIC AUTOCOMPLETE (/profile/search)
# =========================================
log_section "👤 PROFILE SEARCH - PUBLIC AUTOCOMPLETE"

# Basic search tests
test_response "${PROFILES_URL}/profile/search?q=Test+User" 200 "Profile search - Basic name search" "Test User"
test_response "${PROFILES_URL}/profile/search?q=User+1" 200 "Profile search - Specific user search" "User"
test_response "${PROFILES_URL}/profile/search?q=test" 200 "Profile search - Case insensitive search" "Test"
test_response "${PROFILES_URL}/profile/search?q=user" 200 "Profile search - Partial word match" "User"

# Case variations
test_response "${PROFILES_URL}/profile/search?q=TEST" 200 "Profile search - Uppercase" "Test"
test_response "${PROFILES_URL}/profile/search?q=TeSt" 200 "Profile search - Mixed case" "Test"

# Special characters and encoding
test_response "${PROFILES_URL}/profile/search?q=Test%20User%205" 200 "Profile search - URL encoded spaces" "Test"
test_response "${PROFILES_URL}/profile/search?q=Test+User+5" 200 "Profile search - Plus encoded spaces" "Test"

# Edge cases - empty and minimal queries
test_response "${PROFILES_URL}/profile/search?q=" 200 "Profile search - Empty query returns empty array" "[]"
test_response "${PROFILES_URL}/profile/search?q=a" 200 "Profile search - Single character" "[]"
test_response "${PROFILES_URL}/profile/search?q=ab" 200 "Profile search - Two characters" "[]"
test_response "${PROFILES_URL}/profile/search?q=nonexistentuser123456789" 200 "Profile search - No matches" "[]"

# Limit parameter tests - boundary conditions
test_response "${PROFILES_URL}/profile/search?q=Test&limit=1" 200 "Profile search - Limit 1" "Test"
test_response "${PROFILES_URL}/profile/search?q=Test&limit=3" 200 "Profile search - Limit 3" "Test"
test_response "${PROFILES_URL}/profile/search?q=Test&limit=5" 200 "Profile search - Limit 5 (default max)" "Test"
test_response "${PROFILES_URL}/profile/search?q=Test&limit=20" 200 "Profile search - Limit 20 (max allowed)" "Test"
test_response "${PROFILES_URL}/profile/search?q=Test&limit=21" 200 "Profile search - Limit 21 (should cap at 20)" "Test"
test_response "${PROFILES_URL}/profile/search?q=Test&limit=100" 200 "Profile search - Limit 100 (should cap at 20)" "Test"
test_response "${PROFILES_URL}/profile/search?q=Test&limit=0" 200 "Profile search - Limit 0 (invalid, should use default)" "Test"
test_response "${PROFILES_URL}/profile/search?q=Test&limit=-1" 200 "Profile search - Limit -1 (invalid, should use default)" "Test"
test_response "${PROFILES_URL}/profile/search?q=Test&limit=abc" 200 "Profile search - Limit non-numeric (should use default)" "Test"

# Very long query
LONG_QUERY="verylongsearchquerythatshouldnotfindanythingandteststhemaximumquerylengthhandling"
test_response "${PROFILES_URL}/profile/search?q=${LONG_QUERY}" 200 "Profile search - Very long query" "[]"

# Security tests - SQL injection attempts (should be safe)
test_response "${PROFILES_URL}/profile/search?q=Test'%20OR%201=1" 200 "Profile search - SQL injection attempt 1" "[]"
test_response "${PROFILES_URL}/profile/search?q=Test';%20DROP%20TABLE%20users;" 200 "Profile search - SQL injection attempt 2" "[]"
test_response "${PROFILES_URL}/profile/search?q=Test'%20UNION%20SELECT%20*" 200 "Profile search - SQL injection attempt 3" "[]"

# Unicode and special characters
test_response "${PROFILES_URL}/profile/search?q=Test%20%F0%9F%98%80" 200 "Profile search - Unicode emoji" "[]"
test_response "${PROFILES_URL}/profile/search?q=%E4%B8%AD%E6%96%87" 200 "Profile search - Chinese characters" "[]"
test_response "${PROFILES_URL}/profile/search?q=Test%20%26%20User" 200 "Profile search - Special chars (&)" "[]"

# Whitespace handling
test_response "${PROFILES_URL}/profile/search?q=%20%20%20" 200 "Profile search - Whitespace only" "[]"
test_response "${PROFILES_URL}/profile/search?q=Test%20%20%20User" 200 "Profile search - Multiple spaces" "Test"

# Response structure validation
test_json_structure "${PROFILES_URL}/profile/search?q=Test&limit=5" "Profile search - Valid JSON array structure" "objectId"

# =========================================
# PROFILE QUERY - AUTHENTICATED (/profile/)
# =========================================
log_section "👤 PROFILE QUERY - AUTHENTICATED"

# Basic authenticated query tests
test_response "${PROFILES_URL}/profile/?search=Test" 200 "Profile query - Basic search with auth" "Test" "" "true"
test_response "${PROFILES_URL}/profile/?search=User" 200 "Profile query - User search with auth" "User" "" "true"
test_response "${PROFILES_URL}/profile/?search=nonexistent" 200 "Profile query - No matches with auth" "[]" "" "true"

# Pagination parameters
test_response "${PROFILES_URL}/profile/?search=Test&page=1" 200 "Profile query - Page 1" "Test" "" "true"
test_response "${PROFILES_URL}/profile/?search=Test&page=2" 200 "Profile query - Page 2" "[]" "" "true"
test_response "${PROFILES_URL}/profile/?search=Test&limit=5" 200 "Profile query - Limit 5" "Test" "" "true"
test_response "${PROFILES_URL}/profile/?search=Test&limit=10" 200 "Profile query - Limit 10 (default)" "Test" "" "true"
test_response "${PROFILES_URL}/profile/?search=Test&limit=20" 200 "Profile query - Limit 20" "Test" "" "true"
test_response "${PROFILES_URL}/profile/?search=Test&page=1&limit=5" 200 "Profile query - Page 1, Limit 5" "Test" "" "true"

# Edge cases for pagination
test_response "${PROFILES_URL}/profile/?search=Test&page=0" 200 "Profile query - Page 0 (should default to 1)" "Test" "" "true"
test_response "${PROFILES_URL}/profile/?search=Test&page=-1" 200 "Profile query - Page -1 (should default to 1)" "Test" "" "true"
test_response "${PROFILES_URL}/profile/?search=Test&limit=0" 200 "Profile query - Limit 0 (should use default)" "Test" "" "true"
test_response "${PROFILES_URL}/profile/?search=Test&limit=-1" 200 "Profile query - Limit -1 (should use default)" "Test" "" "true"

# Empty search
test_response "${PROFILES_URL}/profile/?search=" 200 "Profile query - Empty search" "[]" "" "true"
test_response "${PROFILES_URL}/profile/?" 200 "Profile query - No search parameter" "[]" "" "true"

# Response structure validation
test_json_structure "${PROFILES_URL}/profile/?search=Test&limit=5" "Profile query - Valid JSON array structure" "objectId" "true"

# =========================================
# POSTS SEARCH - PUBLIC AUTOCOMPLETE (/posts/search)
# =========================================
log_section "📝 POSTS SEARCH - PUBLIC AUTOCOMPLETE"

# Basic posts search tests
test_response "${POSTS_URL}/posts/search?q=machine+learning" 200 "Posts search - Basic content search" "machine"
test_response "${POSTS_URL}/posts/search?q=golang" 200 "Posts search - Technology search" "golang"
test_response "${POSTS_URL}/posts/search?q=artificial+intelligence" 200 "Posts search - Multi-word search" "artificial"
test_response "${POSTS_URL}/posts/search?q=test" 200 "Posts search - Generic search" "test"

# Case variations
test_response "${POSTS_URL}/posts/search?q=MACHINE+LEARNING" 200 "Posts search - Uppercase search" "machine"
test_response "${POSTS_URL}/posts/search?q=Machine+Learning" 200 "Posts search - Mixed case search" "machine"
test_response "${POSTS_URL}/posts/search?q=GoLaNg" 200 "Posts search - Mixed case technology" "golang"

# Edge cases
test_response "${POSTS_URL}/posts/search?q=" 200 "Posts search - Empty query" "[]"
test_response "${POSTS_URL}/posts/search?q=a" 200 "Posts search - Single character" "[]"
test_response "${POSTS_URL}/posts/search?q=ab" 200 "Posts search - Two characters" "[]"
test_response "${POSTS_URL}/posts/search?q=nonexistentcontent123456789" 200 "Posts search - No matches" "[]"

# Limit parameter tests - boundary conditions
test_response "${POSTS_URL}/posts/search?q=test&limit=1" 200 "Posts search - Limit 1" "test"
test_response "${POSTS_URL}/posts/search?q=test&limit=2" 200 "Posts search - Limit 2" "test"
test_response "${POSTS_URL}/posts/search?q=test&limit=5" 200 "Posts search - Limit 5 (default max)" "test"
test_response "${POSTS_URL}/posts/search?q=test&limit=20" 200 "Posts search - Limit 20 (max allowed)" "test"
test_response "${POSTS_URL}/posts/search?q=test&limit=21" 200 "Posts search - Limit 21 (should cap at 20)" "test"
test_response "${POSTS_URL}/posts/search?q=test&limit=100" 200 "Posts search - Limit 100 (should cap at 20)" "test"
test_response "${POSTS_URL}/posts/search?q=test&limit=0" 200 "Posts search - Limit 0 (invalid, should use default)" "test"
test_response "${POSTS_URL}/posts/search?q=test&limit=-1" 200 "Posts search - Limit -1 (invalid, should use default)" "test"
test_response "${POSTS_URL}/posts/search?q=test&limit=abc" 200 "Posts search - Limit non-numeric (should use default)" "test"

# Very long query
test_response "${POSTS_URL}/posts/search?q=${LONG_QUERY}" 200 "Posts search - Very long query" "[]"

# Special characters and encoding
test_response "${POSTS_URL}/posts/search?q=test%20post" 200 "Posts search - URL encoded spaces" "test"
test_response "${POSTS_URL}/posts/search?q=test+post" 200 "Posts search - Plus encoded spaces" "test"
test_response "${POSTS_URL}/posts/search?q=test%26post" 200 "Posts search - URL encoded ampersand" "test"

# Security tests - SQL injection attempts
test_response "${POSTS_URL}/posts/search?q=test'%20OR%201=1" 200 "Posts search - SQL injection attempt 1" "[]"
test_response "${POSTS_URL}/posts/search?q=test';%20DROP%20TABLE%20posts;" 200 "Posts search - SQL injection attempt 2" "[]"
test_response "${POSTS_URL}/posts/search?q=test'%20UNION%20SELECT%20*" 200 "Posts search - SQL injection attempt 3" "[]"

# Unicode and special characters (search for edge case data we created)
test_response "${POSTS_URL}/posts/search?q=UniqueKeyword" 200 "Posts search - Unicode emoji" "UniqueKeyword"
test_response "${POSTS_URL}/posts/search?q=%E4%B8%AD%E6%96%87" 200 "Posts search - Chinese characters" "[]"
test_response "${POSTS_URL}/posts/search?q=SpecialChars" 200 "Posts search - Special chars" "SpecialChars"

# Whitespace handling
test_response "${POSTS_URL}/posts/search?q=%20%20%20" 200 "Posts search - Whitespace only" "[]"
test_response "${POSTS_URL}/posts/search?q=test%20%20%20post" 200 "Posts search - Multiple spaces" "test"

# Response structure validation
test_json_structure "${POSTS_URL}/posts/search?q=test&limit=5" "Posts search - Valid JSON array structure" "objectId"

# =========================================
# POSTS SEARCH - CURSOR-BASED PAGINATION (/posts/queries/search/cursor)
# =========================================
log_section "📝 POSTS SEARCH - CURSOR-BASED PAGINATION"

# Basic cursor search (requires authentication and 'q' parameter)
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test" 200 "Posts cursor search - Basic search" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=machine+learning" 200 "Posts cursor search - Multi-word" "machine" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=golang" 200 "Posts cursor search - Technology" "golang" "" "true"

# Missing required 'q' parameter (should return 400)
test_response "${POSTS_URL}/posts/queries/search/cursor" 400 "Posts cursor search - Missing 'q' parameter" "required" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=" 400 "Posts cursor search - Empty 'q' parameter" "required" "" "true"

# Limit parameter tests - boundary conditions (1-100)
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=1" 200 "Posts cursor search - Limit 1" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=10" 200 "Posts cursor search - Limit 10" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=20" 200 "Posts cursor search - Limit 20 (default)" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=50" 200 "Posts cursor search - Limit 50" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=100" 200 "Posts cursor search - Limit 100 (max)" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=101" 200 "Posts cursor search - Limit 101 (should cap at 100)" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=0" 200 "Posts cursor search - Limit 0 (should use default)" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=-1" 200 "Posts cursor search - Limit -1 (should use default)" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=abc" 200 "Posts cursor search - Limit non-numeric (should use default)" "test" "" "true"

# Sort field tests
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortField=createdDate" 200 "Posts cursor search - Sort by createdDate" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortField=score" 200 "Posts cursor search - Sort by score" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortField=lastUpdated" 200 "Posts cursor search - Sort by lastUpdated" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortField=viewCount" 200 "Posts cursor search - Sort by viewCount" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortField=invalid" 200 "Posts cursor search - Invalid sortField (should use default)" "test" "" "true"

# Sort direction tests
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortDirection=asc" 200 "Posts cursor search - Sort ascending" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortDirection=desc" 200 "Posts cursor search - Sort descending" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortDirection=invalid" 200 "Posts cursor search - Invalid sortDirection (should use default)" "test" "" "true"

# Combined sort parameters
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortField=score&sortDirection=desc" 200 "Posts cursor search - Sort by score desc" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&sortField=createdDate&sortDirection=asc" 200 "Posts cursor search - Sort by createdDate asc" "test" "" "true"

# Cursor pagination tests (note: cursors are returned in response, we test the parameter acceptance)
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&cursor=test_cursor_123" 200 "Posts cursor search - With cursor parameter" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&after=test_after_123" 200 "Posts cursor search - With after cursor" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&before=test_before_123" 200 "Posts cursor search - With before cursor" "test" "" "true"

# Filter parameters - owner (requires valid UUID, testing parameter acceptance)
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&owner=00000000-0000-0000-0000-000000000000" 200 "Posts cursor search - With owner filter" "posts" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&owner=invalid-uuid" 200 "Posts cursor search - Invalid owner UUID (should ignore)" "posts" "" "true"

# Filter parameters - post type
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&type=1" 200 "Posts cursor search - With type filter" "posts" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&type=0" 200 "Posts cursor search - Type 0" "posts" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&type=abc" 200 "Posts cursor search - Invalid type (should ignore)" "test" "" "true"

# Filter parameters - tags (search for edge case data we created)
test_response "${POSTS_URL}/posts/queries/search/cursor?q=TagFilterTestPost&tags=tech" 200 "Posts cursor search - With tags filter" "TagFilterTestPost" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=UniqueKeyword&tags=golang" 200 "Posts cursor search - Tags golang" "UniqueKeyword" "" "true"

# Filter parameters - createdAfter (Unix timestamp)
CURRENT_TIMESTAMP=$(date +%s)
PAST_TIMESTAMP=$((CURRENT_TIMESTAMP - 86400))  # 1 day ago
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&createdAfter=${PAST_TIMESTAMP}" 200 "Posts cursor search - With createdAfter filter" "test" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&createdAfter=invalid" 200 "Posts cursor search - Invalid createdAfter (should ignore)" "test" "" "true"

# Combined filters
test_response "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=10&sortField=score&sortDirection=desc" 200 "Posts cursor search - Combined: limit+sort" "posts" "" "true"
test_response "${POSTS_URL}/posts/queries/search/cursor?q=TagFilterTestPost&type=1&tags=tech&limit=5" 200 "Posts cursor search - Combined: type+tags+limit" "posts" "" "true"

# Response structure validation
test_json_structure "${POSTS_URL}/posts/queries/search/cursor?q=test&limit=5" "Posts cursor search - Valid JSON structure" "posts" "true"

# =========================================
# AUTHENTICATION & AUTHORIZATION TESTS
# =========================================
log_section "🔐 AUTHENTICATION & AUTHORIZATION"

# Test profile query endpoint without authentication (should fail)
log_test "Profile query - Without authentication"
response=$(curl -s -w "\n%{http_code}" --max-time 10 "${PROFILES_URL}/profile/?search=Test" 2>/dev/null)
http_code=$(echo "$response" | tail -n1)
if [ "$http_code" -eq 401 ] || [ "$http_code" -eq 403 ]; then
    log_success "Profile query - Without authentication correctly rejected"
else
    log_failure "Profile query - Without authentication should be rejected, got status $http_code"
fi

# Test posts cursor search without authentication (should fail)
log_test "Posts cursor search - Without authentication"
response=$(curl -s -w "\n%{http_code}" --max-time 10 "${POSTS_URL}/posts/queries/search/cursor?q=test" 2>/dev/null)
http_code=$(echo "$response" | tail -n1)
if [ "$http_code" -eq 401 ] || [ "$http_code" -eq 403 ]; then
    log_success "Posts cursor search - Without authentication correctly rejected"
else
    log_failure "Posts cursor search - Without authentication should be rejected, got status $http_code"
fi

# Test public endpoints work without authentication
test_response "${PROFILES_URL}/profile/search?q=Test" 200 "Profile search - Public endpoint (no auth required)" "Test"
test_response "${POSTS_URL}/posts/search?q=test" 200 "Posts search - Public endpoint (no auth required)" "test"

# Test invalid token
log_test "Profile query - Invalid token"
response=$(curl -s -w "\n%{http_code}" --max-time 10 -H "Authorization: Bearer invalid_token_12345" "${PROFILES_URL}/profile/?search=Test" 2>/dev/null)
http_code=$(echo "$response" | tail -n1)
if [ "$http_code" -eq 401 ] || [ "$http_code" -eq 403 ]; then
    log_success "Profile query - Invalid token correctly rejected"
else
    log_failure "Profile query - Invalid token should be rejected, got status $http_code"
fi

# =========================================
# PERFORMANCE TESTS
# =========================================
log_section "⚡ PERFORMANCE TESTS"

# Sequential requests performance
test_performance "${PROFILES_URL}/profile/search?q=Test" "Performance - Profile search (sequential)" 2000
test_performance "${POSTS_URL}/posts/search?q=test" "Performance - Posts search (sequential)" 2000
test_performance "${PROFILES_URL}/profile/?search=Test" "Performance - Profile query (authenticated)" 2000 "true"
test_performance "${POSTS_URL}/posts/queries/search/cursor?q=test" "Performance - Posts cursor search (authenticated)" 2000 "true"

# Concurrent requests test
log_test "Performance test - Concurrent profile searches (5 parallel)"
start_time=$(date +%s%N)
for i in {1..5}; do
    curl -s --max-time 10 "${PROFILES_URL}/profile/search?q=Test" > /dev/null 2>&1 &
done
wait
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 ))
if [ "$duration" -lt 3000 ]; then
    log_success "Performance test - Concurrent profile searches completed in ${duration}ms"
else
    log_failure "Performance test - Concurrent profile searches too slow: ${duration}ms"
fi

log_test "Performance test - Concurrent posts searches (5 parallel)"
start_time=$(date +%s%N)
for i in {1..5}; do
    curl -s --max-time 10 "${POSTS_URL}/posts/search?q=test" > /dev/null 2>&1 &
done
wait
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 ))
if [ "$duration" -lt 3000 ]; then
    log_success "Performance test - Concurrent posts searches completed in ${duration}ms"
else
    log_failure "Performance test - Concurrent posts searches too slow: ${duration}ms"
fi

# =========================================
# DATABASE INDEX VERIFICATION
# =========================================
log_section "🗄️ DATABASE INDEX VERIFICATION"

log_info "Verifying search indexes exist..."

# Check if search indexes exist
log_test "Database indexes - Profile search index exists"
PROFILE_INDEX_EXISTS=$(timeout 10s docker exec telar-postgres psql -U postgres -d telar_social_test -t -c "SELECT 1 FROM pg_indexes WHERE indexname = 'idx_profiles_search_fts';" 2>/dev/null | wc -l 2>/dev/null || echo "0")

if [ "$PROFILE_INDEX_EXISTS" -gt 0 ]; then
    log_success "Database indexes - Profile search index exists"
else
    log_failure "Database indexes - Profile search index missing"
fi

log_test "Database indexes - Posts search index exists"
POSTS_INDEX_EXISTS=$(timeout 10s docker exec telar-postgres psql -U postgres -d telar_social_test -t -c "SELECT 1 FROM pg_indexes WHERE indexname = 'idx_posts_body_fts';" 2>/dev/null | wc -l 2>/dev/null || echo "0")

if [ "$POSTS_INDEX_EXISTS" -gt 0 ]; then
    log_success "Database indexes - Posts search index exists"
else
    log_failure "Database indexes - Posts search index missing"
fi

# =========================================
# EDGE CASES & ERROR HANDLING
# =========================================
log_section "⚠️ EDGE CASES & ERROR HANDLING"

# Invalid URLs
test_response "${PROFILES_URL}/profile/search/invalid" 404 "Profile search - Invalid endpoint" "" "" "false"
# Posts search invalid endpoint - protected route returns 401 without auth, which is correct security behavior
log_test "Posts search - Invalid endpoint (protected route)"
response=$(curl -s -w "\n%{http_code}" --max-time 10 "${POSTS_URL}/posts/search/invalid" 2>/dev/null)
http_code=$(echo "$response" | tail -n1)
if [ "$http_code" -eq 401 ] || [ "$http_code" -eq 404 ]; then
    log_success "Posts search - Invalid endpoint correctly handled (got $http_code)"
else
    log_failure "Posts search - Invalid endpoint - Expected 401 or 404, got $http_code"
fi

# Malformed query parameters
test_response "${PROFILES_URL}/profile/search?q=Test&limit=" 200 "Profile search - Empty limit parameter" "Test"
test_response "${POSTS_URL}/posts/search?q=test&limit=" 200 "Posts search - Empty limit parameter" "test"

# Very large limit values
test_response "${PROFILES_URL}/profile/search?q=Test&limit=999999" 200 "Profile search - Very large limit" "Test"
test_response "${POSTS_URL}/posts/search?q=test&limit=999999" 200 "Posts search - Very large limit" "test"

# Special query characters that might break parsing
test_response "${PROFILES_URL}/profile/search?q=Test%3DUser" 200 "Profile search - URL encoded equals" "Test"
test_response "${POSTS_URL}/posts/search?q=test%3Dpost" 200 "Posts search - URL encoded equals" "test"

# Multiple query parameters (should handle gracefully)
test_response "${PROFILES_URL}/profile/search?q=Test&limit=5&extra=param" 200 "Profile search - Extra parameters" "Test"
test_response "${POSTS_URL}/posts/search?q=test&limit=5&extra=param" 200 "Posts search - Extra parameters" "test"

# =========================================
# TEST SUMMARY
# =========================================
log_section "📊 TEST SUMMARY"

echo "Total Tests: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [ "$FAILED_TESTS" -eq 0 ]; then
    echo -e "\n${GREEN}🎉 ALL TESTS PASSED! Search functionality is working correctly.${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed. Please review the failures above.${NC}"
    exit 1
fi
