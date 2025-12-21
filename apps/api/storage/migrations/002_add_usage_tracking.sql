-- Storage usage tracking for quota enforcement
-- Tracks daily upload/download counts per user to prevent abuse

-- Track usage per user per day
CREATE TABLE IF NOT EXISTS storage_usage_daily (
    user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
    day DATE NOT NULL DEFAULT CURRENT_DATE,
    upload_count INT DEFAULT 0,
    download_count INT DEFAULT 0,
    total_bytes_uploaded BIGINT DEFAULT 0,
    
    PRIMARY KEY (user_id, day)
);

-- Index for efficient daily quota lookups
CREATE INDEX IF NOT EXISTS idx_storage_usage_daily_user_day ON storage_usage_daily(user_id, day DESC);

-- Global circuit breaker (Approximate stats, single row)
CREATE TABLE IF NOT EXISTS storage_system_stats (
    id INT PRIMARY KEY DEFAULT 1,
    total_files BIGINT DEFAULT 0,
    total_storage_bytes BIGINT DEFAULT 0,
    last_gc_run TIMESTAMPTZ,
    CHECK (id = 1)
);

-- Initialize system stats row if it doesn't exist
INSERT INTO storage_system_stats (id, total_files, total_storage_bytes, last_gc_run)
VALUES (1, 0, 0, NOW())
ON CONFLICT (id) DO NOTHING;



