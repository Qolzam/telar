#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

AUTH_URL="${AUTH_URL:-$BASE_URL}"
POSTS_URL="${POSTS_URL:-$BASE_URL}"
TEST_EMAIL="${TEST_EMAIL:-test-signup-1760066266@telar.dev}"
TEST_PASSWORD="${TEST_PASSWORD:-5gHQAEz@QD\$j3Sm}"
NUM_POSTS="${NUM_POSTS:-100}"

log_info "Logging in as ${TEST_EMAIL}..."
LOGIN_RESPONSE=$(curl -s -X POST "${AUTH_URL}/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\",\"responseType\":\"json\"}")

if [ $? -ne 0 ]; then
    log_error "Failed to connect to auth service. Is it running?"
    exit 1
fi

JWT_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"accessToken":"[^"]*' | cut -d'"' -f4)

if [ -z "$JWT_TOKEN" ]; then
    log_error "Failed to get JWT token. Response: $LOGIN_RESPONSE"
    exit 1
fi

log_success "Logged in successfully. Token obtained."

log_info "Creating ${NUM_POSTS} test posts..."

CREATED=0
FAILED=0

for i in $(seq 1 $NUM_POSTS); do
    POST_BODY="Test post #${i} - Generated at $(date +%H:%M:%S). This is a test post for infinite scroll functionality. Lorem ipsum dolor sit amet, consectetur adipiscing elit."
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${POSTS_URL}/posts" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -d "{
            \"postTypeId\": 1,
            \"body\": \"${POST_BODY}\",
            \"permission\": \"Public\"
        }")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    if [ "$HTTP_CODE" -eq 200 ] || [ "$HTTP_CODE" -eq 201 ]; then
        CREATED=$((CREATED + 1))
        if [ $((i % 10)) -eq 0 ]; then
            log_info "Created ${i}/${NUM_POSTS} posts..."
        fi
    else
        FAILED=$((FAILED + 1))
        log_error "Failed to create post #${i}. HTTP ${HTTP_CODE}: ${BODY}"
    fi
    sleep 0.1
done

log_success "Finished creating posts. Success: ${CREATED}, Failed: ${FAILED}"
log_info "You can now test infinite scrolling in the browser!"

