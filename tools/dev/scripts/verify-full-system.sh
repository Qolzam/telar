#!/bin/bash

set -e # Exit immediately if any command exits with a non-zero status

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[VERIFY]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

fail() {
    echo -e "${RED}[FAILURE]${NC} $1"
    exit 1
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
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
MAX_WAIT=30
WAIT_COUNT=0
while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    if curl -s http://127.0.0.1:8080/health > /dev/null 2>&1 || curl -s http://127.0.0.1:8080/posts > /dev/null 2>&1; then
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
if ! ./tools/dev/scripts/auth_e2e_test.sh; then
    fail "Auth E2E failed"
fi
success "✓ Auth E2E passed"
echo ""

log ">> Running Profile E2E..."
if ! ./tools/dev/scripts/profile_e2e_test.sh; then
    fail "Profile E2E failed"
fi
success "✓ Profile E2E passed"
echo ""

log ">> Running Posts E2E..."
if ! ./tools/dev/scripts/posts_e2e_test.sh; then
    fail "Posts E2E failed"
fi
success "✓ Posts E2E passed"
echo ""

log ">> Running Comments E2E..."
if ! ./tools/dev/scripts/comments_e2e_test.sh; then
    fail "Comments E2E failed"
fi
success "✓ Comments E2E passed"
echo ""

log ">> Running Votes E2E..."
if ! ./tools/dev/scripts/votes_e2e_test.sh; then
    fail "Votes E2E failed"
fi
success "✓ Votes E2E passed"
echo ""

echo "========================================================"
echo -e "${GREEN}✅  VERIFICATION COMPLETE. READY TO MERGE.${NC}"
echo "========================================================"

