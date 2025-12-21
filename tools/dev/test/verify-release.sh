#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

log() {
    log_info "$1"
}

success() {
    log_success "$1"
}

fail() {
    log_error "$1"
    exit 1
}

warning() {
    log_warn "$1"
}

# Error Trap: Ensure we clean up background processes even if a test fails
cleanup() {
    log "Cleaning up environment..."
    make stop-api-background > /dev/null 2>&1 || true
}

trap cleanup EXIT

echo "========================================================"
echo "   🚀 STARTING FULL SYSTEM VERIFICATION PROTOCOL"
echo "========================================================"
echo ""

# 0. PRE-FLIGHT FILE INTEGRITY CHECK
log "Step 0: Pre-Flight File Integrity Check"
echo ""

REQUIRED_FILES=(
  "tools/dev/test/e2e-auth.sh"
  "tools/dev/test/e2e-profile.sh"
  "tools/dev/test/e2e-posts.sh"
  "tools/dev/test/e2e-comments.sh"
  "tools/dev/test/e2e-votes.sh"
  "apps/api/cmd/server/main.go"
)

for file in "${REQUIRED_FILES[@]}"; do
  if [ ! -f "$file" ]; then
    fail "CRITICAL: Missing required file: $file"
  fi
done

# Ensure all scripts have execution permissions
chmod +x tools/dev/test/*.sh

success "✓ File System Integrity OK"
echo ""

# 1. CODE HYGIENE & LINTING
log "Step 1: Code Hygiene & Linting"
echo ""

# Check for fmt.Print in production code (excluding tests, main.go, log package, testutil, and config)
# We exclude these directories as they legitimately use fmt.Print for logging/CLI output
EXCLUDED_DIRS="--exclude-dir=tools --exclude-dir=internal/pkg/log --exclude-dir=internal/testutil --exclude-dir=internal/platform/config"
if grep -r "fmt\.Print" apps/api --include="*.go" --exclude="*_test.go" --exclude="main.go" $EXCLUDED_DIRS 2>/dev/null | grep -v "internal/pkg/log\|internal/testutil\|internal/platform/config"; then
    fail "Code Hygiene Check Failed: fmt.Print found in production code. Use log package instead."
fi
success "✓ No fmt.Print in production code"

# Run the linter (skip if not installed)
log "Running golangci-lint..."
if command -v golangci-lint >/dev/null 2>&1; then
    if ! make lint; then
        fail "Linter failed. Fix linting errors before proceeding."
    fi
    success "Linting passed."
else
    warning "golangci-lint not found, skipping lint check"
fi
success "Hygiene & Linting passed."
echo ""

# 2. COMPILATION CHECK (The "Build" Gate)
log "Step 2: Compilation Integrity"
echo ""

# Force a clean build of all entry points
log "Cleaning build cache..."
go clean -cache > /dev/null 2>&1 || true

log "Building all entry points..."
if ! go build -v -o /dev/null ./apps/api/cmd/server/... ./apps/api/cmd/services/... 2>&1; then
    fail "Compilation failed. Fix build errors before proceeding."
fi
success "All binaries compile successfully."
echo ""

# 3. UNIT & INTEGRATION TESTS (The "Code" Gate)
log "Step 3: Unit & Integration Tests"
echo ""

# Clean databases silently
log "Preparing test environment..."
make clean-dbs > /dev/null 2>&1 || warning "Database cleanup had issues (may be expected)"

log "Running all Go tests..."
if ! make test-all; then
    fail "Unit/Integration tests failed. Fix test failures before proceeding."
fi
success "All Go tests passed."
echo ""

# 4. END-TO-END (E2E) TESTS (The "Product" Gate)
log "Step 4: End-to-End (E2E) Validation"
echo ""

# Start the server in the background
log "Starting API server in background..."
if ! make run-api-background; then
    fail "Failed to start API server. Check server logs."
fi

# Wait for server to be ready
log "Waiting for server to be ready..."
API_PORT="${API_PORT:-9099}"
MAX_WAIT=30
WAIT_COUNT=0
while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    if curl -s "http://127.0.0.1:${API_PORT}/health" > /dev/null 2>&1 || curl -s "http://127.0.0.1:${API_PORT}/posts" > /dev/null 2>&1; then
        success "Server is ready"
        break
    fi
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ $WAIT_COUNT -eq $MAX_WAIT ]; then
    fail "Server did not become ready within ${MAX_WAIT} seconds"
fi

echo ""

# Run the Gauntlet
log ">> Running Auth E2E..."
if ! bash "${SCRIPT_DIR}/e2e-auth.sh"; then
    fail "Auth E2E failed"
fi
success "✓ Auth E2E passed"
echo ""

log ">> Running Profile E2E..."
if ! bash "${SCRIPT_DIR}/e2e-profile.sh"; then
    fail "Profile E2E failed"
fi
success "✓ Profile E2E passed"
echo ""

log ">> Running Posts E2E..."
if ! bash "${SCRIPT_DIR}/e2e-posts.sh"; then
    fail "Posts E2E failed"
fi
success "✓ Posts E2E passed"
echo ""

log ">> Running Comments E2E..."
if ! bash "${SCRIPT_DIR}/e2e-comments.sh"; then
    fail "Comments E2E failed"
fi
success "✓ Comments E2E passed"
echo ""

log ">> Running Comments Optimization Tests (Enterprise Test Protocol)..."
log "   Prerequisites: clean-dbs, restart-servers, seed/users.sh 2"
log "   Seeding test users..."
if ! bash "${SCRIPT_DIR}/../seed/users.sh" 2 > /dev/null 2>&1; then
    warning "User seeding had issues, but continuing..."
fi
log "   Running optimization tests..."
if [ -f "apps/web/tests/comments-optimization.spec.ts" ]; then
    if ! (cd apps/web && pnpm test:e2e tests/comments-optimization.spec.ts --reporter=list --workers=1); then
        fail "Comments Optimization Tests failed"
    fi
    success "✓ Comments Optimization Tests passed"
else
    warning "Comments optimization test file not found, skipping..."
fi
echo ""

log ">> Running Votes E2E..."
if ! bash "${SCRIPT_DIR}/e2e-votes.sh"; then
    fail "Votes E2E failed"
fi
success "✓ Votes E2E passed"
echo ""

echo "========================================================"
echo -e "${GREEN}✅  VERIFICATION COMPLETE. READY TO MERGE.${NC}"
echo "========================================================"

