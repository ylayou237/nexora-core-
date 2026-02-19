package services

import (
	"context"
	"errors"

	"github.com/yvan/nexora-core/internal/core/domain"
)

// AuditRepository définit l'interface attendue par le service (Inversion de dépendance)
type AuditRepository interface {
	LogEvent(ctx context.Context, logEntry *domain.AuditLog) error
	FindLogs(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditLog, error)
}

// AuditService orchestre la lecture et l'écriture des journaux d'audit
type AuditService struct {
	repo AuditRepository
}

// NewAuditService crée une nouvelle instance du service
func NewAuditService(repo AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// Log enregistre un nouvel événement de manière asynchrone (Fire-and-Forget)
// Note: En production lourde, on passerait par un channel ou Kafka.
func (s *AuditService) Log(ctx context.Context, entry *domain.AuditLog) error {
	if entry.TenantID == "" {
		return errors.New("audit_service: tenantID is required to log an event")
	}
	return s.repo.LogEvent(ctx, entry)
}

// GetLogs récupère les journaux selon les filtres.
// Cette méthode garantit l'isolation multi-tenant.
func (s *AuditService) GetLogs(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditLog, error) {
	// SÉCURITÉ CRITIQUE : Interdire toute requête sans TenantID
	if filter.TenantID == "" {
		return nil, errors.New("audit_service: tenantID is missing in filter, access denied")
	}

	// Valeurs par défaut pour la pagination si non fournies
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000 // Protection contre les requêtes trop lourdes
	}

	return s.repo.FindLogs(ctx, filter)
}
