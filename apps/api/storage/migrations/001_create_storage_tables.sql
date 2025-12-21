-- Storage schema for file metadata
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(512) NOT NULL,
    mime_type VARCHAR(127) NOT NULL,
    size_bytes BIGINT NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'r2',
    bucket VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Optimize queries by owner
CREATE INDEX IF NOT EXISTS idx_files_owner ON files(owner_user_id);

-- Critical for Garbage Collection (quota enforcement)
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at);

-- Optimize status-based queries (e.g., finding pending files)
CREATE INDEX IF NOT EXISTS idx_files_status ON files(status);



