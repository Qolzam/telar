# ====================================================================================
# Telar Microservices - Test & CI Orchestration Makefile
#
# Philosophy:
# 1. Use variables for common settings (parallelism, timeouts, flags) for consistency.
# 2. The Makefile manages the ENVIRONMENT (Docker containers, build tags).
# 3. The Go test suite manages the test LOGIC (running against PostgreSQL).
# ====================================================================================

.PHONY: all help \
        up-dbs-dev up-postgres down-postgres clean-dbs status logs-postgres docker-start \
        test test-all test-posts test-comments test-votes test-userrels test-auth test-profile test-circles test-setting test-admin test-gallery test-notifications test-actions test-storage test-cache \
        test-db-operations test-posts-operations test-database-compatibility bench-db-operations test-all-operations \
        local-test-all \
        ci-fast ci-test ci-full ci-nightly \
        report open-report clean-reports \
        bench bench-env bench-calibrated bench-summary open-profiles \
        test-transactions \
        lint lint-fix \
        run-api run-web run-both run-profile run-profile-standalone run-posts run-comments dev stop-servers restart-servers pre-flight-check logs-api logs-web \
        test-e2e-posts test-e2e-profile test-e2e-comments

# --- Configuration Variables ---
PARALLEL ?= 8
TIMEOUT ?= 15m
REPORT_DIR := reports
PROFILES_DIR := $(REPORT_DIR)/profiles
TEST_ENV_SCRIPT := tools/dev/test_env.sh

GO_TEST_FLAGS := -count=1 -v -race -covermode=atomic -timeout=$(TIMEOUT) -parallel=$(PARALLEL)

TEST_ENV_VARS := RUN_DB_TESTS=1

# --- Linting & Code Quality ---
lint:
	@echo "Running golangci-lint..."
	@cd apps/api && golangci-lint run --config=../../.golangci.yml

lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	@cd apps/api && golangci-lint run --config=../../.golangci.yml --fix

# --- Docker & Database Management ---

# Development: Start databases WITHOUT modifying .env (preserves your settings)
up-dbs-dev:
	@echo "Starting PostgreSQL for DEVELOPMENT (preserving your .env settings)..."
	@docker start telar-postgres 2>/dev/null || \
		(echo "Creating PostgreSQL container..." && \
		 docker run -d --name telar-postgres \
		 -e POSTGRES_PASSWORD=postgres \
		 -e POSTGRES_USER=postgres \
		 -e POSTGRES_DB=telar_social_test \
		 -p 5432:5432 postgres:15)
	@docker start telar-mailhog 2>/dev/null || \
		(echo "Creating MailHog container..." && \
		 docker run -d --name telar-mailhog \
		 -p 1025:1025 -p 8025:8025 mailhog/mailhog:latest)
	@echo "✅ PostgreSQL ready for development (your .env settings preserved)"

# Testing: Use test_env.sh (configures .env for consistent test environment)
up-postgres:
	@$(TEST_ENV_SCRIPT) up postgres

down-postgres:
	@$(TEST_ENV_SCRIPT) down postgres

clean-dbs:
	@echo "Recreating fresh PostgreSQL container and removing all data volumes..."
	@$(MAKE) down-postgres
	@echo "Removing any existing PostgreSQL data volumes..."
	@docker volume ls -q --filter name=telar-postgres | xargs -r docker volume rm || true
	@docker volume ls -q --filter name=postgres | xargs -r docker volume rm || true
	@$(MAKE) up-postgres
	@echo "PostgreSQL is clean and ready with the latest schema."

status:
	@$(TEST_ENV_SCRIPT) status

logs-postgres:
	@$(TEST_ENV_SCRIPT) logs postgres

docker-start:
	@./tools/dev/scripts/docker-start.sh

all: test-all

test-posts: up-postgres
	@echo "Testing 'posts' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./posts $(GO_TEST_FLAGS)

test-comments: up-postgres
	@echo "Testing 'comments' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./comments $(GO_TEST_FLAGS)

test-votes: up-postgres
	@echo "Testing 'votes' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./votes $(GO_TEST_FLAGS)

test-userrels: up-postgres
	@echo "Testing 'userrels' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./userrels $(GO_TEST_FLAGS)

test-auth: up-postgres
	@echo "Testing 'auth' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./auth $(GO_TEST_FLAGS)

test-profile: up-postgres
	@echo "Testing 'profile' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./profile/... $(GO_TEST_FLAGS)

test-circles: up-postgres
	@echo "Testing 'circles' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./circles $(GO_TEST_FLAGS)

