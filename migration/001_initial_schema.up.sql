-- ==================================================================================
-- 1. EXTENSIONS & CONFIGURATION
-- ==================================================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; 

-- ==================================================================================
-- 2. TABLE TENANTS (Hiérarchie & Configuration)
-- ==================================================================================
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_id UUID REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('provider', 'reseller', 'customer')),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'archived')),
    
    -- Config
    portal_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    settings JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_parent ON tenants(parent_id);

-- ==================================================================================
-- 3. TABLE NAS (Network Access Server)
-- ==================================================================================
CREATE TABLE IF NOT EXISTS nas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL,
    short_name VARCHAR(64),
    type VARCHAR(50) DEFAULT 'other',
    
    -- Sécurité (Chiffrement PGP)
    secret_encrypted TEXT NOT NULL, 
    
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_nas_ip UNIQUE (ip_address)
);

CREATE INDEX IF NOT EXISTS idx_nas_tenant ON nas(tenant_id);
CREATE INDEX IF NOT EXISTS idx_nas_ip ON nas(ip_address);

-- ==================================================================================
-- 4. TABLE USERS (Authentification & Radius)
-- ==================================================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username VARCHAR(64) NOT NULL,
    email VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    mac_address VARCHAR(17),
    active BOOLEAN NOT NULL DEFAULT TRUE,

    -- Sécurité Lockout
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMPTZ,

    -- Radius Limits
    expired_at TIMESTAMPTZ,
    max_sessions INTEGER DEFAULT 1,
    data_quota BIGINT DEFAULT 0,
    used_data BIGINT DEFAULT 0,
    frammed_ip INET,
    
    -- Concurrence
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_users_tenant_username UNIQUE (tenant_id, username),
    CONSTRAINT check_email_format CHECK (email IS NULL OR email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT check_mac_format CHECK (mac_address IS NULL OR mac_address ~* '^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$')
);

CREATE INDEX IF NOT EXISTS idx_users_lookup ON users(tenant_id, username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_mac ON users(mac_address);

-- ==================================================================================
-- 5. AUDIT LOGS (Partitionnement & Index Optimisés)
-- ==================================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    actor_type VARCHAR(50) NOT NULL,
    action VARCHAR(100) NOT NULL,
    metadata JSONB DEFAULT '{}',
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

CREATE TABLE IF NOT EXISTS audit_logs_default PARTITION OF audit_logs DEFAULT;

-- --- INDEX OPTIMISÉS POUR L'API (FUSION RÉUSSIE) ---

-- 1. Index Principal (Tri Chronologique)
CREATE INDEX IF NOT EXISTS idx_audit_tenant_created 
ON audit_logs (tenant_id, created_at DESC);

-- 2. Index Recherche Utilisateur (Optimisé Partial Index)
CREATE INDEX IF NOT EXISTS idx_audit_tenant_user 
ON audit_logs (tenant_id, user_id) 
WHERE user_id IS NOT NULL;

-- 3. Index Filtre Action
CREATE INDEX IF NOT EXISTS idx_audit_tenant_action 
ON audit_logs (tenant_id, action);

-- 4. Index Sécurité (IP)
CREATE INDEX IF NOT EXISTS idx_audit_tenant_ip 
ON audit_logs (tenant_id, ip_address);

-- 5. Index JSONB (Recherche dans les métadonnées)
CREATE INDEX IF NOT EXISTS idx_audit_metadata 
ON audit_logs USING GIN (metadata);

-- ==================================================================================
-- 6. FONCTIONS D'AUTOMATISATION
-- ==================================================================================

-- A. Trigger Updated At
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- B. Création Partition Mensuelle
CREATE OR REPLACE FUNCTION create_audit_partition_next_month()
RETURNS void AS $$
DECLARE
    next_month_date date;
    partition_name text;
    start_date text;
    end_date text;
BEGIN
    next_month_date := date_trunc('month', now() + interval '1 month');
    partition_name := 'audit_logs_' || to_char(next_month_date, 'YYYY_MM');
    start_date := to_char(next_month_date, 'YYYY-MM-DD');
    end_date := to_char(next_month_date + interval '1 month', 'YYYY-MM-DD');

    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_logs FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- C. Archivage Automatique
CREATE OR REPLACE FUNCTION archive_old_audit_logs()
RETURNS void AS $$
DECLARE
    partition_record RECORD;
    cutoff_date date := NOW() - INTERVAL '7 years';
    partition_date date;
BEGIN
    FOR partition_record IN 
        SELECT child.relname AS partition_name
        FROM pg_inherits
        JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
        JOIN pg_class child ON pg_inherits.inhrelid = child.oid
        WHERE parent.relname = 'audit_logs'
    LOOP
        BEGIN
            partition_date := to_date(substring(partition_record.partition_name from 'audit_logs_(\d{4}_\d{2})'), 'YYYY_MM');
            IF partition_date < cutoff_date THEN
                EXECUTE format('DROP TABLE IF EXISTS %I', partition_record.partition_name);
            END IF;
        EXCEPTION WHEN OTHERS THEN
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- ==================================================================================
-- 7. TRIGGERS & INIT
-- ==================================================================================

CREATE TRIGGER update_tenants_modtime BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_nas_modtime BEFORE UPDATE ON nas FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_users_modtime BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Init Partition du mois courant
DO $$
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS audit_logs_%s PARTITION OF audit_logs FOR VALUES FROM (%L) TO (%L)',
        to_char(now(), 'YYYY_MM'),
        to_char(date_trunc('month', now()), 'YYYY-MM-DD'),
        to_char(date_trunc('month', now() + interval '1 month'), 'YYYY-MM-DD')
    );
END $$;