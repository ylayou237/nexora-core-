package domain

import (
	"fmt"
	"time"
)

//
// ---- Error Codes (Stable Contract Layer) ----
//

type ErrorCode string

const (
	// Identity
	CodeInvalidUserID       ErrorCode = "INVALID_USER_ID"
	CodeInvalidTenantID     ErrorCode = "INVALID_TENANT_ID"
	CodeInvalidPlanID       ErrorCode = "INVALID_PLAN_ID"    // 👈 Ajouté
	CodeInvalidNasID        ErrorCode = "INVALID_NAS_ID"     // 👈 Ajouté
	CodeInvalidSessionID    ErrorCode = "INVALID_SESSION_ID" // 👈 Ajouté
	CodeInvalidEmail        ErrorCode = "INVALID_EMAIL"
	CodeInvalidUsername     ErrorCode = "INVALID_USERNAME"
	CodeInvalidMAC          ErrorCode = "INVALID_MAC"
	CodeInvalidPasswordHash ErrorCode = "INVALID_PASSWORD_HASH"
	CodeInvalidRole         ErrorCode = "INVALID_ROLE"
	CodeAccountLocked       ErrorCode = "ACCOUNT_LOCKED"

	// Hierarchy
	CodeInvalidTenantType ErrorCode = "INVALID_TENANT_TYPE"
	CodeInvalidHierarchy  ErrorCode = "TENANT_HIERARCHY_VIOLATION"
	CodeTenantNotFound    ErrorCode = "TENANT_NOT_FOUND"

	// Auth / Security
	CodeUserNotFound       ErrorCode = "USER_NOT_FOUND"
	CodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	CodeUserInactive       ErrorCode = "USER_INACTIVE"
	CodeUserExpired        ErrorCode = "USER_EXPIRED"
	CodeUnauthorizedTenant ErrorCode = "CROSS_TENANT_ACCESS_DENIED"

	// Resource / Quota
	CodeQuotaExceeded       ErrorCode = "QUOTA_EXCEEDED"
	CodeSessionLimitReached ErrorCode = "SESSION_LIMIT_REACHED"
	CodeOverflowDetected    ErrorCode = "USAGE_OVERFLOW"

	// Concurrency
	CodeOptimisticLockFailed ErrorCode = "OPTIMISTIC_LOCK_FAILED"
	CodeSessionExpired       ErrorCode = "SESSION_EXPIRED"
	CodeImmutablePolicy      ErrorCode = "IMMUTABLE_POLICY"

	//token
	TokenTypeAccess    TokenType = "access"
	TokenTypeRefresh   TokenType = "refresh"
	CodeInvalidOTP     ErrorCode = "AUTH_INVALID_OTP"
	CodeInvalidRefresh ErrorCode = "AUTH_INVALID_REFRESH"
	CodeReplayDetected ErrorCode = "AUTH_REPLAY_DETECTED"
	CodeInvalidToken   ErrorCode = "AUTH_INVALID_TOKEN"
)

//
// ---- Domain Error Type ----
//

type DomainError struct {
	Code    ErrorCode
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

// Is permet à errors.Is(err, ErrCode) de fonctionner même si l'erreur est wrappée
func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

//
// ---- Constructor ----
//

func NewError(code ErrorCode, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
	}
}

//
// ---- Wrapping helper ----
//

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

//
// ---- Sentinel Errors (Compatibility Layer) ----
//

var (
	ErrInvalidUserID       = NewError(CodeInvalidUserID, "invalid user id")
	ErrInvalidTenantID     = NewError(CodeInvalidTenantID, "invalid tenant id")
	ErrInvalidPlanID       = NewError(CodeInvalidPlanID, "invalid plan id")       // 👈 Ajouté
	ErrInvalidNasID        = NewError(CodeInvalidNasID, "invalid nas id")         // 👈 Ajouté
	ErrInvalidSessionID    = NewError(CodeInvalidSessionID, "invalid session id") // 👈 Ajouté
	ErrInvalidEmail        = NewError(CodeInvalidEmail, "invalid email")
	ErrInvalidUsername     = NewError(CodeInvalidUsername, "invalid username")
	ErrInvalidMAC          = NewError(CodeInvalidMAC, "invalid mac address")
	ErrInvalidPasswordHash = NewError(CodeInvalidPasswordHash, "invalid password hash")
	ErrInvalidRole         = NewError(CodeInvalidRole, "invalid role for context")

	ErrInvalidTenantType = NewError(CodeInvalidTenantType, "invalid tenant type")
	ErrInvalidHierarchy  = NewError(CodeInvalidHierarchy, "tenant hierarchy violation")
	ErrTenantNotFound    = NewError(CodeTenantNotFound, "tenant not found")

	ErrUserNotFound       = NewError(CodeUserNotFound, "user not found")
	ErrUserInactive       = NewError(CodeUserInactive, "user inactive")
	ErrUserExpired        = NewError(CodeUserExpired, "user expired")
	ErrUnauthorizedTenant = NewError(CodeUnauthorizedTenant, "cross tenant access denied")

	ErrQuotaExceeded       = NewError(CodeQuotaExceeded, "data quota exceeded")
	ErrSessionLimitReached = NewError(CodeSessionLimitReached, "max active sessions reached")
	ErrOverflowDetected    = NewError(CodeOverflowDetected, "usage counter overflow")

	ErrOptimisticLockFailed = NewError(CodeOptimisticLockFailed, "optimistic lock failed")
	ErrSessionExpired       = NewError(CodeSessionExpired, "session expired")
	ErrImmutablePolicy      = NewError(CodeImmutablePolicy, "policy is immutable")

	ErrInvalidOTP         = NewError(CodeInvalidOTP, "code OTP invalide")
	ErrInvalidRefresh     = NewError(CodeInvalidRefresh, "refresh token invalide ou expiré")
	ErrReplayDetected     = NewError(CodeReplayDetected, "rejeu de token détecté : faille de sécurité")
	ErrInvalidToken       = NewError(CodeInvalidToken, "token invalide ou malformé")
	ErrAccountLocked      = NewError(CodeAccountLocked, "compte temporairement bloqué suite à trop d'échecs ")
	ErrInvalidCredentials = NewError(CodeInvalidCredentials, "identifiants invalides")
)

// RateLimitError wrap une erreur domain (ex: ErrAccountLocked) avec un Retry-After.
type RateLimitError struct {
	Cause      error
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("rate limited (retry_after=%s)", e.RetryAfter)
	}
	return fmt.Sprintf("%v (retry_after=%s)", e.Cause, e.RetryAfter)
}

func (e *RateLimitError) Unwrap() error { return e.Cause }

// Helper
func NewRateLimitError(cause error, retryAfter time.Duration) error {
	return &RateLimitError{Cause: cause, RetryAfter: retryAfter}
}
