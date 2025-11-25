-- Migration: 005_create_comments_table.sql
-- Description: Creates comments table for comments service
-- Dependencies: Requires posts table (001_create_posts_table.sql) and user_auths table (003_create_auth_tables.sql)

-- Table: comments
-- Purpose: Stores comments on posts, with support for nested replies via parent_comment_id
CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    owner_user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    score BIGINT DEFAULT 0,
    owner_display_name VARCHAR(255),
    owner_avatar VARCHAR(512),
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_date BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
);

-- Indexes for comments
-- Critical: post_id index for fetching comments by post (most common query)
CREATE INDEX IF NOT EXISTS idx_comments_post ON comments(post_id);
-- Index for fetching replies to a specific comment
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_comment_id) WHERE parent_comment_id IS NOT NULL;
-- Index for fetching comments by user
CREATE INDEX IF NOT EXISTS idx_comments_owner ON comments(owner_user_id);
-- Index for ordering comments by creation time (most recent first)
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comments_created_date ON comments(created_date DESC);
-- Index for filtering out deleted comments (common query pattern)
CREATE INDEX IF NOT EXISTS idx_comments_deleted ON comments(is_deleted) WHERE is_deleted = FALSE;
-- Composite index for common query: post + not deleted + created date (for pagination)
CREATE INDEX IF NOT EXISTS idx_comments_post_active ON comments(post_id, created_date DESC) WHERE is_deleted = FALSE;

