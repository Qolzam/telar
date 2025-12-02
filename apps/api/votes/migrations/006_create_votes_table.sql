-- Migration: 006_create_votes_table.sql
-- Description: Creates votes table for votes service
-- Dependencies: Requires posts table (001_create_posts_table.sql) and user_auths table (003_create_auth_tables.sql)
-- Reference: StackExchange Data Explorer (Votes, VoteTypes)

-- Table: votes
-- Purpose: Stores user votes on posts with atomic constraints to prevent race conditions
CREATE TABLE IF NOT EXISTS votes (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL,
    owner_user_id UUID NOT NULL,
    vote_type_id SMALLINT NOT NULL CHECK (vote_type_id IN (1, 2)), -- 1=UpVote, 2=DownVote
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Critical: Unique constraint prevents duplicate votes (one vote per user per post)
-- This enforces data integrity at the database level, preventing race condition duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_unique_user_post ON votes(post_id, owner_user_id);

-- Index for counting/fetching votes by post (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_votes_post_id ON votes(post_id);

-- Index for fetching votes by user (for user activity queries)
CREATE INDEX IF NOT EXISTS idx_votes_owner_user_id ON votes(owner_user_id);

-- Index for vote type queries (if needed for analytics)
CREATE INDEX IF NOT EXISTS idx_votes_vote_type_id ON votes(vote_type_id);

-- Note: Foreign key constraints are handled by application logic
-- If FK constraints are needed, uncomment below:
-- ALTER TABLE votes ADD CONSTRAINT fk_votes_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE;
-- ALTER TABLE votes ADD CONSTRAINT fk_votes_user FOREIGN KEY (owner_user_id) REFERENCES user_auths(id) ON DELETE CASCADE;

