package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UserClaims contient les données métier et les claims standards JWT.
type UserClaims struct {
	Jti      string   `json:"jti"` // Redondant avec RegisteredClaims.ID mais utile pour la clarté
	UserID   UserID   `json:"user_id"`
	TenantID TenantID `json:"tenant_id"`
	Role     Role     `json:"role"`
	Username string   `json:"username"`

	// RegisteredClaims inclut iat, exp, jti, sub, iss, etc.
	// Cette composition permet de satisfaire automatiquement l'interface jwt.Claims
	jwt.RegisteredClaims
}

// TokenPair représente le duo de jetons envoyés au client après auth.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}
