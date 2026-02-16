package ports

import (
	"context"
	"net"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

// --- Control Plane (PostgreSQL - Cold Storage) ---

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id domain.TenantID) (*domain.Tenant, error)
	Exists(ctx context.Context, id domain.TenantID) (bool, error)
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id domain.UserID) (*domain.User, error)
	// GetByUsername : Utilisé lors du login RADIUS/API.
	GetByUsername(ctx context.Context, tenantID domain.TenantID, username domain.Username) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
}

type PlanRepository interface {
	Create(ctx context.Context, plan *domain.Plan) error
	GetByID(ctx context.Context, id domain.PlanID) (*domain.Plan, error)
	ListByTenant(ctx context.Context, tenantID domain.TenantID) ([]*domain.Plan, error)
}

type NASRepository interface {
	Create(ctx context.Context, nas *domain.NAS) error
	GetByIP(ctx context.Context, ip net.IP) (*domain.NAS, error)
	Update(ctx context.Context, nas *domain.NAS) error
	Delete(ctx context.Context, id string) error
}
type CacheRepository interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	// Ajout du port pour SetNX
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
}

// --- Audit Trail (Compliance) ---

type AuditRepository interface {
	Log(ctx context.Context, entry AuditLogEntry) error
}

type AuditLogEntry struct {
	TenantID  domain.TenantID
	ActorID   string // UserID ou "system"
	Action    string // "USER_LOGIN", "PLAN_CREATED", etc.
	EntityID  string // ID de l'objet modifié
	Changes   string // Diff au format JSON
	Timestamp time.Time
	IPAddress string
}

// --- Data Plane (Redis - Hot Storage) ---

type SessionRepository interface {
	// StartSession : Correspond à l'ouverture du bail (Access-Accept)
	StartSession(ctx context.Context, s *domain.ActiveSession) error

	// GetByID : Récupération pour rehydration via SessionID
	GetByID(ctx context.Context, id domain.SessionID) (*domain.ActiveSession, error)

	// UpdateUsage : LE COEUR DU SYSTÈME (Exécute le script LUA atomique)
	UpdateUsage(ctx context.Context, id domain.SessionID, delta domain.UsageDelta) error

	// TerminateSession : Nettoyage (Accounting Stop)
	TerminateSession(ctx context.Context, id domain.SessionID) error

	// Exists : Vérification rapide de présence en cache
	Exists(ctx context.Context, id domain.SessionID) (bool, error)
}
