# ====================================================================================
# Telar Microservices - Test & CI Orchestration Makefile
#
# Philosophy:
# 1. Use variables for common settings (parallelism, timeouts, flags) for consistency.
# 2. The Makefile manages the ENVIRONMENT (Docker containers, build tags).
# 3. The Go test suite manages the test LOGIC (running against Mongo/Postgres, skipping).
# ====================================================================================

.PHONY: all help \
        up-dbs-dev up-mongo up-postgres up-both down-both clean-dbs status logs-mongo logs-postgres docker-start \
        test test-all test-posts test-comments test-votes test-userrels test-auth test-profile test-circles test-setting test-admin test-gallery test-notifications test-actions test-storage test-cache \
        test-db-operations test-posts-operations test-database-compatibility bench-db-operations test-all-operations \
        local-test-all \
        ci-fast ci-test ci-full ci-nightly \
        report open-report clean-reports \
        bench bench-env bench-calibrated bench-summary open-profiles \
        up-both-replica test-transactions \
        lint lint-fix \
        run-api run-web run-both run-profile run-profile-standalone run-both-bg stop-servers restart-servers pre-flight-check logs-api logs-web

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
	@echo "Starting databases for DEVELOPMENT (preserving your .env settings)..."
	@docker start telar-mongo 2>/dev/null || \
		(echo "Creating MongoDB container..." && \
		 docker run -d --name telar-mongo -p 27017:27017 mongo:6)
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
	@echo "✅ Databases ready for development (your .env settings preserved)"

# Testing: Use test_env.sh (configures .env for consistent test environment)
up-mongo:
	@$(TEST_ENV_SCRIPT) up mongo

up-postgres:
	@$(TEST_ENV_SCRIPT) up postgres

up-both:
	@$(TEST_ENV_SCRIPT) up both

down-both:
	@$(TEST_ENV_SCRIPT) down both

clean-dbs:
	@echo "Recreating fresh database containers..."
	@$(TEST_ENV_SCRIPT) down both
	@$(TEST_ENV_SCRIPT) up both
	@echo "Databases are clean and ready."

status:
	@$(TEST_ENV_SCRIPT) status

logs-mongo:
	@$(TEST_ENV_SCRIPT) logs mongo

logs-postgres:
	@$(TEST_ENV_SCRIPT) logs postgres

docker-start:
	@./tools/dev/scripts/docker-start.sh

all: test-all

test-posts: up-both
	@echo "Testing 'posts' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./posts $(GO_TEST_FLAGS)

test-comments: up-both
	@echo "Testing 'comments' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./comments $(GO_TEST_FLAGS)

test-votes: up-both
	@echo "Testing 'votes' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./votes $(GO_TEST_FLAGS)

test-userrels: up-both
	@echo "Testing 'userrels' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./userrels $(GO_TEST_FLAGS)

test-auth: up-both
	@echo "Testing 'auth' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./auth $(GO_TEST_FLAGS)

test-profile: up-both
	@echo "Testing 'profile' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./profile/... $(GO_TEST_FLAGS)

test-circles: up-both
	@echo "Testing 'circles' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./circles $(GO_TEST_FLAGS)

test-setting: up-both
	@echo "Testing 'setting' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./setting $(GO_TEST_FLAGS)

test-admin: up-both
	@echo "Testing 'admin' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./admin $(GO_TEST_FLAGS)

test-gallery: up-both
	@echo "Testing 'gallery' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./gallery $(GO_TEST_FLAGS)

test-notifications: up-both
	@echo "Testing 'notifications' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./notifications $(GO_TEST_FLAGS)

test-actions: up-both
	@echo "Testing 'actions' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./actions $(GO_TEST_FLAGS)

test-storage: up-both
	@echo "Testing 'storage' microservice..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./storage $(GO_TEST_FLAGS)

test-cache:
	@echo "Testing internal cache..."
	@cd apps/api && go test ./internal/cache $(GO_TEST_FLAGS)

test-all: up-both
	@echo "Running all tests for all microservices with parallelism $(PARALLEL)..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./... $(GO_TEST_FLAGS)

