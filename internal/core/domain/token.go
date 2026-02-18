package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	Jti      string `json:"jti"` // <- assure-toi qu’il existe
	UserID   UserID
	TenantID TenantID
	Role     Role
	Username string
	jwt.RegisteredClaims
}

// TokenPair représente le duo de jetons envoyés au client après auth.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}
