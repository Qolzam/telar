-- Migration: Fix url_key NULL constraint
-- This migration fixes existing NULL values and enforces NOT NULL constraint

-- 1. Fix existing corruption (Critical before adding constraint)
UPDATE posts SET url_key = concat('post-', id::text) WHERE url_key IS NULL;

-- 2. Enforce integrity
ALTER TABLE posts ALTER COLUMN url_key SET NOT NULL;
ALTER TABLE posts ALTER COLUMN url_key SET DEFAULT '';

