-- Migration: Create comment_votes table for comment likes
-- This table stores user votes (likes) for comments
-- Composite primary key enforces one vote per user per comment
-- Foreign keys with ON DELETE CASCADE ensure referential integrity

CREATE TABLE IF NOT EXISTS comment_votes (
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    owner_user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Composite Primary Key enforces uniqueness (One like per user per comment)
    PRIMARY KEY (comment_id, owner_user_id)
);

-- Index for retrieving "My Liked Comments" efficiently
CREATE INDEX IF NOT EXISTS idx_comment_votes_owner ON comment_votes(owner_user_id);

-- Note: An index on comment_id is implicit in the Primary Key, so we don't need a separate one.

