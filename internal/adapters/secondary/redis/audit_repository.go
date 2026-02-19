package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/yvan/nexora-core/internal/core/domain"
)

// RedisAuditRepository implémente services.AuditRepository avec Redis
type RedisAuditRepository struct {
	client redis.UniversalClient
	key    string
}

// NewAuditRepository crée un repository Redis pour les logs d’audit
func NewAuditRepository(client redis.UniversalClient) *RedisAuditRepository {
	return &RedisAuditRepository{
		client: client,
		key:    "audit_logs", // clé Redis où seront stockés les logs
	}
}

// FindLogs récupère les derniers logs d’audit avec filtrage
func (r *RedisAuditRepository) FindLogs(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditLog, error) {
	// 1. Détermination de la limite (priorité au filtre, sinon 50)
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	// 2. Récupération des données brutes depuis la liste Redis
	// On utilise bien 'results' ici
	results, err := r.client.LRange(ctx, r.key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("redis: failed to fetch audit logs: %w", err)
	}

	// 3. Déclaration de la variable 'logs' pour le retour
	logs := make([]domain.AuditLog, 0, len(results))

	// 4. Désérialisation JSON
	for _, item := range results {
		var logEntry domain.AuditLog
		if err := json.Unmarshal([]byte(item), &logEntry); err != nil {
			// On logge l'erreur de parsing mais on continue pour ne pas bloquer tout l'affichage
			fmt.Printf("⚠️ redis: failed to unmarshal audit log: %v\n", err)
			continue
		}

		// Ici, tu pourrais ajouter des filtres manuels (ex: filter.Action)
		// si tu veux affiner la recherche Redis après coup.
		logs = append(logs, logEntry)
	}

	return logs, nil
}

// LogEvent stocke un log d’audit dans Redis (Anciennement SaveLog)
// On change le nom pour correspondre à l'interface ports.AuditRepository
// LogEvent reçoit maintenant un pointeur pour matcher l'interface
func (r *RedisAuditRepository) LogEvent(ctx context.Context, logEntry *domain.AuditLog) error {
	data, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("redis: failed to marshal audit log: %w", err)
	}

	if err := r.client.LPush(ctx, r.key, data).Err(); err != nil {
		return fmt.Errorf("redis: failed to push audit log: %w", err)
	}

	_ = r.client.LTrim(ctx, r.key, 0, 9999)
	return nil
}
