package ports

import (
	"context"
	"net"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

// --- Types Personnalisés & Enums ---

type AcctStatus string

const (
	AcctStart         AcctStatus = "Start"
	AcctStop          AcctStatus = "Stop"
	AcctInterimUpdate AcctStatus = "Interim-Update"
	AcctOn            AcctStatus = "Accounting-On"
	AcctOff           AcctStatus = "Accounting-Off"
)

// --- Authentication & Identity Service ---

type AuthService interface {
	// Authenticate : Vérifie credentials et retourne le User Domain complet
	Authenticate(ctx context.Context, tenantID domain.TenantID, username, password string) (*domain.User, error)

	// Register : Crée un user depuis une entrée API (Primitives -> Domain)
	Register(ctx context.Context, cmd RegisterUserCommand) (*domain.User, error)
}

// RegisterUserCommand est un DTO (Data Transfer Object) pour l'inscription
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

// RadiusAuthRequest représente une demande d'accès RADIUS
type RadiusAuthRequest struct {
	NasIP      net.IP
	Username   string
	Password   string
	MacAddress string
	SessionID  string
}

// RadiusAuthResponse représente la réponse à envoyer au NAS
type RadiusAuthResponse struct {
	Accept       bool
	RejectReason string
	// ReplyAttributes contient les attributs comme Framed-IP-Address ou Rate-Limit
	ReplyAttributes map[string]interface{}
}

// RadiusAcctRequest représente une demande de comptabilité RADIUS
type RadiusAcctRequest struct {
	NasIP          net.IP
	SessionID      string
	Username       string
	StatusType     AcctStatus // Utilisation du type sécurisé
	InputOctets    uint64
	OutputOctets   uint64
	SessionTime    uint64
	EventTimestamp time.Time
}

// --- Billing Service ---

type BillingService interface {
	// GenerateInvoice crée une facture pour un utilisateur donné sur une période
	GenerateInvoice(ctx context.Context, userID domain.UserID, start, end time.Time) error

	// ProcessSubscriptionRenewal traite les renouvellements automatiques
	ProcessSubscriptionRenewal(ctx context.Context) error
}
