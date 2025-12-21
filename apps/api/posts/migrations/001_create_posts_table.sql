-- Migration: Create posts table
-- This migration creates a normalized, relational schema for posts
-- replacing the generic JSONB blob storage pattern

CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL,
    post_type_id INT NOT NULL,
    body TEXT,
    score BIGINT DEFAULT 0,
    view_count BIGINT DEFAULT 0,
    comment_count BIGINT DEFAULT 0,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_date BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    tags TEXT[],
    url_key VARCHAR(255),
    owner_display_name VARCHAR(255),
    owner_avatar VARCHAR(512),
    image VARCHAR(512),
    image_full_path VARCHAR(512),
    video VARCHAR(512),
    thumbnail VARCHAR(512),
    disable_comments BOOLEAN DEFAULT FALSE,
    disable_sharing BOOLEAN DEFAULT FALSE,
    permission VARCHAR(50) DEFAULT 'Public',
    version VARCHAR(50),
    -- Only use JSONB for truly dynamic data that isn't queried often
    -- (e.g., votes map, album, access_user_list)
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_posts_owner ON posts(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_created_date ON posts(created_date DESC);
CREATE INDEX IF NOT EXISTS idx_posts_tags ON posts USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_posts_post_type ON posts(post_type_id);
CREATE INDEX IF NOT EXISTS idx_posts_deleted ON posts(is_deleted) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_posts_url_key ON posts(url_key) WHERE url_key IS NOT NULL;

