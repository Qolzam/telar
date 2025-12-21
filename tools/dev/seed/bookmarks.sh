#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

USERS_FILE="test_users.json"

if [[ ! -f "$USERS_FILE" ]]; then
    log_error "test_users.json not found. Please run seed/users.sh first."
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    log_error "jq is required but not installed. Please install jq to use this script."
    exit 1
fi

TOKEN=$(jq -r '.token' "$USERS_FILE" 2>/dev/null || echo "")

if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
    log_error "Could not extract token from test_users.json"
    exit 1
fi

HEADERS=(
    -H "Cookie: access_token=${TOKEN}"
    -H "Content-Type: application/json"
)

log_info "Creating 5 test posts..."
POST_IDS=()

for i in {1..5}; do
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/posts" \
        "${HEADERS[@]}" \
        -d "{\"text\": \"Test bookmark post ${i}\", \"tagLine\": \"test\"}")
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    
    if [[ "$HTTP_CODE" == "200" || "$HTTP_CODE" == "201" ]]; then
        POST_ID=$(echo "$BODY" | jq -r '.objectId' 2>/dev/null || echo "")
        if [[ -n "$POST_ID" && "$POST_ID" != "null" ]]; then
            POST_IDS+=("$POST_ID")
            log_success "Created post ${i}: $POST_ID"
        else
            log_warn "Created post ${i} but could not extract objectId"
        fi
    else
        log_error "Failed to create post ${i}. HTTP ${HTTP_CODE}: ${BODY}"
    fi
done

log_info "Bookmarking all posts..."
BOOKMARKED=0

for POST_ID in "${POST_IDS[@]}"; do
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/bookmarks/${POST_ID}/toggle" \
        "${HEADERS[@]}")
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    
    if [[ "$HTTP_CODE" == "200" || "$HTTP_CODE" == "201" ]]; then
        BOOKMARKED=$((BOOKMARKED + 1))
        log_success "Bookmarked: $POST_ID"
    else
        log_warn "Failed to bookmark $POST_ID. HTTP ${HTTP_CODE}"
    fi
done

log_success "Total bookmarks created: ${BOOKMARKED}/${#POST_IDS[@]}"