test-setting: up-postgres
	@echo "Testing 'setting' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./setting $(GO_TEST_FLAGS)

test-admin: up-postgres
	@echo "Testing 'admin' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./admin $(GO_TEST_FLAGS)

test-gallery: up-postgres
	@echo "Testing 'gallery' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./gallery $(GO_TEST_FLAGS)

test-notifications: up-postgres
	@echo "Testing 'notifications' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./notifications $(GO_TEST_FLAGS)

test-actions: up-postgres
	@echo "Testing 'actions' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./actions $(GO_TEST_FLAGS)

test-storage: up-postgres
	@echo "Testing 'storage' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./storage $(GO_TEST_FLAGS)

test-cache:
	@echo "Testing internal cache..."
	@cd apps/api && go test ./internal/cache $(GO_TEST_FLAGS)

test-all: up-postgres
	@echo "Running all tests for all microservices with parallelism $(PARALLEL)..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./... $(GO_TEST_FLAGS)

test-all-race: up-postgres
	@echo "Running all tests with race detector..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./... -count=1 -v -race -timeout=20m -parallel=8

local-test-all: docker-start test-all

# --- CI/CD Targets ---

ci-fast: up-postgres test-posts test-comments test-votes test-userrels test-admin

ci-test: up-postgres test-posts test-comments test-votes test-userrels test-auth test-profile test-circles test-setting test-admin test-gallery test-notifications test-actions test-storage test-cache

ci-full: ci-test

ci-nightly: test-all

# --- Reporting & Profiling ---

report: up-postgres
	@echo "Generating test and coverage reports..."
	@mkdir -p $(REPORT_DIR)
	@cd apps/api && RUN_DB_TESTS=1 go test ./... $(GO_TEST_FLAGS) -coverprofile=coverage.out -json > ../../$(REPORT_DIR)/test.json
	@cd apps/api && go tool cover -func=coverage.out > ../../$(REPORT_DIR)/coverage.txt
	@cd apps/api && go tool cover -html=coverage.out -o ../../$(REPORT_DIR)/coverage.html
	@echo "\n==== Coverage Summary ===="
	@grep -E "^total:" $(REPORT_DIR)/coverage.txt || echo "No coverage data found."
	@echo "Reports generated in $(REPORT_DIR)/. Open with 'make open-report'"

open-report:
	@open $(REPORT_DIR)/coverage.html || echo "Could not open report. See $(REPORT_DIR)/coverage.html"

clean-reports:
	@echo "Cleaning reports directory..."
	@rm -rf $(REPORT_DIR)

bench: bench-calibrated

bench-env: | $(REPORT_DIR)
	@echo "Collecting benchmark environment details..."
	@{ \
		date; \
		go version || true; \
		uname -a || true; \
		sysctl -n machdep.cpu.brand_string 2>/dev/null || true; \
		echo "\n-- Docker Info --"; \
		docker version 2>/dev/null || true; \
		docker info 2>/dev/null | grep -E "(Server Version|Kernel Version|Operating System|Total Memory|NCPU)" || true; \
	} > $(REPORT_DIR)/bench_env.txt
	@echo "Environment details saved to $(REPORT_DIR)/bench_env.txt"

$(REPORT_DIR):
	@mkdir -p $(REPORT_DIR)

$(PROFILES_DIR): | $(REPORT_DIR)
	@mkdir -p $(PROFILES_DIR)

bench-calibrated: up-postgres bench-env $(PROFILES_DIR)
	@echo "Running calibrated benchmarks and generating profiles..."
	@cd apps/api && RUN_DB_TESTS=1 go test -bench=. -benchmem -run=^Benchmark ./... \
		-cpuprofile=../../$(PROFILES_DIR)/all_cpu.pprof \
		-memprofile=../../$(PROFILES_DIR)/all_mem.pprof | tee ../../$(REPORT_DIR)/bench_results.txt
	@echo "Profiling data saved to $(PROFILES_DIR)/"
	@echo "Benchmark results saved to $(REPORT_DIR)/bench_results.txt"
	@$(MAKE) bench-summary

bench-summary: | $(REPORT_DIR)
	@echo "Summarizing benchmark results..."
	@grep -E "^Benchmark" $(REPORT_DIR)/bench_results.txt > $(REPORT_DIR)/bench_summary.txt || echo "No benchmark results found."
	@echo "\nSummary saved to $(REPORT_DIR)/bench_summary.txt"

