-- ==================================================================================
-- NETTOYAGE COMPLET (Ordre inverse de la création)
-- ==================================================================================

-- 1. SUPPRESSION DES TRIGGERS & FONCTIONS
-- On supprime d'abord les fonctions automatiques pour arrêter l'hémorragie
DROP FUNCTION IF EXISTS archive_old_audit_logs();
DROP FUNCTION IF EXISTS create_audit_partition_next_month();

-- Suppression du trigger générique de mise à jour de date
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- 2. SUPPRESSION DES TABLES (Avec CASCADE pour gérer les dépendances)

-- A. Audit Logs : CASCADE est obligatoire ici pour supprimer toutes les partitions enfants
-- (audit_logs_default, audit_logs_2026_02, etc.) d'un seul coup.
DROP TABLE IF EXISTS audit_logs CASCADE;

-- B. Users : Supprime les utilisateurs
DROP TABLE IF EXISTS users CASCADE;

-- C. NAS : Supprime les équipements
DROP TABLE IF EXISTS nas CASCADE;

-- D. Tenants : Supprime la racine. CASCADE nettoierait users et nas s'ils existaient encore.
DROP TABLE IF EXISTS tenants CASCADE;

-- 3. SUPPRESSION DES EXTENSIONS
-- Attention : Ne les supprimer que si aucune autre table de la DB ne les utilise.
-- Dans le contexte d'une migration "init", c'est correct de nettoyer.
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";

En francais