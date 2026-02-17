-- Extension pour les identifiants UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. TENANTS
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_id UUID REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    portal_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. NAS
CREATE TABLE IF NOT EXISTS nas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL, -- Type optimisé pour PostgreSQL
    short_name VARCHAR(64),
    type VARCHAR(50) DEFAULT 'other',
    secret VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_nas_ip UNIQUE (ip_address)
);

-- 3. USERS (Corrigé avec DEFAULT uuid)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), -- <--- AJOUT DU DEFAULT ICI
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username VARCHAR(64) NOT NULL,
    email VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    mac_address VARCHAR(17),
    active BOOLEAN NOT NULL DEFAULT FALSE,
    expired_at TIMESTAMPTZ,
    max_sessions INTEGER DEFAULT 1,
    data_quota BIGINT DEFAULT 0,
    used_data BIGINT DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_users_tenant_username UNIQUE (tenant_id, username)
);

-- 4. AUDIT_LOGS
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    actor_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    changes JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. INDEXATION
CREATE INDEX idx_nas_lookup ON nas(ip_address);
CREATE INDEX idx_users_auth ON users(tenant_id, username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_audit_lookup ON audit_logs(tenant_id, created_at DESC);
-- 6. AUTOMATISATION DU UPDATED_AT
-- Fonction réutilisable pour mettre à jour le timestamp automatiquement
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Application du trigger aux tables principales
CREATE TRIGGER update_tenants_modtime BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_nas_modtime BEFORE UPDATE ON nas FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_users_modtime BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- 7. OPTIMISATION AUDIT (Recherche dans le JSON)
-- Permet de filtrer instantanément par champ dans les logs d'audit
CREATE INDEX idx_audit_changes_gin ON audit_logs USING GIN (changes);

-- 8. CONTRAINTES DE SÉCURITÉ (Check Constraints)
ALTER TABLE users ADD CONSTRAINT check_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');
ALTER TABLE users ADD CONSTRAINT check_mac_format CHECK (mac_address IS NULL OR mac_address ~* '^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$');