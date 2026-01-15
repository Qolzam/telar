#!/bin/bash
# Initialize Ollama with required models
# This runs as a separate init container or can be called after Ollama starts
# Pulls all required models based on provider configuration

set -e

OLLAMA_URL="${OLLAMA_BASE_URL:-http://ollama:11434}"
MAX_RETRIES=60
RETRY_INTERVAL=2

# Warmup behavior control
# Set to "true" to warmup all models (including existing ones), "false" to skip warmup for existing models
WARMUP_EXISTING_MODELS="${WARMUP_EXISTING_MODELS:-false}"
# Timeout for warmup: shorter for existing models, longer for newly pulled models
WARMUP_TIMEOUT_EXISTING="${WARMUP_TIMEOUT_EXISTING:-15}"  # 15 seconds for existing models
WARMUP_TIMEOUT_NEW="${WARMUP_TIMEOUT_NEW:-120}"  # 120 seconds for newly pulled models

# Check which features use Ollama
KNOWLEDGE_EMBEDDING_PROVIDER="${KNOWLEDGE_EMBEDDING_PROVIDER:-ollama}"
GENERATOR_PROVIDER="${GENERATOR_PROVIDER:-ollama}"
MODERATION_FALLBACK_PROVIDER="${MODERATION_FALLBACK_PROVIDER:-ollama}"

# Collect models to pull based on which features use Ollama
MODELS_TO_PULL=()

# If using Ollama for knowledge embeddings, pull embedding model
if [ "$KNOWLEDGE_EMBEDDING_PROVIDER" = "ollama" ]; then
    KNOWLEDGE_EMBEDDING_MODEL="${KNOWLEDGE_EMBEDDING_MODEL:-nomic-embed-text}"
    MODELS_TO_PULL+=("$KNOWLEDGE_EMBEDDING_MODEL")
fi

# If using Ollama for generator, pull generator model
if [ "$GENERATOR_PROVIDER" = "ollama" ]; then
    GENERATOR_MODEL="${GENERATOR_MODEL:-llama3:8b}"
    MODELS_TO_PULL+=("$GENERATOR_MODEL")
fi

# If using Ollama for moderation fallback, pull moderation fallback model
if [ "$MODERATION_FALLBACK_PROVIDER" = "ollama" ]; then
    MODERATION_FALLBACK_MODEL="${MODERATION_FALLBACK_MODEL:-qwen2.5:1.5b}"
    MODELS_TO_PULL+=("$MODERATION_FALLBACK_MODEL")
fi

# Remove duplicates
MODELS_TO_PULL=($(printf '%s\n' "${MODELS_TO_PULL[@]}" | sort -u))

