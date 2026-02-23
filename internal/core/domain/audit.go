package domain

import (
	"errors"
	"time"
)

// ======================= ENUM TYPES =======================

type ActorType string
type ActionType string
type AuditStatus string

const (
	ActorUser   ActorType = "user"
	ActorSystem ActorType = "system"
	ActorAdmin  ActorType = "admin"
)

const (
	// Auth Actions
	ActionLoginSuccess  ActionType = "LOGIN_SUCCESS"
	ActionLoginFailed   ActionType = "LOGIN_FAILED"
	ActionAccountLocked ActionType = "ACCOUNT_LOCKED"
	ActionLogout        ActionType = "LOGOUT"

	// MFA Actions
	ActionMFAChallengeSent ActionType = "MFA_CHALLENGE_SENT"
	ActionMFASuccess       ActionType = "MFA_SUCCESS"
	ActionMFAFailed        ActionType = "MFA_FAILED"

	// Token Actions
	ActionRefreshSuccess      ActionType = "REFRESH_SUCCESS"
	ActionRefreshFailed       ActionType = "REFRESH_FAILED"
	ActionRefreshReplay       ActionType = "REFRESH_REPLAY_DETECTED"
	ActionRefreshFamilyRevoke ActionType = "REFRESH_FAMILY_REVOKED"

	// Authorization Actions
	ActionRBACDenied  ActionType = "RBAC_ACCESS_DENIED"
	ActionRBACGranted ActionType = "RBAC_ACCESS_GRANTED"

	// User Management Actions
	ActionCreateUser ActionType = "CREATE_USER"
	ActionUpdateUser ActionType = "UPDATE_USER"
	ActionDeleteUser ActionType = "DELETE_USER"
)

const (
	AuditStatusSuccess  AuditStatus = "SUCCESS"
	AuditStatusFailure  AuditStatus = "FAILURE"
	AuditStatusWarning  AuditStatus = "WARNING"
	AuditStatusCritical AuditStatus = "CRITICAL"
)

// ======================= ENTITY: AuditLog =======================

type AuditLog struct {
	ID        string    `json:"id"`
	TraceID   string    `json:"trace_id"` // Corrélation SIEM
	TenantID  TenantID  `json:"tenant_id"`
	ActorID   string    `json:"actor_id"` // ID de l'utilisateur ou du système
	ActorType ActorType `json:"actor_type"`

	Action ActionType  `json:"action"`
	Status AuditStatus `json:"status"`

	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	DeviceID  string `json:"device_id,omitempty"`

	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// ======================= VALIDATION MÉTIER =======================

func (a *AuditLog) Validate() error {
	if a.TenantID.IsZero() {
		return errors.New("tenant_id is required")
	}
	if a.ActorID == "" {
		return errors.New("actor_id is required")
	}
	if a.Action == "" {
		return errors.New("action is required")
	}
	if a.TraceID == "" {
		return errors.New("trace_id is required for correlation")
	}
	return nil
}

// ======================= FILTER STRUCT =======================

type AuditFilter struct {
	TenantID  TenantID
	ActorID   string
	Action    ActionType
	Status    AuditStatus
	TraceID   string
	IPAddress string
	StartDate time.Time
	EndDate   time.Time
	Limit     int
	Offset    int
}

func (f *AuditFilter) Normalize() {
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
