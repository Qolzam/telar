#!/bin/bash
# Preload Ollama models required for AI Engine
# This script ensures required models are available before the service starts

set -e

OLLAMA_URL="${OLLAMA_BASE_URL:-http://ollama:11434}"
MODEL="${OLLAMA_CLASSIFICATION_MODEL:-qwen2.5:1.5b}"
MAX_RETRIES=30
RETRY_INTERVAL=2

echo "Waiting for Ollama to be ready..."
for i in $(seq 1 $MAX_RETRIES); do
    if curl -s -f "${OLLAMA_URL}/api/tags" > /dev/null 2>&1; then
        echo "✓ Ollama is ready"
        break
    fi
    if [ $i -eq $MAX_RETRIES ]; then
        echo "✗ Ollama failed to start within expected time"
        exit 1
    fi
    sleep $RETRY_INTERVAL
done

echo "Checking if model '${MODEL}' is available..."
if curl -s -f "${OLLAMA_URL}/api/tags" | grep -q "\"${MODEL}\""; then
    echo "✓ Model '${MODEL}' is already available"
    exit 0
fi

echo "Pulling model '${MODEL}'..."
curl -X POST "${OLLAMA_URL}/api/pull" -d "{\"name\": \"${MODEL}\"}"

echo "✓ Model '${MODEL}' pull initiated (this may take several minutes)"
echo "Note: The model will be available once the pull completes"


