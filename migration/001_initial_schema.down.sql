/* ROLLBACK 001 : Suppression complète de la structure.
   L'ordre est important à cause des clés étrangères (Foreign Keys).
*/

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS nas; -- Ajout de la suppression de la table nas
DROP TABLE IF EXISTS tenants;