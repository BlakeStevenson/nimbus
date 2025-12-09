-- Create users table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_admin BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create auth_providers table to support multiple auth methods
CREATE TABLE auth_providers (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_type TEXT NOT NULL, -- 'password', 'oauth', 'saml', etc.
    provider_id TEXT, -- External provider identifier (null for password auth)
    credentials JSONB NOT NULL DEFAULT '{}', -- Encrypted credentials/password hash
    metadata JSONB NOT NULL DEFAULT '{}', -- Provider-specific metadata
    is_primary BOOLEAN NOT NULL DEFAULT false, -- Primary authentication method
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, provider_type, provider_id)
);

-- Create refresh_tokens table for JWT refresh tokens
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

-- Indexes for performance
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;

CREATE INDEX idx_auth_providers_user_id ON auth_providers(user_id);
CREATE INDEX idx_auth_providers_type ON auth_providers(provider_type);
CREATE INDEX idx_auth_providers_primary ON auth_providers(user_id, is_primary) WHERE is_primary = true;

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE NOT revoked;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auth_providers_updated_at BEFORE UPDATE ON auth_providers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