test-all-race: up-both
	@echo "Running all tests with race detector..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./... -count=1 -v -race -timeout=20m -parallel=8

local-test-all: docker-start test-all

# --- CI/CD Targets ---

ci-fast: up-both test-posts test-comments test-votes test-userrels test-admin

ci-test: up-both test-posts test-comments test-votes test-userrels test-auth test-profile test-circles test-setting test-admin test-gallery test-notifications test-actions test-storage test-cache

ci-full: ci-test

ci-nightly: test-all

# --- Reporting & Profiling ---

report: up-both
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

bench-calibrated: up-both bench-env $(PROFILES_DIR)
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

up-both-replica:
	@./tools/dev/scripts/setup-mongo-replica.sh

test-transactions: up-both-replica
	@echo "Testing enterprise transaction management..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./internal/database/ -v -run TestTransactionSuite $(GO_TEST_FLAGS)

# --- Database Operations Testing ---

test-db-operations: up-both
	@echo "Testing PostgreSQL database operations..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./internal/database/postgresql -v -run TestOperations $(GO_TEST_FLAGS)

test-posts-operations: up-both
	@echo "Testing posts service operations..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./posts -v -run TestPostsOperations $(GO_TEST_FLAGS)

test-database-compatibility: up-both
	@echo "Testing database compatibility between MongoDB and PostgreSQL..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./posts -v -run TestDatabaseCompatibility $(GO_TEST_FLAGS)

bench-db-operations: up-both
	@echo "Benchmarking database operations..."
	@cd apps/api && RUN_DB_TESTS=1 go test ./internal/database/postgresql -bench=. -benchmem -run=^Benchmark $(GO_TEST_FLAGS)

test-all-operations: up-both
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
	@echo "  up-dbs-dev        - Start databases for DEVELOPMENT (preserves your .env settings)"
	@echo "  up-both           - Start databases for TESTING (configures .env for tests)"
	@echo "  up-mongo          - Start MongoDB only (for testing)"
	@echo "  up-postgres       - Start PostgreSQL only (for testing)"
	@echo "  down-both         - Stop all database containers"
	@echo "  clean-dbs         - Recreate fresh database containers (for testing)"
	@echo "  status            - Show status of Docker containers"
	@echo "  logs-mongo        - Tail logs for the MongoDB container"
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
	@echo "  test-database-compatibility - Test compatibility between MongoDB and PostgreSQL."
	@echo "  bench-db-operations       - Benchmark database operations performance."
	@echo "  test-all-operations       - Run all database operations tests."
	@echo ""
	@echo "Transaction Testing:"
	@echo "  up-both-replica   - Start databases with MongoDB replica set for transactions."
	@echo "  test-transactions - Run enterprise transaction management tests."
	@echo ""
	@echo "Development Servers:"
	@echo "  run-api           - Start the Telar API server on port 8080 (requires databases)."
	@echo "  run-profile       - Start the Profile microservice on port 8081 (requires databases)."
	@echo "  run-web           - Start the Next.js web frontend development server."
	@echo "  run-both          - Start both API and web frontend servers concurrently."
	@echo "  run-both-bg       - Start both servers in background (recommended for development)."
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
# NOTE: These use up-dbs-dev (NOT up-both) to preserve your .env settings

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

run-both-bg: up-dbs-dev
	@./tools/dev/scripts/start-servers-bg.sh

stop-servers:
	@./tools/dev/scripts/stop-servers.sh

restart-servers:
	@./tools/dev/scripts/restart-servers.sh


pre-flight-check:
	@./tools/dev/scripts/pre-flight-check.sh

logs-api:
	@echo "📋 Tailing API server logs (Ctrl+C to exit)..."
	@tail -f /tmp/telar-logs/api.log 2>/dev/null || echo "No API logs found. Is the server running?"

logs-web:
	@echo "📋 Tailing Web server logs (Ctrl+C to exit)..."
	@tail -f /tmp/telar-logs/web.log 2>/dev/null || echo "No Web logs found. Is the server running?"
