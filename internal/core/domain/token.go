package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

type UserClaims struct {
	Jti       string    `json:"jti"`
	UserID    UserID    `json:"user_id"`
	TenantID  TenantID  `json:"tenant_id"`
	Role      Role      `json:"role"`
	Username  string    `json:"username"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type RefreshTokenID string
type TokenFamilyID string

type RefreshToken struct {
	ID       RefreshTokenID
	FamilyID TokenFamilyID
	UserID   UserID
	TenantID TenantID

	TokenHash string
	TokenRaw  string // ✅ AJOUTE CECI : Nécessaire pour retourner le token au client après rotation
	ParentID  *RefreshTokenID

	ExpiresAt time.Time
	CreatedAt time.Time
	UsedAt    *time.Time
	RevokedAt *time.Time
}
