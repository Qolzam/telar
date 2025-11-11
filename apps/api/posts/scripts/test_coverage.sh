#!/bin/bash

# Test Coverage Reporting Script for Posts Microservice
# This script generates comprehensive test coverage reports for all test suites

set -e

echo "🧪 Posts Microservice - Comprehensive Test Coverage Report"
echo "=========================================================="
echo

# Configuration
POSTS_DIR="/Users/qolzam/projects/telar/telar/apps/api/posts"
REPORTS_DIR="$POSTS_DIR/reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create reports directory
mkdir -p "$REPORTS_DIR"

cd "$POSTS_DIR"

# Function to run tests with coverage
run_test_coverage() {
    local test_name="$1"
    local test_path="$2"
    local coverage_file="$REPORTS_DIR/${test_name}_coverage.out"
    local html_file="$REPORTS_DIR/${test_name}_coverage.html"
    
    echo "📋 Running $test_name tests..."
    
    if [ -d "$test_path" ] || [ -f "$test_path" ]; then
        go test -v -coverprofile="$coverage_file" "$test_path" || echo "⚠️  $test_name tests had issues"
        
        if [ -f "$coverage_file" ]; then
            # Generate HTML coverage report
            go tool cover -html="$coverage_file" -o "$html_file"
            
            # Get coverage percentage
            local coverage_percent=$(go tool cover -func="$coverage_file" | grep total | awk '{print $3}')
            echo "✅ $test_name coverage: $coverage_percent"
            echo "   📄 HTML report: $html_file"
            echo
        else
            echo "❌ No coverage file generated for $test_name"
            echo
        fi
    else
        echo "⚠️  $test_name test path not found: $test_path"
        echo
    fi
}

# Function to run benchmark tests
run_benchmark_tests() {
    local test_name="$1"
    local test_path="$2"
    local benchmark_file="$REPORTS_DIR/${test_name}_benchmark.out"
    
    echo "🚀 Running $test_name benchmarks..."
    
    if [ -d "$test_path" ] || [ -f "$test_path" ]; then
        go test -bench=. -benchmem -run=^$ "$test_path" > "$benchmark_file" 2>&1 || echo "⚠️  $test_name benchmarks had issues"
        
        if [ -f "$benchmark_file" ]; then
            echo "✅ $test_name benchmarks completed"
            echo "   📄 Report: $benchmark_file"
            echo
        else
            echo "❌ No benchmark file generated for $test_name"
            echo
        fi
    else
        echo "⚠️  $test_name test path not found: $test_path"
        echo
    fi
}

echo "🏁 Starting comprehensive test coverage analysis..."
echo

# 1. Service Layer Tests
run_test_coverage "Services" "./services"

# 2. Handler Tests
run_test_coverage "Handlers" "./handlers"

# 3. Model Tests
run_test_coverage "Models" "./models"

# 4. Error Handling Tests
run_test_coverage "Errors" "./errors"

# 5. Security Tests
run_test_coverage "Security" "./security"

# 6. Database Tests
run_test_coverage "Database" "./database"

# 7. Performance Tests (with benchmarks)
run_test_coverage "Performance" "./performance"
run_benchmark_tests "Performance" "./performance"

# 8. Configuration Tests
run_test_coverage "Configuration" "./config"

# 9. Integration Tests
run_test_coverage "Integration" "./posts_integration_test.go"

# 10. HTTP Compatibility Tests
run_test_coverage "HTTP" "./http_compat_test.go"

# 11. Handler Persistence Tests
run_test_coverage "Persistence" "./handlers_persistence_test.go"

# 12. Common Utilities Tests
run_test_coverage "Common" "./common"

echo "📊 Generating combined coverage report..."
echo

# Combine all coverage files
COMBINED_COVERAGE="$REPORTS_DIR/combined_coverage.out"
echo "mode: set" > "$COMBINED_COVERAGE"

