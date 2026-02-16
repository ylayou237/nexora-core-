package services

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
)

// AuthService coordonne l'authentification et la gestion des sessions.
type AuthService struct {
	userRepo    ports.UserRepository
	sessionRepo ports.SessionRepository
	clock       domain.Clock
	jwtSecret   []byte
}

// NewAuthService crée une nouvelle instance carrier-grade de l'AuthService.
func NewAuthService(
	uRepo ports.UserRepository,
	sRepo ports.SessionRepository,
	clock domain.Clock,
	secret string,
) *AuthService {
	return &AuthService{
		userRepo:    uRepo,
		sessionRepo: sRepo,
		clock:       clock,
		jwtSecret:   []byte(secret),
	}
}

// Login vérifie les identifiants, crée une session Redis et génère une paire de tokens JWT.
func (s *AuthService) Login(ctx context.Context, tenantID domain.TenantID, username string, password string) (*domain.TokenPair, error) {
	// 1. Récupération de l'utilisateur depuis le repository (Postgres)
	user, err := s.userRepo.GetByUsername(ctx, tenantID, domain.Username(username))
	if err != nil {
		// On retourne une erreur générique pour éviter l'énumération d'utilisateurs
		return nil, domain.ErrInvalidCredentials
	}

	// 2. Vérification sécurisée du mot de passe avec Bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash()), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// 3. Initialisation de la session (JTI unique)
	sessionID := domain.SessionID(domain.NewUUID())

	// Politique par défaut pour les sessions API (Quota illimité par défaut ici)
	policy := domain.PolicySnapshot{DataQuota: 0}

	// 4. Création de l'entité de domaine ActiveSession avec validation
	activeSession, err := domain.NewActiveSession(
		sessionID,
		user.ID(),
		"api-gateway",
		nil, // Pas d'adresse MAC requise pour les sessions API
		policy,
		1*time.Hour, // Durée de vie de la session dans Redis
		s.clock,
	)
	if err != nil {
		return nil, fmt.Errorf("auth: domain validation failed: %w", err)
	}

	// 5. Persistance de la session dans le cache (Redis)
	if err := s.sessionRepo.StartSession(ctx, activeSession); err != nil {
		return nil, fmt.Errorf("auth: session storage failed: %w", err)
	}

	// 6. Préparation des Claims pour les tokens
	claims := domain.UserClaims{
		Jti:      sessionID.String(),
		UserID:   user.ID(),
		TenantID: user.TenantID(),
		Role:     user.Role(),
		Username: user.Username().String(),
	}

	// 7. Génération des tokens Access et Refresh
	accessToken, err := s.generateToken(claims, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken(claims, 24*time.Hour*7) // Refresh valide 7 jours
	if err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    s.clock.Now().Add(15 * time.Minute),
	}, nil
}

// Logout révoque la session utilisateur en la supprimant de Redis.
func (s *AuthService) Logout(ctx context.Context, sessionID domain.SessionID) error {
	if err := s.sessionRepo.TerminateSession(ctx, sessionID); err != nil {
		return fmt.Errorf("auth: logout failed: %w", err)
	}
	return nil
}

// generateToken est une méthode helper privée pour signer les tokens JWT.
func (s *AuthService) generateToken(claims domain.UserClaims, duration time.Duration) (string, error) {
	now := s.clock.Now()

	// Mise à jour des claims temporels standards
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        claims.Jti,
		Subject:   claims.Username,
		Issuer:    "nexora-core",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		NotBefore: jwt.NewNumericDate(now),
	}

	// Création et signature du token (HS256 pour l'instant avec s.jwtSecret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("auth: failed to sign token: %w", err)
	}

	return signedToken, nil
}
