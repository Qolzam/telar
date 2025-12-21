-- Standard Full Text Search Index for English content
CREATE INDEX idx_posts_body_fts 
ON posts 
USING GIN (to_tsvector('english', body));






