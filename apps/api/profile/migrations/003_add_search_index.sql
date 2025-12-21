-- Standard Full Text Search Index for English profile content
-- Indexes full_name, social_name, and tagline for efficient search
CREATE INDEX idx_profiles_search_fts
ON profiles
USING GIN (to_tsvector('english',
    COALESCE(full_name, '') || ' ' ||
    COALESCE(social_name, '') || ' ' ||
    COALESCE(tagline, '')
));


