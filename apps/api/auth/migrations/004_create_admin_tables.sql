-- Migration: 004_create_admin_tables.sql
-- Description: Creates admin_logs and invitations tables for admin service
-- Dependencies: Requires user_auths table (003_create_auth_tables.sql)

-- Table: admin_logs
-- Purpose: Audit trail for admin actions (create_user, delete_post, etc.)
CREATE TABLE IF NOT EXISTS admin_logs (
    id UUID PRIMARY KEY,
    admin_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id UUID,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
);

-- Indexes for admin_logs
CREATE INDEX IF NOT EXISTS idx_admin_logs_admin ON admin_logs(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_created_at ON admin_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_logs_created_date ON admin_logs(created_date DESC);
CREATE INDEX IF NOT EXISTS idx_admin_logs_target ON admin_logs(target_type, target_id) WHERE target_type IS NOT NULL AND target_id IS NOT NULL;

-- Table: invitations
-- Purpose: Track user invitations sent by admins
CREATE TABLE IF NOT EXISTS invitations (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    invited_by UUID NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    code VARCHAR(50) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
);

-- Indexes for invitations
CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_email ON invitations(email);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_code ON invitations(code);
CREATE INDEX IF NOT EXISTS idx_invitations_expires_at ON invitations(expires_at);
CREATE INDEX IF NOT EXISTS idx_invitations_invited_by ON invitations(invited_by);
CREATE INDEX IF NOT EXISTS idx_invitations_used ON invitations(used) WHERE used = FALSE;