# Merge coverage files (skip mode line from subsequent files)
for coverage_file in "$REPORTS_DIR"/*_coverage.out; do
    if [ -f "$coverage_file" ]; then
        tail -n +2 "$coverage_file" >> "$COMBINED_COVERAGE" 2>/dev/null || true
    fi
done

# Generate combined HTML report
if [ -f "$COMBINED_COVERAGE" ]; then
    COMBINED_HTML="$REPORTS_DIR/combined_coverage.html"
    go tool cover -html="$COMBINED_COVERAGE" -o "$COMBINED_HTML"
    
    # Get total coverage percentage
    TOTAL_COVERAGE=$(go tool cover -func="$COMBINED_COVERAGE" | grep total | awk '{print $3}' || echo "N/A")
    
    echo "🎯 OVERALL POSTS MICROSERVICE COVERAGE: $TOTAL_COVERAGE"
    echo "   📄 Combined HTML report: $COMBINED_HTML"
    echo
fi

# Generate summary report
SUMMARY_FILE="$REPORTS_DIR/coverage_summary_$TIMESTAMP.txt"
echo "Posts Microservice Test Coverage Summary" > "$SUMMARY_FILE"
echo "Generated: $(date)" >> "$SUMMARY_FILE"
echo "=========================================" >> "$SUMMARY_FILE"
echo >> "$SUMMARY_FILE"

echo "📋 Individual Test Suite Coverage:" >> "$SUMMARY_FILE"
for coverage_file in "$REPORTS_DIR"/*_coverage.out; do
    if [ -f "$coverage_file" ]; then
        test_name=$(basename "$coverage_file" _coverage.out)
        coverage_percent=$(go tool cover -func="$coverage_file" | grep total | awk '{print $3}' 2>/dev/null || echo "N/A")
        echo "  - $test_name: $coverage_percent" >> "$SUMMARY_FILE"
    fi
done

echo >> "$SUMMARY_FILE"
echo "🎯 Overall Coverage: $TOTAL_COVERAGE" >> "$SUMMARY_FILE"
echo >> "$SUMMARY_FILE"

# Add test file counts
echo "📁 Test File Analysis:" >> "$SUMMARY_FILE"
echo "  - Total test files: $(find . -name "*_test.go" | wc -l)" >> "$SUMMARY_FILE"
echo "  - Service tests: $(find ./services -name "*_test.go" 2>/dev/null | wc -l || echo 0)" >> "$SUMMARY_FILE"
echo "  - Handler tests: $(find ./handlers -name "*_test.go" 2>/dev/null | wc -l || echo 0)" >> "$SUMMARY_FILE"
echo "  - Model tests: $(find ./models -name "*_test.go" 2>/dev/null | wc -l || echo 0)" >> "$SUMMARY_FILE"
echo "  - Integration tests: $(find . -maxdepth 1 -name "*integration*_test.go" | wc -l)" >> "$SUMMARY_FILE"
echo "  - Performance tests: $(find ./performance -name "*_test.go" 2>/dev/null | wc -l || echo 0)" >> "$SUMMARY_FILE"
echo "  - Security tests: $(find ./security -name "*_test.go" 2>/dev/null | wc -l || echo 0)" >> "$SUMMARY_FILE"
echo "  - Database tests: $(find ./database -name "*_test.go" 2>/dev/null | wc -l || echo 0)" >> "$SUMMARY_FILE"
echo "  - Configuration tests: $(find ./config -name "*_test.go" 2>/dev/null | wc -l || echo 0)" >> "$SUMMARY_FILE"

echo >> "$SUMMARY_FILE"
echo "📄 Generated Reports:" >> "$SUMMARY_FILE"
ls -la "$REPORTS_DIR"/*.html "$REPORTS_DIR"/*.out 2>/dev/null | awk '{print "  - " $9}' >> "$SUMMARY_FILE" 2>/dev/null || true

echo "📄 Coverage summary saved to: $SUMMARY_FILE"
echo

# Display summary
echo "📋 FINAL SUMMARY:"
echo "==================="
cat "$SUMMARY_FILE"

echo
echo "🎉 Test coverage analysis completed!"
echo "📂 All reports saved in: $REPORTS_DIR"
echo

# Optional: Open combined coverage report in browser (macOS)
if command -v open >/dev/null 2>&1 && [ -f "$COMBINED_HTML" ]; then
    echo "🌐 Opening combined coverage report in browser..."
    open "$COMBINED_HTML"
fi
