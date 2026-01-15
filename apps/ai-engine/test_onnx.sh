#!/bin/bash
# Performance Proof Test Script for ONNX Integration
# This script tests the tiered moderation pipeline and captures performance metrics

set -e

API_URL="http://localhost:9066/api/v1/analyze/content"
API_KEY="${INTERNAL_API_KEY:-test-key}"

echo "=========================================="
echo "ONNX Performance Proof Test"
echo "=========================================="
echo ""

# Test 1: Toxic Content (High Confidence ONNX)
echo "Test 1: Toxic Content (Expected: L3-ONNX-Local, < 50ms)"
echo "Request: 'You are an idiot'"
echo "---"
START_TIME=$(date +%s%N)
RESPONSE=$(curl -s -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -H "X-Internal-API-Key: $API_KEY" \
  -d '{"content": "You are an idiot"}')
END_TIME=$(date +%s%N)
ELAPSED_MS=$(( (END_TIME - START_TIME) / 1000000 ))

echo "Response Time: ${ELAPSED_MS}ms"
echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""
echo "---"
echo ""

# Test 2: Ambiguous Content (Low Confidence ONNX -> LLM Fallback)
echo "Test 2: Ambiguous Content (Expected: L4-Semantic-Model, ~3000ms)"
echo "Request: 'I strongly disagree with this approach.'"
echo "---"
START_TIME=$(date +%s%N)
RESPONSE=$(curl -s -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -H "X-Internal-API-Key: $API_KEY" \
  -d '{"content": "I strongly disagree with this approach."}')
END_TIME=$(date +%s%N)
ELAPSED_MS=$(( (END_TIME - START_TIME) / 1000000 ))

echo "Response Time: ${ELAPSED_MS}ms"
echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""
echo "---"
echo ""

# Test 3: Safe Content (High Confidence ONNX Safe)
echo "Test 3: Safe Content (Expected: L3-ONNX-Local, < 50ms)"
echo "Request: 'I love this community.'"
echo "---"
START_TIME=$(date +%s%N)
RESPONSE=$(curl -s -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -H "X-Internal-API-Key: $API_KEY" \
  -d '{"content": "I love this community."}')
END_TIME=$(date +%s%N)
ELAPSED_MS=$(( (END_TIME - START_TIME) / 1000000 ))

echo "Response Time: ${ELAPSED_MS}ms"
echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""
echo "=========================================="
echo "Test Complete"
echo "=========================================="


