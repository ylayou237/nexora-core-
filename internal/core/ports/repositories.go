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
	Health(ctx context.Context) error
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
	// LogEvent enregistre un nouvel événement de sécurité (remplace l'ancien "Log")
	LogEvent(ctx context.Context, entry *domain.AuditLog) error

	// FindLogs recherche des événements pour le dashboard admin (Nouvelle fonctionnalité)
	FindLogs(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditLog, error)
	Health(ctx context.Context) error
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
	Health(ctx context.Context) error
}

type RefreshTokenRepository interface {
	// Stocke un nouveau token (hashé)
	Save(ctx context.Context, token *domain.RefreshToken) error

	// Récupère un token via son hash (pour vérification)
	GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)

	// Révoque un token spécifique (ex: lors d'un logout)
	Revoke(ctx context.Context, id domain.RefreshTokenID) error

	// Révoque TOUTE la famille de tokens (ex: détection de vol/replay)
	// On passe la raison pour l'audit SIEM
	RevokeFamily(ctx context.Context, familyID domain.TokenFamilyID, revokedAt time.Time, reason string) error

	// La méthode maîtresse : Effectue la rotation atomique via Lua.
	// Elle marque l'ancien comme utilisé et crée le nouveau en une seule opération Redis.
	Rotate(ctx context.Context, oldTokenID domain.RefreshTokenID, now time.Time, ip, ua, device string) (*domain.RefreshToken, error)
	Health(ctx context.Context) error
}

// --- Sécurité ---

// TokenHasher définit le contrat pour le hachage des tokens (ex: SHA-256)
type TokenHasher interface {
	Hash(token string) string
}

type AuthProtectionRepository interface {
	IsLocked(ctx context.Context, key string) (bool, time.Duration, error)
	RecordFailedAttempt(ctx context.Context, key string) (int64, error)

	// ✅ AJOUT
	ClearAttempts(ctx context.Context, key string) error
}
