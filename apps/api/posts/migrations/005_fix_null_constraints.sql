-- Migration: Fix NULL constraints for all string fields
-- This migration fixes existing NULL values and enforces NOT NULL constraints
-- to match the Go struct definitions (string fields are non-nullable in Go)

-- 1. Sanitize Owner Display Names (Critical - causes scan errors)
UPDATE posts 
SET owner_display_name = 'Unknown User' 
WHERE owner_display_name IS NULL;

-- 2. Sanitize Owner Avatars
UPDATE posts 
SET owner_avatar = '' 
WHERE owner_avatar IS NULL;

-- 3. Sanitize Image fields (optional fields, but must not be NULL)
UPDATE posts 
SET image = '' 
WHERE image IS NULL;

UPDATE posts 
SET image_full_path = '' 
WHERE image_full_path IS NULL;

-- 4. Sanitize Video fields
UPDATE posts 
SET video = '' 
WHERE video IS NULL;

UPDATE posts 
SET thumbnail = '' 
WHERE thumbnail IS NULL;

-- 5. Sanitize Version (optional field, but must not be NULL)
UPDATE posts 
SET version = '' 
WHERE version IS NULL;

-- 6. Sanitize Body (should never be NULL, but legacy data might have it)
UPDATE posts 
SET body = '' 
WHERE body IS NULL;

-- 7. Sanitize Permission (should have default, but ensure no NULLs)
UPDATE posts 
SET permission = 'Public' 
WHERE permission IS NULL;

-- 8. Enforce Integrity - Add NOT NULL constraints with defaults
ALTER TABLE posts ALTER COLUMN owner_display_name SET NOT NULL;
ALTER TABLE posts ALTER COLUMN owner_display_name SET DEFAULT 'Unknown User';

ALTER TABLE posts ALTER COLUMN owner_avatar SET NOT NULL;
ALTER TABLE posts ALTER COLUMN owner_avatar SET DEFAULT '';

ALTER TABLE posts ALTER COLUMN image SET NOT NULL;
ALTER TABLE posts ALTER COLUMN image SET DEFAULT '';

ALTER TABLE posts ALTER COLUMN image_full_path SET NOT NULL;
ALTER TABLE posts ALTER COLUMN image_full_path SET DEFAULT '';

ALTER TABLE posts ALTER COLUMN video SET NOT NULL;
ALTER TABLE posts ALTER COLUMN video SET DEFAULT '';

ALTER TABLE posts ALTER COLUMN thumbnail SET NOT NULL;
ALTER TABLE posts ALTER COLUMN thumbnail SET DEFAULT '';

ALTER TABLE posts ALTER COLUMN version SET NOT NULL;
ALTER TABLE posts ALTER COLUMN version SET DEFAULT '';

ALTER TABLE posts ALTER COLUMN body SET NOT NULL;
ALTER TABLE posts ALTER COLUMN body SET DEFAULT '';

ALTER TABLE posts ALTER COLUMN permission SET NOT NULL;
-- Permission already has default 'Public', so just enforce NOT NULL