# If no Ollama models needed, exit early
if [ ${#MODELS_TO_PULL[@]} -eq 0 ]; then
    echo "No Ollama models required (KNOWLEDGE_EMBEDDING_PROVIDER=$KNOWLEDGE_EMBEDDING_PROVIDER, GENERATOR_PROVIDER=$GENERATOR_PROVIDER, MODERATION_FALLBACK_PROVIDER=$MODERATION_FALLBACK_PROVIDER)"
    exit 0
fi

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
    echo "  Waiting... (${i}/${MAX_RETRIES})"
    sleep $RETRY_INTERVAL
done

# Give Ollama time to fully initialize and index existing models from volume
# This prevents false negatives when checking for existing models
# Ollama may be "ready" but models from volume may not be indexed yet
echo "Waiting for Ollama to index existing models from volume..."
max_index_wait=30  # Maximum 30 seconds to wait for model indexing
index_wait_interval=2
waited=0
models_indexed=false

while [ $waited -lt $max_index_wait ]; do
    tags_response=$(curl -s "${OLLAMA_URL}/api/tags" 2>/dev/null)
    if [ -n "$tags_response" ]; then
        # Check if response contains models array and has at least one model
        model_count=$(echo "$tags_response" | grep -o '"name"' | wc -l)
        if [ "$model_count" -gt 0 ]; then
            echo "✓ Models indexed (found $model_count model(s) in Ollama)"
            models_indexed=true
            break
        fi
    fi
    if [ $((waited % 6)) -eq 0 ] && [ $waited -gt 0 ]; then
        echo "  Still waiting for models to be indexed... (${waited}s elapsed)"
    fi
    sleep $index_wait_interval
    waited=$((waited + index_wait_interval))
done

if [ "$models_indexed" = false ]; then
    echo "⚠ Warning: Models may not be fully indexed yet, but proceeding with checks"
    echo "  (This is normal if no models exist in the volume yet)"
fi

# Helper function to check if model exists
# Retries up to 3 times to handle cases where Ollama is still indexing models
model_exists() {
    local model="$1"
    local retries=3
    local retry_delay=2
    
    # Check for exact match or match with :latest tag
    # Ollama API returns model names, and they may have tags
    local model_base="${model%%:*}"  # Remove tag if present
    
    for attempt in $(seq 1 $retries); do
        # Check if model exists (with or without tag) in the response
        local tags_response=$(curl -s "${OLLAMA_URL}/api/tags" 2>/dev/null)
        if [ -z "$tags_response" ]; then
            if [ $attempt -lt $retries ]; then
                sleep $retry_delay
                continue
            fi
            return 1
        fi
        
        # Extract all model names from the JSON response
        # Ollama API returns: {"models": [{"name": "model:tag", ...}, ...]}
        # Use a more robust approach: extract model names and check against them
        local model_names=$(echo "$tags_response" | grep -oE '"name"\s*:\s*"[^"]+"' | sed -E 's/"name"\s*:\s*"([^"]+)"/\1/' 2>/dev/null)
        
        if [ -z "$model_names" ]; then
            if [ $attempt -lt $retries ]; then
                sleep $retry_delay
                continue
            fi
            return 1
        fi
        
        # Check for exact model name match
        if echo "$model_names" | grep -qFx "$model" 2>/dev/null; then
            return 0
        fi
        
        # Check for match with model base name (handles :latest tag)
        if echo "$model_names" | grep -qE "^${model_base}(:.*)?$" 2>/dev/null; then
            return 0
        fi
        
        # Check for match with :latest tag if no tag was specified in the model name
        if [ "$model" = "$model_base" ]; then
            if echo "$model_names" | grep -qE "^${model_base}:latest$" 2>/dev/null; then
                return 0
            fi
        fi
        
        # If we get here and this isn't the last attempt, retry
        if [ $attempt -lt $retries ]; then
            sleep $retry_delay
        fi
    done
    
    return 1
}

# Helper function to pull a model and wait for completion
pull_model() {
    local model="$1"
    echo "Pulling model '${model}' (this may take several minutes)..."
    
    # Start pull in background (Ollama API streams progress)
    (curl -s -X POST "${OLLAMA_URL}/api/pull" -d "{\"name\": \"${model}\"}" > /dev/null 2>&1) &
    local pull_pid=$!
    
    # Monitor progress by checking if model appears in list
    local max_wait=1800  # 30 minutes max wait
    local waited=0
    local check_interval=5
    
    while [ $waited -lt $max_wait ]; do
        sleep $check_interval
        waited=$((waited + check_interval))
        
        # Check if model now exists
        if model_exists "$model"; then
            echo "  ✓ Model '${model}' pulled successfully (took ${waited}s)"
            wait $pull_pid 2>/dev/null || true
            return 0
        fi
        
        # Check if pull process finished (successfully or not)
        if ! kill -0 $pull_pid 2>/dev/null; then
            # Process finished, wait for it and check final status
            wait $pull_pid 2>/dev/null || true
            # Give Ollama a moment to update its model list
            sleep 3
            if model_exists "$model"; then
                echo "  ✓ Model '${model}' pulled successfully (took ${waited}s)"
                return 0
            else
                # Check one more time after a longer wait
                echo "  Checking model status again..."
                sleep 5
                if model_exists "$model"; then
                    echo "  ✓ Model '${model}' is now available"
                    return 0
                else
                    echo "  ⚠ Model '${model}' pull completed but model not found in list"
                    echo "     This may be a timing issue. Model might still be available."
                    echo "     Check manually with: docker exec telar-ollama ollama list"
                    # Don't fail - model might still work
                    return 0
                fi
            fi
        fi
        
        # Show progress every 30 seconds
        if [ $((waited % 30)) -eq 0 ]; then
            echo "  Still pulling '${model}'... (${waited}s elapsed)"
        fi
    done
    
    # Timeout - kill pull process and check final status
    kill $pull_pid 2>/dev/null || true
    wait $pull_pid 2>/dev/null || true
    
    if model_exists "$model"; then
        echo "  ✓ Model '${model}' pulled successfully (took ${waited}s)"
        return 0
    else
        echo "  ⚠ Model '${model}' pull timed out after ${waited}s"
        echo "     Model may still be downloading. Check with: docker exec telar-ollama ollama list"
        return 1
    fi
}

# Helper function to warm up a model (ensure it's loaded and ready)
warmup_model() {
    local model="$1"
    local timeout="$2"  # Timeout in seconds
    local is_existing="$3"  # "true" if model already existed, "false" if newly pulled
    
    if [ "$is_existing" = "true" ] && [ "$WARMUP_EXISTING_MODELS" != "true" ]; then
        echo "  ⏭ Skipping warmup for existing model '${model}' (set WARMUP_EXISTING_MODELS=true to enable)"
        return 0
    fi
    
    if [ "$is_existing" = "true" ]; then
        echo "Warming up existing model '${model}' (quick check, ${timeout}s timeout)..."
    else
        echo "Warming up newly pulled model '${model}' (ensuring it's loaded, ${timeout}s timeout)..."
    fi
    
    # Make a small test request to ensure the model is loaded into memory
    local warmup_response=$(curl -s --max-time "$timeout" -X POST "${OLLAMA_URL}/api/generate" \
        -H "Content-Type: application/json" \
        -d "{\"model\":\"${model}\",\"prompt\":\"test\",\"stream\":false}" 2>&1)
    
    if echo "$warmup_response" | grep -q "\"response\""; then
        echo "  ✓ Model '${model}' is warmed up and ready"
        return 0
    else
        if [ "$is_existing" = "true" ]; then
            # For existing models, warmup failure is not critical
            echo "  ⚠ Model '${model}' warmup check had issues, but model exists and should work (response: ${warmup_response:0:100})"
            return 0
        else
            echo "  ⚠ Model '${model}' warmup had issues (response: ${warmup_response:0:100})"
            return 1
        fi
    fi
}

# Process each model
failed_models=()
for model in "${MODELS_TO_PULL[@]}"; do
    echo "Checking if model '${model}' is available..."
    if model_exists "$model"; then
        echo "✓ Model '${model}' is already available (skipping pull)"
        # Optionally warm up existing models (disabled by default for speed)
        set +e
        warmup_model "$model" "$WARMUP_TIMEOUT_EXISTING" "true"
        set -e
    else
        echo "Pulling model '${model}'..."
        # Temporarily disable set -e for this call
        set +e
        pull_model "$model"
        pull_result=$?
        set -e
        
        if [ $pull_result -ne 0 ]; then
            echo "⚠ Warning: Model '${model}' pull had issues"
            failed_models+=("$model")
        else
            # Always warm up newly pulled models to ensure they're ready
            set +e
            warmup_model "$model" "$WARMUP_TIMEOUT_NEW" "false"
            set -e
        fi
    fi
done

# Final check - verify all models are available
echo ""
echo "Verifying all required models are available..."
all_available=true
for model in "${MODELS_TO_PULL[@]}"; do
    if model_exists "$model"; then
        echo "✓ Model '${model}' is available"
    else
        echo "✗ Model '${model}' is NOT available"
        all_available=false
    fi
done

if [ "$all_available" = true ]; then
    echo ""
    echo "✓ Ollama model initialization complete - all models available"
    exit 0
else
    echo ""
    echo "⚠ Warning: Some models may not be available:"
    for model in "${failed_models[@]}"; do
        echo "   - ${model}"
    done
    echo "   The service may still work if models are pulled manually."
    echo "   Check with: docker exec telar-ollama ollama list"
    # Don't fail - allow service to start and handle missing models gracefully
    exit 0
fi


