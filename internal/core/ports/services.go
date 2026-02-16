package ports

import (
	"context"
	"net"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

// --- Authentication & Identity Service ---

type AuthService interface {
	// Authenticate : Vérifie credentials et retourne le User Domain complet
	Authenticate(ctx context.Context, tenantID domain.TenantID, username, password string) (*domain.User, error)

	// Register : Crée un user depuis une entrée API (Primitives -> Domain)
	Register(ctx context.Context, cmd RegisterUserCommand) (*domain.User, error)
}

// DTO pour l'inscription (Entrée API)
type RegisterUserCommand struct {
	TenantID   string // UUID String
	Username   string
	Email      string
	Password   string
	PlanID     string
	MacAddress string // Optionnel
}

// --- RADIUS Core Service ---

type RadiusService interface {
	// HandleAccessRequest : Décision Auth (Accept/Reject)
	HandleAccessRequest(ctx context.Context, req RadiusAuthRequest) (*RadiusAuthResponse, error)

	// HandleAccountingRequest : Traitement Accounting (Start/Stop/Update)
	HandleAccountingRequest(ctx context.Context, req RadiusAcctRequest) error
}

// DTOs RADIUS (Agnostiques du driver réseau)
type RadiusAuthRequest struct {
	NasIP      net.IP
	Username   string
	Password   string
	MacAddress string
	SessionID  string
}

type RadiusAuthResponse struct {
	Accept          bool
	RejectReason    string
	ReplyAttributes map[string]interface{} // Ex: Framed-IP-Address, Rate-Limit
}

type RadiusAcctRequest struct {
	NasIP          net.IP
	SessionID      string
	Username       string
	StatusType     string // "Start", "Stop", "Interim-Update"
	InputOctets    uint64
	OutputOctets   uint64
	SessionTime    uint64
	EventTimestamp time.Time
}

// --- Billing Service (Module 5) ---

type BillingService interface {
	GenerateInvoice(ctx context.Context, userID domain.UserID, start, end time.Time) error
	ProcessSubscriptionRenewal(ctx context.Context) error
}
