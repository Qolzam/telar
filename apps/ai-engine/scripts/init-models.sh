#!/bin/bash
set -e

MODEL_DIR="/app/models/distilbert"
REQUIRED_FILES=("model.onnx" "model.onnx.data" "tokenizer.json" "vocab.txt")

# R2 Configuration (using existing environment variables)
R2_ENDPOINT="${R2_ENDPOINT:-}"
R2_BUCKET_NAME="${R2_BUCKET_NAME:-}"
R2_ACCESS_KEY_ID="${R2_ACCESS_KEY_ID:-}"
R2_SECRET_ACCESS_KEY="${R2_SECRET_ACCESS_KEY:-}"
R2_PUBLIC_URL="${R2_PUBLIC_URL:-}"
R2_PREFIX="${R2_MODEL_PREFIX:-models/distilbert/}"  # Path prefix in bucket

echo "[INIT] Checking AI Models..."

# Create directory if missing
if [ ! -d "$MODEL_DIR" ]; then
    echo "[INIT] Creating model directory..."
    mkdir -p "$MODEL_DIR"
fi

# Check if all required files exist
check_models_exist() {
    local missing=0
    for file in "${REQUIRED_FILES[@]}"; do
        if [ ! -f "$MODEL_DIR/$file" ]; then
            missing=1
            break
        fi
    done
    return $missing
}

# Download from R2 using AWS CLI (S3-compatible)
download_from_r2() {
    # Check if R2 is configured
    if [ -z "$R2_ENDPOINT" ] || [ -z "$R2_BUCKET_NAME" ] || [ -z "$R2_ACCESS_KEY_ID" ] || [ -z "$R2_SECRET_ACCESS_KEY" ]; then
        echo "[INIT] R2 credentials not configured, skipping R2 download"
        return 1
    fi

    echo "[INIT] Attempting to download models from R2..."
    
    # Install AWS CLI if not present (lightweight, one-time)
    if ! command -v aws &> /dev/null; then
        echo "[INIT] Installing AWS CLI for R2 access..."
        apt-get update -qq && apt-get install -y --no-install-recommends awscli && rm -rf /var/lib/apt/lists/*
    fi

    # Configure AWS CLI for R2
    export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
    export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
    export AWS_DEFAULT_REGION="auto"
    
    # Download each file
    local success=0
    for file in "${REQUIRED_FILES[@]}"; do
        local target="$MODEL_DIR/$file"
        local r2_path="s3://$R2_BUCKET_NAME/$R2_PREFIX$file"
        
        echo "[INIT] Downloading $file from R2..."
        if aws s3 cp "$r2_path" "$target" --endpoint-url="$R2_ENDPOINT" 2>/dev/null; then
            echo "[INIT] ✓ Downloaded $file"
            success=1
        else
            echo "[INIT] ✗ Failed to download $file from R2"
        fi
    done
    
    # If at least one file was downloaded, consider it a partial success
    if [ $success -eq 1 ]; then
        return 0
    fi
    
    return 1
}

# Alternative: Download from R2 Public URL (if configured)
download_from_r2_public() {
    if [ -z "$R2_PUBLIC_URL" ]; then
        return 1
    fi

    echo "[INIT] Attempting to download models from R2 Public URL..."
    
    local success=0
    for file in "${REQUIRED_FILES[@]}"; do
        local target="$MODEL_DIR/$file"
        local url="${R2_PUBLIC_URL%/}/${R2_PREFIX%/}${file}"
        
        echo "[INIT] Downloading $file from $url..."
        if wget -q --show-progress -O "$target" "$url" 2>/dev/null; then
            echo "[INIT] ✓ Downloaded $file"
            success=1
        else
            echo "[INIT] ✗ Failed to download $file from public URL"
        fi
    done
    
    return $success
}

# Main provisioning logic
if check_models_exist; then
    echo "[INIT] ✓ All model files present"
else
    echo "[INIT] Model files missing, attempting to provision..."
    
    # Strategy 1: Try R2 S3 API first (production)
    if [ -z "$SKIP_MODEL_DOWNLOAD" ]; then
        if download_from_r2; then
            echo "[INIT] ✓ Models provisioned from R2 (S3 API)"
        elif download_from_r2_public; then
            echo "[INIT] ✓ Models provisioned from R2 (Public URL)"
        else
            echo "[INIT] R2 download failed or not configured"
            echo "[INIT] Models must be provided via volume mount or exported via model-builder service"
            echo "[INIT] Continuing without ONNX models (L3 layer will be disabled)"
        fi
    else
        echo "[INIT] SKIP_MODEL_DOWNLOAD=true, expecting models via volume mount"
        if ! check_models_exist; then
            echo "[INIT] WARNING: Models not found in volume mount. L3 ONNX layer will be disabled."
        fi
    fi
fi

# Hand over control to the Go Binary
echo "[INIT] Starting AI Engine..."
exec "$@"
