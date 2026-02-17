/* ROLLBACK 001 : Suppression sécurisée de la structure Nexora.
   On désactive les triggers et supprime les tables dans l'ordre inverse des dépendances.
*/

-- 1. Suppression des triggers (si tu as ajouté les automatisations précédemment)
DROP TRIGGER IF EXISTS update_users_modtime ON users;
DROP TRIGGER IF EXISTS update_nas_modtime ON nas;
DROP TRIGGER IF EXISTS update_tenants_modtime ON tenants;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- 2. Suppression des tables (Ordre : Enfants -> Parents)
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS nas;
DROP TABLE IF EXISTS tenants;

-- 3. Nettoyage des extensions (Optionnel, à garder si d'autres modules l'utilisent)
-- DROP EXTENSION IF EXISTS "uuid-ossp";