open-profiles:
	@echo "To inspect CPU profile: go tool pprof -http=:0 $(PROFILES_DIR)/all_cpu.pprof"
	@echo "To inspect Memory profile: go tool pprof -http=:0 $(PROFILES_DIR)/all_mem.pprof"

# --- Transaction Testing ---

test-transactions: up-postgres
	@echo "Testing enterprise transaction management with PostgreSQL..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./internal/database/ -v -run TestTransactionSuite $(GO_TEST_FLAGS)

# --- Database Operations Testing ---

test-db-operations: up-postgres
	@echo "Testing PostgreSQL database operations..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./internal/database/postgresql -v -run TestOperations $(GO_TEST_FLAGS)

test-posts-operations: up-postgres
	@echo "Testing posts service operations..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./posts -v -run TestPostsOperations $(GO_TEST_FLAGS)

test-database-compatibility: up-postgres
	@echo "Testing PostgreSQL database compatibility..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./posts -v -run TestDatabaseCompatibility $(GO_TEST_FLAGS)

bench-db-operations: up-postgres
	@echo "Benchmarking database operations..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./internal/database/postgresql -bench=. -benchmem -run=^Benchmark $(GO_TEST_FLAGS)

test-all-operations: up-postgres
	@echo "Running all database operations tests..."
	@$(MAKE) test-db-operations
	@$(MAKE) test-posts-operations
	@$(MAKE) test-database-compatibility
	@$(MAKE) bench-db-operations
	@echo "✅ All database operations tests completed"

# --- Help ---
help:
	@echo "Available commands:"
	@echo ""
	@echo "Docker & Database Commands:"
	@echo "  up-dbs-dev        - Start PostgreSQL for DEVELOPMENT (preserves your .env settings)"
	@echo "  up-postgres       - Start PostgreSQL only (for testing)"
	@echo "  down-postgres     - Stop PostgreSQL container"
	@echo "  clean-dbs         - Recreate fresh PostgreSQL container (for testing)"
	@echo "  status            - Show status of Docker containers"
	@echo "  logs-postgres     - Tail logs for the PostgreSQL container"
	@echo ""
	@echo "Test Commands:"
	@echo "  test-all          - Run all tests across all microservices (default)"
	@echo "  test-<service>    - Run tests for a specific microservice (e.g., make test-posts)"
	@echo "  test-profile      - Run tests for profile microservice"
	@echo "  local-test-all    - Ensure Docker is running, then run all tests"
	@echo ""
	@echo "CI/CD Commands:"
	@echo "  ci-fast           - Run quick CI checks on core services"
	@echo "  ci-test           - Run standard CI validation for PRs"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint              - Run golangci-lint for code quality checks"
	@echo "  lint-fix          - Run golangci-lint with auto-fix enabled"
	@echo ""
	@echo "Reporting:"
	@echo "  report            - Generate test and coverage reports in ./reports/"
	@echo "  open-report       - Open the HTML coverage report in a browser."
	@echo "  clean-reports     - Remove the reports directory."
	@echo ""
	@echo "  bench             - Run all benchmarks and generate pprof profiles."
	@echo "  open-profiles     - Show commands to inspect performance profiles."
	@echo ""
	@echo "Database Operations Testing:"
	@echo "  test-db-operations        - Test PostgreSQL database operations."
	@echo "  test-posts-operations     - Test posts service operations."
	@echo "  test-database-compatibility - Test PostgreSQL database compatibility."
	@echo "  bench-db-operations       - Benchmark database operations performance."
	@echo "  test-all-operations       - Run all database operations tests."
	@echo ""
	@echo "Transaction Testing:"
	@echo "  test-transactions - Run enterprise transaction management tests with PostgreSQL."
	@echo ""
	@echo "Development Servers:"
	@echo "  run-api           - Start the Telar API server on port 8080 (requires databases)."
	@echo "  run-profile       - Start the Profile microservice on port 8081 (requires databases)."
	@echo "  run-posts         - Start the Posts microservice on port 8082 (requires databases)."
	@echo "  run-comments      - Start the Comments microservice on port 8083 (requires databases)."
	@echo "  run-web           - Start the Next.js web frontend development server."
	@echo "  run-both          - Start both API and web frontend servers concurrently."
	@echo "  dev       - Start both servers in background (recommended for development)."
	@echo "  run-api-background - Start API server in background with PID file (for E2E tests)."
	@echo "  stop-api-background - Stop background API server using PID file."
	@echo "  test-e2e-posts    - Run Posts E2E tests with automatic server management."
	@echo "  test-e2e-comments - Run Comments E2E tests with automatic server management."
	@echo "  test-e2e-profile  - Run Profile E2E tests with automatic server management."
	@echo "  stop-servers      - Stop all running servers."
	@echo "  restart-servers   - Restart all servers safely (preserves Cursor processes)."
	@echo "  pre-flight-check  - Check system readiness before server startup."
	@echo "  logs-api          - Tail API server logs."
	@echo "  logs-web          - Tail web server logs."
	@echo ""
	@echo "Options:"
	@echo "  PARALLEL=<N>      - Set the number of parallel tests to run (default: 8)."
	@echo "  TIMEOUT=<duration>  - Set the test timeout (default: 15m)."

