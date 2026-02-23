package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yvan/nexora-core/internal/core/domain"
)

// RedisAuditConfig définit les limites de sécurité et de rétention
type RedisAuditConfig struct {
	KeyPrefix     string
	TTL           time.Duration
	MaxPerTenant  int64 // Limite circulaire dans Redis
	FetchOnFilter int64 // Nombre de logs lus pour le filtrage en mémoire
	MaxLimit      int   // Sécurité anti-OOM pour l'API
}

func DefaultRedisAuditConfig() RedisAuditConfig {
	return RedisAuditConfig{
		KeyPrefix:     "audit:logs",
		TTL:           30 * 24 * time.Hour,
		MaxPerTenant:  10000,
		FetchOnFilter: 1000,
		MaxLimit:      1000,
	}
}

type RedisAuditRepository struct {
	client redis.UniversalClient
	cfg    RedisAuditConfig
	logger *log.Logger
}

func NewRedisAuditRepository(client redis.UniversalClient, cfg RedisAuditConfig, logger *log.Logger) *RedisAuditRepository {
	if logger == nil {
		logger = log.Default()
	}
	return &RedisAuditRepository{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// buildKey assure l'isolation Multi-Tenant et la compatibilité Redis Cluster
func (r *RedisAuditRepository) buildKey(tenantID domain.TenantID) string {
	return fmt.Sprintf("%s:{%s}", r.cfg.KeyPrefix, tenantID.String())
}

// ======================= ÉCRITURE (WRITE) =======================

func (r *RedisAuditRepository) LogEvent(ctx context.Context, entry *domain.AuditLog) error {
	if entry == nil || entry.TenantID.IsZero() {
		return fmt.Errorf("redis_audit: entrée invalide")
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("redis_audit: marshal failed: %w", err)
	}

	key := r.buildKey(entry.TenantID)

	// Pipeline Atomique pour la performance
	pipe := r.client.Pipeline()
	pipe.LPush(ctx, key, data)
	pipe.LTrim(ctx, key, 0, r.cfg.MaxPerTenant-1) // Garde la liste circulaire
	pipe.Expire(ctx, key, r.cfg.TTL)              // Auto-nettoyage mémoire

	_, err = pipe.Exec(ctx)
	return err
}

// ======================= LECTURE (READ) =======================

func (r *RedisAuditRepository) FindLogs(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditLog, error) {
	if filter.TenantID.IsZero() {
		return nil, fmt.Errorf("redis_audit: tenant_id requis")
	}

	// 1. Détermination du nombre d'éléments à lire
	limit := filter.Limit
	if limit <= 0 || limit > r.cfg.MaxLimit {
		limit = 50
	}

	// Si l'utilisateur applique des filtres, on "sur-lit" la liste Redis
	// car Redis ne sait pas filtrer nativement les LISTS.
	var fetchCount int64 = int64(limit)
	if r.hasActiveFilters(filter) {
		fetchCount = r.cfg.FetchOnFilter
	}

	key := r.buildKey(filter.TenantID)
	rawLogs, err := r.client.LRange(ctx, key, 0, fetchCount-1).Result()
	if err != nil {
		return nil, fmt.Errorf("redis_audit: lrange failed: %w", err)
	}

	// 2. Désérialisation et filtrage SIEM en mémoire
	logs := make([]domain.AuditLog, 0, limit)
	for _, raw := range rawLogs {
		var l domain.AuditLog
		if err := json.Unmarshal([]byte(raw), &l); err != nil {
			continue
		}

		if r.applyFilters(l, filter) {
			logs = append(logs, l)
		}

		if len(logs) >= limit {
			break
		}
	}

	return logs, nil
}

// hasActiveFilters vérifie si une recherche spécifique est demandée
func (r *RedisAuditRepository) hasActiveFilters(f domain.AuditFilter) bool {
	return f.Status != "" || f.Action != "" || f.ActorID != "" || f.TraceID != ""
}

// applyFilters retourne true si le log correspond aux critères du SIEM
func (r *RedisAuditRepository) applyFilters(l domain.AuditLog, f domain.AuditFilter) bool {
	if f.Status != "" && l.Status != f.Status {
		return false
	}
	if f.Action != "" && l.Action != f.Action {
		return false
	}
	if f.ActorID != "" && l.ActorID != f.ActorID {
		return false
	}
	if f.TraceID != "" && l.TraceID != f.TraceID {
		return false
	}
	return true
}

// Health vérifie que le cache Redis pour l'audit est bien en ligne.
func (r *RedisAuditRepository) Health(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("redis_audit: client non initialisé")
	}
	return r.client.Ping(ctx).Err()
}
