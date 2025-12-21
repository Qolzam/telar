#!/bin/bash
# tools/dev/lib/common.sh
# Shared library for development scripts
# Provides common logging, colors, and configuration

# Colors
export RED='\033[0;31m'
export GREEN='\033[0;32m'
export BLUE='\033[0;34m'
export YELLOW='\033[1;33m'
export CYAN='\033[0;36m'
export NC='\033[0m'

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_banner() { echo -e "${CYAN}$1${NC}"; }

# Common Config
export API_URL="http://localhost:9099"
export WEB_URL="http://localhost:3000"
export BASE_URL="http://127.0.0.1:9099"
export AUTH_BASE="${BASE_URL}/auth"
export MAILHOG_URL="http://localhost:8025"

