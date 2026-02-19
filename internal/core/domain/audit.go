package domain

import (
	"errors"
	"time"
)

//
// ======================================================
// ENUM TYPES (Strongly typed pour éviter les typos)
// ======================================================
//

type ActorType string
type ActionType string

const (
	// Actor types
	ActorSystem ActorType = "system"
	ActorUser   ActorType = "user"
	ActorAdmin  ActorType = "admin"

	// Actions - Auth
	ActionLoginSuccess ActionType = "LOGIN_SUCCESS"
	ActionLoginFailed  ActionType = "LOGIN_FAILED"
	ActionLogout       ActionType = "LOGOUT"

	// Actions - User Management
	ActionCreateUser ActionType = "CREATE_USER"
	ActionUpdateUser ActionType = "UPDATE_USER"
	ActionDeleteUser ActionType = "DELETE_USER"

	// Actions - Radius
	ActionRadiusAuthSuccess ActionType = "RADIUS_AUTH_SUCCESS"
	ActionRadiusAuthFailed  ActionType = "RADIUS_AUTH_FAILED"

	// Fallback
	ActionUnknown ActionType = "UNKNOWN"
)

//
// ======================================================
// ENTITY : AuditLog
// ======================================================
//

type AuditLog struct {
	ID        string                 `json:"id"`
	TenantID  TenantID               `json:"tenant_id"`
	UserID    *UserID                `json:"user_id,omitempty"` // Peut être nil
	ActorType ActorType              `json:"actor_type"`
	Action    ActionType             `json:"action"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

//
// ======================================================
// VALIDATION MÉTIER
// ======================================================
//

func (a *AuditLog) Validate() error {
	if a.TenantID.IsZero() {
		return errors.New("tenant_id is required")
	}

	if a.ActorType == "" {
		return errors.New("actor_type is required")
	}

	if a.Action == "" {
		return errors.New("action is required")
	}

	return nil
}

//
// ======================================================
// FILTER STRUCT (Recherche & Pagination)
// ======================================================
//

type AuditFilter struct {
	TenantID  TenantID
	UserID    *UserID
	Action    ActionType
	IPAddress string
	StartDate time.Time
	EndDate   time.Time
	Limit     int
	Offset    int
}

//
// ======================================================
// NORMALISATION FILTRE (Sécurité API)
// ======================================================
//

func (f *AuditFilter) Normalize() {
	// Sécurité pagination
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}

	if f.Offset < 0 {
		f.Offset = 0
	}
}
