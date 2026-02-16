/* MIGRATION 001 : Structure Fondamentale Nexora (Identité & Infrastructure)
   Ce fichier définit les bases du Multi-Tenant et les équipements réseau (NAS).
*/

-- Activation de l'extension pour les identifiants UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. TABLE TENANTS : Hiérarchie commerciale et isolation
CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    parent_id UUID REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- operator, provider, reseller
    portal_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. TABLE NAS : Équipements réseau (MikroTik, Ubiquiti, etc.)
-- Chaque NAS appartient à un Tenant et possède un secret partagé pour RADIUS.
CREATE TABLE nas (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL,            -- Type INET pour validation IP native Postgres
    short_name VARCHAR(64),              -- Utilisé dans les logs RADIUS
    type VARCHAR(50) DEFAULT 'other',    -- ex: mikrotik, ubiquiti, cisco
    secret VARCHAR(255) NOT NULL,        -- Secret partagé RADIUS
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Protection : Une IP ne peut pas être dupliquée pour un même NAS
    CONSTRAINT uq_nas_ip UNIQUE (ip_address)
);

-- 3. TABLE USERS : Comptes clients et administrateurs
CREATE TABLE users (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username VARCHAR(64) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_users_tenant_username UNIQUE (tenant_id, username)
);

-- 4. TABLE AUDIT_LOGS : Traçabilité immuable
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    actor_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    changes JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- INDEXATION : Optimisation des performances
-- Accélère la recherche du NAS par son IP lors d'une requête RADIUS entrante
CREATE INDEX idx_nas_ip ON nas(ip_address);
-- Accélère l'authentification des utilisateurs
CREATE INDEX idx_users_tenant_username ON users(tenant_id, username);
-- Accélère la lecture des logs d'audit par date
CREATE INDEX idx_audit_tenant_date ON audit_logs(tenant_id, created_at DESC);