# --- Development Servers ---
# NOTE: These use up-dbs-dev (NOT up-postgres) to preserve your .env settings

run-api: up-dbs-dev
	@echo "Starting Telar API server (using your .env settings)..."
	@cd apps/api && go run cmd/server/main.go

run-web:
	@echo "Starting Next.js web frontend development server..."
	@cd apps/web && pnpm dev

run-profile: up-dbs-dev
	@echo "Starting Profile microservice (using your .env settings)..."
	@cd apps/api && go run cmd/services/profile/main.go

run-profile-standalone:
	@echo "Starting Profile microservice standalone on port 8081..."
	@cd apps/api && go run cmd/services/profile/main.go

run-posts: up-dbs-dev
	@echo "Starting Posts microservice (using your .env settings)..."
	@cd apps/api && go run cmd/services/posts/main.go

run-comments: up-dbs-dev
	@echo "Starting Comments microservice (using your .env settings)..."
	@cd apps/api && go run cmd/services/comments/main.go

run-both: up-dbs-dev
	@echo "Starting both API and web frontend servers..."
	@echo "API server will be available at: http://localhost:8080"
	@echo "Web frontend will be available at: http://localhost:3000"
	@echo ""
	@trap 'kill 0' EXIT; \
	(cd apps/api && go run cmd/server/main.go) & \
	(cd apps/web && pnpm dev) & \
	wait

# --- Background Process Management ---

dev: up-dbs-dev
	@./tools/dev/scripts/start-servers-bg.sh

stop-servers:
	@./tools/dev/scripts/stop-servers.sh

restart-servers:
	@./tools/dev/scripts/restart-servers.sh

# Target to run the API stack in the background for E2E tests
run-api-background: up-dbs-dev
	@echo "Starting API in background..."
	@ps aux | grep "go run.*cmd/server/main.go\|go run.*cmd/main.go" | grep -v grep | awk '{print $$2}' | xargs kill -9 2>/dev/null || true
	@sleep 1
	@cd apps/api && nohup go run cmd/server/main.go > ../../api.log 2>&1 & echo $$! > ../../api.pid
	@echo "API started with PID `cat api.pid`. Tailing logs..."
	@sleep 5

# Target to stop the background API
stop-api-background:
	@echo "Stopping background API..."
	@if [ -f api.pid ]; then \
		kill `cat api.pid` 2>/dev/null && rm api.pid api.log 2>/dev/null && echo "API stopped."; \
	else \
		echo "API not running or PID file not found. Cleaning up any stale processes..."; \
		ps aux | grep "go run.*cmd/server/main.go\|go run.*cmd/main.go" | grep -v grep | awk '{print $$2}' | xargs kill -9 2>/dev/null || true; \
	fi

# Running E2E tests for posts service
test-e2e-posts: stop-api-background run-api-background
	@echo "Running Posts E2E tests..."
	@./tools/dev/scripts/posts_e2e_test.sh || true
	@$(MAKE) stop-api-background

# Running E2E tests for comments service
test-e2e-comments: stop-api-background run-api-background
	@echo "Running Comments E2E tests..."
	@./tools/dev/scripts/comments_e2e_test.sh || true
	@$(MAKE) stop-api-background

# Running E2E tests for  profile service
test-e2e-profile: stop-api-background run-api-background
	@echo "Running Profile E2E tests..."
	@./tools/dev/scripts/profile_e2e_test.sh || true
	@$(MAKE) stop-api-background


pre-flight-check:
	@./tools/dev/scripts/pre-flight-check.sh

logs-api:
	@echo "📋 Tailing API server logs (Ctrl+C to exit)..."
	@tail -f /tmp/telar-logs/api.log 2>/dev/null || echo "No API logs found. Is the server running?"

logs-web:
	@echo "📋 Tailing Web server logs (Ctrl+C to exit)..."
	@tail -f /tmp/telar-logs/web.log 2>/dev/null || echo "No Web logs found. Is the server running?"
