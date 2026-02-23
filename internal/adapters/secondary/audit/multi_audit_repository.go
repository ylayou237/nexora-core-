package audit

import (
	"context"
	"log"

	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/ports"
)

// MultiAuditRepository implémente l'interface ports.AuditRepository.
// Il utilise Postgres comme "Source de Vérité" et Redis comme "Cache Rapide".
type MultiAuditRepository struct {
	primary   ports.AuditRepository // Ton PostgresAuditRepository
	secondary ports.AuditRepository // Ton RedisAuditRepository
	logger    *log.Logger
}

// NewMultiAuditRepository crée le routeur de logs.
func NewMultiAuditRepository(primary ports.AuditRepository, secondary ports.AuditRepository, logger *log.Logger) *MultiAuditRepository {
	if logger == nil {
		logger = log.Default()
	}
	return &MultiAuditRepository{
		primary:   primary,
		secondary: secondary,
		logger:    logger,
	}
}

// ============================================================================
// 1. ÉCRITURE (WRITE) : Double écriture sécurisée
// ============================================================================

func (m *MultiAuditRepository) LogEvent(ctx context.Context, entry *domain.AuditLog) error {
	// 1. Écriture dans la Source de Vérité (Postgres) - OBLIGATOIRE
	err := m.primary.LogEvent(ctx, entry)
	if err != nil {
		m.logger.Printf("❌ [MultiAudit] Échec critique de l'écriture Postgres: %v", err)
		return err // On retourne l'erreur car la sauvegarde légale a échoué
	}

	// 2. Écriture dans le Cache (Redis) - BEST EFFORT
	// Si Redis est en panne, on ne bloque pas l'application, on logge juste un warning.
	errCache := m.secondary.LogEvent(ctx, entry)
	if errCache != nil {
		m.logger.Printf("⚠️ [MultiAudit] Échec de l'écriture Redis (ignoré): %v", errCache)
	}

	return nil
}

// ============================================================================
// 2. LECTURE (READ) : Fallback intelligent
// ============================================================================

func (m *MultiAuditRepository) FindLogs(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditLog, error) {
	// STRATÉGIE DE ROUTAGE :
	// Si on cherche une date précise ou une trace complexe, Redis n'aura pas tout l'historique (limité à 30 jours).
	// On envoie donc directement la requête à Postgres.
	requiresDeepSearch := !filter.StartDate.IsZero() || !filter.EndDate.IsZero() || filter.TraceID != ""

	if requiresDeepSearch {
		m.logger.Println("🔍 [MultiAudit] Recherche profonde requise -> Routage vers Postgres")
		return m.primary.FindLogs(ctx, filter)
	}

	// Pour l'affichage par défaut du Dashboard SIEM (les derniers événements) :
	// On tente de lire depuis Redis car c'est 100x plus rapide.
	logs, err := m.secondary.FindLogs(ctx, filter)

	// Si Redis répond avec succès et qu'il n'est pas vide, on renvoie les données.
	if err == nil && len(logs) > 0 {
		return logs, nil
	}

	// FALLBACK (Plan B) :
	// Si Redis est en erreur (ex: redémarrage serveur) ou complètement vide,
	// on se rabat silencieusement sur Postgres.
	if err != nil {
		m.logger.Printf("⚠️ [MultiAudit] Cache Redis indisponible, repli sur Postgres: %v", err)
	} else {
		m.logger.Println("ℹ️ [MultiAudit] Cache Redis vide, repli sur Postgres")
	}

	return m.primary.FindLogs(ctx, filter)
}

func (m *MultiAuditRepository) Health(ctx context.Context) error {
	// On considère le service d'audit "Healthy" si au moins la source
	// primaire (Postgres) est joignable.
	if m.primary == nil {
		return context.DeadlineExceeded // Ou une erreur explicite
	}
	return m.primary.Health(ctx)
}
