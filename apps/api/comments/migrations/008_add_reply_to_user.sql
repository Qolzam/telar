-- Migration: 008_add_reply_to_user.sql
-- Description: Adds reply_to_user_id column to support two-tier comment architecture
-- Dependencies: Requires comments table (005_create_comments_table.sql) and user_auths table (003_create_auth_tables.sql)
-- Purpose: Track which user is being replied to, even when all replies point to the root comment

-- Add reply_to_user_id column to track who is being addressed in a reply
ALTER TABLE comments 
ADD COLUMN IF NOT EXISTS reply_to_user_id UUID REFERENCES user_auths(id) ON DELETE SET NULL;

-- Track display name of the user being replied to for quick rendering
ALTER TABLE comments 
ADD COLUMN IF NOT EXISTS reply_to_display_name VARCHAR(255);

-- Index for "My Mentions" notifications and user-specific reply queries
CREATE INDEX IF NOT EXISTS idx_comments_reply_to_user ON comments(reply_to_user_id) WHERE reply_to_user_id IS NOT NULL;

-- Comment: This migration supports the two-tier architecture where:
-- - parent_comment_id always points to the root comment (or NULL for root comments)
-- - reply_to_user_id tracks the specific user being addressed (for UI display "Replying to @John")




