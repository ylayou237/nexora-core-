package ports

import (
	"context"
	"net"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

// --- Control Plane Repositories (PostgreSQL - Cold Storage) ---

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id domain.TenantID) (*domain.Tenant, error)
	Exists(ctx context.Context, id domain.TenantID) (bool, error)
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id domain.UserID) (*domain.User, error)

	// GetByUsername : Critique pour l'auth. Doit être tenant-aware.
	GetByUsername(ctx context.Context, tenantID domain.TenantID, username domain.Username) (*domain.User, error)

	// Update : Doit gérer l'Optimistic Locking (version)
	Update(ctx context.Context, user *domain.User) error
}

type PlanRepository interface {
	Create(ctx context.Context, plan *domain.Plan) error
	GetByID(ctx context.Context, id domain.PlanID) (*domain.Plan, error)
	ListByTenant(ctx context.Context, tenantID domain.TenantID) ([]*domain.Plan, error)
}

type NASRepository interface {
	Create(ctx context.Context, nas *domain.NAS) error

	// GetByIP : Utilisation de net.IP pour cohérence avec le Domain
	GetByIP(ctx context.Context, ip net.IP) (*domain.NAS, error)

	Update(ctx context.Context, nas *domain.NAS) error
}

// --- Audit Trail (Compliance) ---

type AuditRepository interface {
	Log(ctx context.Context, entry AuditLogEntry) error
}

type AuditLogEntry struct {
	TenantID  domain.TenantID
	ActorID   string // Peut être un UserID ou "system"
	Action    string // Ex: "USER_LOGIN", "PLAN_CREATED"
	EntityID  string // ID de l'objet touché
	Changes   string // JSON diff
	Timestamp time.Time
	IPAddress string
}

// --- Data Plane Repositories (Redis - Hot Storage) ---

type SessionRepository interface {
	// SaveWithTTL : Sauvegarde la session active (ActiveSession)
	SaveWithTTL(ctx context.Context, session *domain.ActiveSession, ttl time.Duration) error

	// GetByID : Récupère via Acct-Session-Id (string)
	GetByID(ctx context.Context, id domain.SessionID) (*domain.ActiveSession, error)

	// Delete : Accounting Stop
	Delete(ctx context.Context, id domain.SessionID) error

	// Exists : Check rapide (Bloom filter ou Key exist)
	Exists(ctx context.Context, id domain.SessionID) (bool, error)
}
