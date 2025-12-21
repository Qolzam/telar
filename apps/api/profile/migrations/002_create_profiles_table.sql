-- Migration: Create profiles table
-- This migration creates a normalized, relational schema for profiles
-- replacing the generic JSONB blob storage pattern

CREATE TABLE IF NOT EXISTS profiles (
    user_id UUID PRIMARY KEY,
    full_name VARCHAR(255),
    social_name VARCHAR(255),
    email VARCHAR(255),
    avatar VARCHAR(512),
    banner VARCHAR(512),
    tagline VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    last_seen BIGINT DEFAULT 0,
    birthday BIGINT DEFAULT 0,
    web_url VARCHAR(512),
    company_name VARCHAR(255),
    country VARCHAR(100),
    address TEXT,
    phone VARCHAR(50),
    vote_count BIGINT DEFAULT 0,
    share_count BIGINT DEFAULT 0,
    follow_count BIGINT DEFAULT 0,
    follower_count BIGINT DEFAULT 0,
    post_count BIGINT DEFAULT 0,
    facebook_id VARCHAR(255),
    instagram_id VARCHAR(255),
    twitter_id VARCHAR(255),
    linkedin_id VARCHAR(255),
    access_user_list TEXT[],
    permission VARCHAR(50) DEFAULT 'Public'
);

-- Indexes for performance
CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_social_name ON profiles(social_name) WHERE social_name IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles(email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_profiles_created_at ON profiles(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_profiles_created_date ON profiles(created_date DESC);
CREATE INDEX IF NOT EXISTS idx_profiles_full_name ON profiles USING GIN(full_name gin_trgm_ops);

-- Note: For full-text search on full_name, the pg_trgm extension must be enabled:
-- CREATE EXTENSION IF NOT EXISTS pg_trgm;

