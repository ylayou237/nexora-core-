package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

type AuditRepository struct {
	pool *Adapter
}

// NewAuditRepository crée une instance du repository
func NewAuditRepository(pool *Adapter) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// -------------------------------------------------------------------------
// 1. ÉCRITURE (WRITE) - Optimisé pour l'insertion massive
// -------------------------------------------------------------------------

func (r *AuditRepository) LogEvent(ctx context.Context, logEntry *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			tenant_id, user_id, actor_type, action, metadata, ip_address, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Sécurisation du JSON (évite le crash si metadata est nil)
	metaJSON, err := json.Marshal(logEntry.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
		log.Printf("⚠️ [AuditRepo] Erreur Marshal JSON: %v", err)
	}

	// Gestion du UserID (Nullable)
	// On utilise un *string pour que le driver pgx comprenne le NULL SQL
	var uid *string
	if logEntry.UserID != nil {
		s := logEntry.UserID.String()
		uid = &s
	}

	// Gestion de la Date
	createdAt := logEntry.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	// Exécution
	_, err = r.pool.Pool.Exec(ctx, query,
		logEntry.TenantID.String(), // $1
		uid,                        // $2
		logEntry.ActorType,         // $3
		logEntry.Action,            // $4
		metaJSON,                   // $5
		logEntry.IPAddress,         // $6
		createdAt,                  // $7
	)

	if err != nil {
		// Log l'erreur mais ne pas faire paniquer l'appli pour un log manqué
		log.Printf("❌ [AuditRepo] Erreur critique insertion: %v", err)
		return err
	}

	return nil
}

// -------------------------------------------------------------------------
// 2. LECTURE (READ) - Recherche Dynamique avec Filtres
// -------------------------------------------------------------------------

// buildFindLogsQuery extrait la logique de construction SQL pour réduire la complexité cognitive
func buildFindLogsQuery(filter domain.AuditFilter) (string, []interface{}) {
	var queryBuilder strings.Builder
	var args []interface{}

	queryBuilder.WriteString(`
		SELECT id, tenant_id, user_id, actor_type, action, metadata, ip_address, created_at
		FROM audit_logs
		WHERE tenant_id = $1
	`)
	args = append(args, filter.TenantID.String())
	argCounter := 2

	if filter.UserID != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND user_id = $%d", argCounter))
		args = append(args, filter.UserID.String())
		argCounter++
	}
	if filter.Action != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND action = $%d", argCounter))
		args = append(args, filter.Action)
		argCounter++
	}
	if filter.IPAddress != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND ip_address = $%d", argCounter))
		args = append(args, filter.IPAddress)
		argCounter++
	}
	if !filter.StartDate.IsZero() {
		queryBuilder.WriteString(fmt.Sprintf(" AND created_at >= $%d", argCounter))
		args = append(args, filter.StartDate)
		argCounter++
	}
	if !filter.EndDate.IsZero() {
		queryBuilder.WriteString(fmt.Sprintf(" AND created_at <= $%d", argCounter))
		args = append(args, filter.EndDate)
		argCounter++
	}

	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCounter, argCounter+1))
	args = append(args, filter.Limit, filter.Offset)

	return queryBuilder.String(), args
}

// --- Structures et Helpers pour réduire la complexité cognitive ---

// rawAuditLog représente la donnée brute telle qu'elle sort de PostgreSQL
type rawAuditLog struct {
	ID        string
	TenantID  string
	UserID    *string
	ActorType string
	Action    string
	Metadata  []byte
	IPAddress string
	CreatedAt time.Time
}

// mapToDomainAuditLog convertit les données SQL brutes en objet métier validé
// mapToDomainAuditLog convertit les données SQL brutes en objet métier validé
func mapToDomainAuditLog(raw rawAuditLog) (domain.AuditLog, error) {
	l := domain.AuditLog{
		ID: raw.ID,
		// 👈 Ajout de la conversion explicite vers tes types de domaine
		ActorType: domain.ActorType(raw.ActorType),
		Action:    domain.ActionType(raw.Action),
		IPAddress: raw.IPAddress,
		CreatedAt: raw.CreatedAt,
	}

	// 1. Validation du TenantID
	tID, err := domain.ParseTenantID(raw.TenantID)
	if err != nil {
		return l, fmt.Errorf("TenantID invalide (%s)", raw.TenantID)
	}
	l.TenantID = tID

	// 2. Validation du UserID
	if raw.UserID != nil {
		uid, err := domain.ParseUserID(*raw.UserID)
		if err == nil {
			l.UserID = &uid
		}
	}

	// 3. Décodage des métadonnées JSONB
	if len(raw.Metadata) > 0 {
		if err := json.Unmarshal(raw.Metadata, &l.Metadata); err != nil {
			l.Metadata = make(map[string]interface{}) // Fallback vide en cas d'erreur JSON
		}
	}

	return l, nil
}

// -------------------------------------------------------------------------

func (r *AuditRepository) FindLogs(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditLog, error) {
	query, args := buildFindLogsQuery(filter)

	rows, err := r.pool.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.AuditLog

	// La boucle est maintenant extrêmement propre et lisible
	for rows.Next() {
		var raw rawAuditLog

		// Scan défensif
		err := rows.Scan(
			&raw.ID, &raw.TenantID, &raw.UserID, &raw.ActorType, &raw.Action, &raw.Metadata, &raw.IPAddress, &raw.CreatedAt,
		)
		if err != nil {
			log.Printf("⚠️ [AuditRepo] Erreur Scan ligne: %v", err)
			continue // On saute la ligne corrompue
		}

		// Mapping (Délégation de la complexité au helper)
		l, err := mapToDomainAuditLog(raw)
		if err != nil {
			log.Printf("⚠️ [AuditRepo] Data Corruption: %v", err)
			continue
		}

		logs = append(logs, l)
	}

	// Vérification finale des erreurs de parcours
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}
