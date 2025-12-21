-- Migration: Create user_auths and verifications tables
-- This migration creates a normalized, relational schema for authentication
-- replacing the generic JSONB blob storage pattern

-- Table: user_auths
-- Stores user authentication credentials and verification status
CREATE TABLE IF NOT EXISTS user_auths (
    id UUID PRIMARY KEY,
    -- username is the email address used for login
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash BYTEA NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
);

-- Table: verifications
-- Stores email/phone verification codes and password reset tokens
CREATE TABLE IF NOT EXISTS verifications (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES user_auths(id) ON DELETE CASCADE,
    -- For password reset, user_id might be NULL if user not created yet
    -- In that case, target (email) is used to identify the user
    -- future_user_id stores the UserId during signup (before user is created)
    -- This allows CompleteSignup to use it without FK constraint violation
    future_user_id UUID,
    code VARCHAR(10) NOT NULL,
    target VARCHAR(255) NOT NULL, -- email or phone number
    target_type VARCHAR(50) NOT NULL, -- 'email', 'phone', 'password_reset'
    counter BIGINT DEFAULT 1,
    created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    remote_ip_address VARCHAR(45), -- IPv6 max length
    is_verified BOOLEAN DEFAULT FALSE,
    -- For password reset flow
    hashed_password BYTEA, -- stored hashed password for password reset
    expires_at BIGINT NOT NULL, -- Unix timestamp
    used BOOLEAN DEFAULT FALSE,
    full_name VARCHAR(255) -- stored full name from signup form
);

-- Indexes for user_auths
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auths_username ON user_auths(username);
CREATE INDEX IF NOT EXISTS idx_user_auths_role ON user_auths(role);
CREATE INDEX IF NOT EXISTS idx_user_auths_created_at ON user_auths(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_auths_created_date ON user_auths(created_date DESC);

-- Add future_user_id column if it doesn't exist (for existing databases)
-- Using IF NOT EXISTS syntax which is more reliable than DO block
ALTER TABLE verifications ADD COLUMN IF NOT EXISTS future_user_id UUID;

-- Indexes for verifications
CREATE INDEX IF NOT EXISTS idx_verifications_user_type ON verifications(user_id, target_type) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_verifications_code ON verifications(code) WHERE used = FALSE;
CREATE INDEX IF NOT EXISTS idx_verifications_target ON verifications(target, target_type) WHERE user_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_verifications_expires_at ON verifications(expires_at) WHERE used = FALSE;
CREATE INDEX IF NOT EXISTS idx_verifications_created_at ON verifications(created_date DESC);

