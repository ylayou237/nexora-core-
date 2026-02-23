package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

type AuditRepository struct {
	pool *Adapter
}

func NewAuditRepository(pool *Adapter) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// -------------------------------------------------------------------------
// 1. ÉCRITURE (WRITE) - Support TraceID & ActorID
// -------------------------------------------------------------------------

func (r *AuditRepository) LogEvent(ctx context.Context, entry *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			trace_id, tenant_id, actor_id, actor_type, 
			action, status, ip_address, user_agent, 
			device_id, metadata, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	// 🛡️ FIX : Gestion des UUIDs optionnels (NULL si vide)
	var traceID, actorID interface{}

	if entry.TraceID != "" {
		traceID = entry.TraceID
	} else {
		traceID = nil
	}
	if entry.ActorID != "" {
		actorID = entry.ActorID
	} else {
		actorID = nil
	}

	// Le TenantID est souvent obligatoire, mais on le sécurise aussi
	var tenantID interface{}
	if !entry.TenantID.IsZero() {
		tenantID = entry.TenantID.String()
	} else {
		tenantID = nil
	}

	metaJSON, _ := json.Marshal(entry.Metadata)
	if metaJSON == nil {
		metaJSON = []byte("{}")
	}

	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := r.pool.Pool.Exec(ctx, query,
		traceID,  // $1 (NULL si "")
		tenantID, // $2 (NULL si vide)
		actorID,  // $3 (NULL si "")
		entry.ActorType,
		entry.Action,
		entry.Status,
		entry.IPAddress,
		entry.UserAgent,
		entry.DeviceID,
		metaJSON,
		createdAt,
	)

	if err != nil {
		log.Printf("❌ [AuditRepo] Erreur insertion SQL: %v (Trace: %s)", err, entry.TraceID)
		return err
	}

	return nil
}

// -------------------------------------------------------------------------
// 2. LECTURE (READ) - Mapping vers le nouveau domaine
// -------------------------------------------------------------------------

func (r *AuditRepository) FindLogs(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditLog, error) {
	// Construction dynamique Carrier-Grade
	query := `
		SELECT id, trace_id, tenant_id, actor_id, actor_type, 
		       action, status, ip_address, user_agent, device_id, 
		       metadata, created_at
		FROM audit_logs
		WHERE tenant_id = $1
	`
	args := []interface{}{filter.TenantID.String()}
	argCount := 2

	// Filtres dynamiques
	if filter.ActorID != "" {
		query += fmt.Sprintf(" AND actor_id = $%d", argCount)
		args = append(args, filter.ActorID)
		argCount++
	}
	if filter.TraceID != "" {
		query += fmt.Sprintf(" AND trace_id = $%d", argCount)
		args = append(args, filter.TraceID)
		argCount++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var rawMeta []byte
		var tenantStr string

		err := rows.Scan(
			&l.ID, &l.TraceID, &tenantStr, &l.ActorID, &l.ActorType,
			&l.Action, &l.Status, &l.IPAddress, &l.UserAgent, &l.DeviceID,
			&rawMeta, &l.CreatedAt,
		)
		if err != nil {
			continue
		}

		// Conversion TenantID
		l.TenantID, _ = domain.ParseTenantID(tenantStr)

		// Unmarshal Metadata
		_ = json.Unmarshal(rawMeta, &l.Metadata)

		logs = append(logs, l)
	}

	return logs, nil
}

// Health vérifie la santé de la connexion Postgres pour l'audit.
func (r *AuditRepository) Health(ctx context.Context) error {
	if r.pool == nil || r.pool.Pool == nil {
		return fmt.Errorf("postgres audit pool not initialized")
	}
	return r.pool.Pool.Ping(ctx)
}
