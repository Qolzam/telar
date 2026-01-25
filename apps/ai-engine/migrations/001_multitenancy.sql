-- Core tenant entity (the paying customer or "organization")
CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    owner_email VARCHAR(255) NOT NULL UNIQUE,
    
    -- Plan & Billing
    plan_tier VARCHAR(50) DEFAULT 'free', -- 'free', 'pro', 'enterprise'
    monthly_quota INT DEFAULT 1000,
    
    -- Encrypted settings (BYOK keys, custom thresholds)
    encrypted_settings BYTEA,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_tenants_email ON tenants(owner_email);

-- Apps/Sources (one tenant can have multiple apps)
CREATE TABLE apps (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    name VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL, -- 'telar', 'github', 'slack', 'api'
    
    -- Authentication
    api_key_hash VARCHAR(255) NOT NULL UNIQUE, -- bcrypt hash
    
    -- Webhook configuration
    webhook_url VARCHAR(512),
    webhook_secret VARCHAR(255),
    
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_apps_tenant ON apps(tenant_id);
CREATE INDEX idx_apps_api_key ON apps(api_key_hash);

-- Moderation queue (Partitioned by Tenant/App)
CREATE TABLE moderation_items (
    id UUID PRIMARY KEY,
    
    -- Partition keys
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    
    -- External reference
    external_id VARCHAR(255) NOT NULL, 
    
    -- Content storage (encrypted)
    encrypted_content BYTEA, 
    content_hash VARCHAR(64) NOT NULL,
    
    -- AI analysis results
    ai_scores JSONB NOT NULL,
    flag_reasons TEXT[],
    model_version VARCHAR(50),
    
    -- Workflow state
    status VARCHAR(50) DEFAULT 'pending', -- pending | approved | rejected
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '7 days')
);

CREATE INDEX idx_mod_items_tenant_status ON moderation_items(tenant_id, status, created_at DESC);
CREATE UNIQUE INDEX idx_mod_items_unique_external ON moderation_items(app_id, external_id);
