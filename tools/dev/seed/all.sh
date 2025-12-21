#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

log_banner "🌱 Seeding All Data"
log_info "This will seed users, posts, comments, and bookmarks in order."
echo ""

log_info "Step 1: Seeding users..."
if ! bash "${SCRIPT_DIR}/users.sh" 5; then
    log_error "User seeding failed. Aborting."
    exit 1
fi
echo ""

log_info "Step 2: Seeding posts..."
if ! bash "${SCRIPT_DIR}/posts.sh"; then
    log_error "Post seeding failed. Aborting."
    exit 1
fi
echo ""

log_info "Step 3: Getting first post ID for comments..."
FIRST_POST_ID=$(curl -s "${API_URL}/posts?limit=1" | grep -o '"objectId":"[^"]*' | cut -d'"' -f4 | head -1)

if [[ -z "$FIRST_POST_ID" ]]; then
    log_warn "Could not get post ID. Skipping comments seeding."
else
    log_info "Step 4: Seeding comments for post $FIRST_POST_ID..."
    export POST_ID="$FIRST_POST_ID"
    if ! bash "${SCRIPT_DIR}/comments.sh"; then
        log_warn "Comment seeding failed. Continuing..."
    fi
    echo ""
fi

log_info "Step 5: Seeding bookmarks..."
if ! bash "${SCRIPT_DIR}/bookmarks.sh"; then
    log_warn "Bookmark seeding failed. Continuing..."
fi
echo ""

log_banner "✅ All seeding complete!"

