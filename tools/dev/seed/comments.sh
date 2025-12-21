#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

AUTH_URL="${AUTH_URL:-$BASE_URL}"
COMMENTS_URL="${COMMENTS_URL:-$BASE_URL}"
TEST_EMAIL="${TEST_EMAIL:-test-signup-1760066266@telar.dev}"
TEST_PASSWORD="${TEST_PASSWORD:-5gHQAEz@QD\$j3Sm}"
POST_ID="${POST_ID:-}"
NUM_COMMENTS="${NUM_COMMENTS:-50}"

if [[ -z "${POST_ID}" ]]; then
  log_error "POST_ID must be provided (export POST_ID=<uuid>)"
  exit 1
fi

log_info "Authenticating as ${TEST_EMAIL}..."
LOGIN_RESPONSE=$(curl -s -X POST "${AUTH_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\",\"responseType\":\"json\"}")

JWT_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"accessToken":"[^"]*' | cut -d'"' -f4)

if [[ -z "${JWT_TOKEN}" ]]; then
  log_error "Unable to obtain JWT token. Response: ${LOGIN_RESPONSE}"
  exit 1
fi

log_info "Generating ${NUM_COMMENTS} comments for post ${POST_ID}..."

SUCCESS=0
for i in $(seq 1 "${NUM_COMMENTS}"); do
  COMMENT_TEXT="Seeded comment #${i} at $(date +%H:%M:%S)"
  RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${COMMENTS_URL}/comments" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${JWT_TOKEN}" \
    -d "{
      \"postId\": \"${POST_ID}\",
      \"text\": \"${COMMENT_TEXT}\"
    }")

  HTTP_CODE=$(echo "${RESPONSE}" | tail -n1)
  if [[ "${HTTP_CODE}" == "200" || "${HTTP_CODE}" == "201" ]]; then
    SUCCESS=$((SUCCESS + 1))
  else
    BODY=$(echo "${RESPONSE}" | sed '$d')
    log_warn "Failed to create comment #${i} (HTTP ${HTTP_CODE}): ${BODY}"
  fi

  sleep 0.05
done

log_info "Completed seeding. Successful comments: ${SUCCESS}/${NUM_COMMENTS}"

