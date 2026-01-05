-- Migration: Add moderation columns to posts table
-- Adds status and moderation_details columns to support AI-powered moderation

ALTER TABLE posts ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'published';
ALTER TABLE posts ADD COLUMN IF NOT EXISTS moderation_details JSONB;

-- Index for efficient querying of posts needing moderation
CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);

-- Add comment explaining status values
COMMENT ON COLUMN posts.status IS 'Post moderation status: published, needs_moderation, rejected, approved';
COMMENT ON COLUMN posts.moderation_details IS 'JSONB containing AI moderation analysis results and metadata';



