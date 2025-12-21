-- Bookmark storage schema
CREATE TABLE IF NOT EXISTS bookmarks (
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    owner_user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (owner_user_id, post_id)
);

-- Optimize "My Bookmarks" feed ordered by newest saved
CREATE INDEX IF NOT EXISTS idx_bookmarks_owner_created ON bookmarks(owner_user_id, created_at DESC);

