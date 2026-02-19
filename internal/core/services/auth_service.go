package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
)

// AuthService coordonne l'authentification et la gestion des sessions.
type AuthService struct {
	userRepo       ports.UserRepository
	sessionRepo    ports.SessionRepository
	protectionRepo domain.AuthProtectionRepository
	clock          domain.Clock
	jwtSecret      []byte
	issuer         string
	accessTTL      time.Duration
	refreshTTL     time.Duration
}

func NewAuthService(
	uRepo ports.UserRepository,
	sRepo ports.SessionRepository,
	protectionRepo domain.AuthProtectionRepository,
	clock domain.Clock,
	secret string,
	accessTTL time.Duration, // 👈 Ajouté
	refreshTTL time.Duration, // 👈 Ajouté
) *AuthService {
	return &AuthService{
		userRepo:       uRepo,
		sessionRepo:    sRepo,
		protectionRepo: protectionRepo,
		clock:          clock,
		jwtSecret:      []byte(secret),
		issuer:         "nexora-core",
		accessTTL:      accessTTL,
		refreshTTL:     refreshTTL,
	}
}

// Login vérifie les identifiants, gère la sécurité anti-brute force et génère les accès.
func (s *AuthService) Login(
	ctx context.Context,
	tenantID domain.TenantID,
	username domain.Username,
	password string,
) (*domain.TokenPair, error) {

	// 🛡️ 1. PROTECTION BRUTE-FORCE (Check préalable)
	// On vérifie le verrouillage avant toute opération coûteuse (DB/Bcrypt)
	locked, _, err := s.protectionRepo.IsLocked(ctx, username.String())
	if err == nil && locked {
		// Note: On retourne l'erreur brute pour la conformité des tests unitaires
		return nil, domain.ErrAccountLocked
	}

	// 🔍 2. RÉCUPÉRATION DE L'IDENTITÉ
	user, err := s.userRepo.GetByUsername(ctx, tenantID, username)
	if err != nil {
		// Sécurité: On enregistre une tentative même si l'user n'existe pas (Anti-Enumeration)
		s.protectionRepo.RecordFailedAttempt(ctx, username.String())
		return nil, domain.ErrInvalidCredentials
	}

	// ⚙️ 3. VÉRIFICATION DES INVARIANTS DOMAINE
	// Vérifie si le compte est actif ou expiré selon les règles métier
	if err := user.CanAuthenticate(s.clock); err != nil {
		return nil, err
	}

	// 🔑 4. VALIDATION CRYPTOGRAPHIQUE
	// Comparaison du hash Bcrypt (Opération CPU intensive)
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash().String()),
		[]byte(password),
	); err != nil {

		// Enregistrement de l'échec et vérification du seuil critique
		attempts, _ := s.protectionRepo.RecordFailedAttempt(ctx, username.String())

		if attempts >= 5 {
			return nil, domain.ErrAccountLocked
		}
		return nil, domain.ErrInvalidCredentials
	}

	// 🔓 5. RÉINITIALISATION SÉCURITÉ
	// Authentification réussie : on nettoie les tentatives de brute-force
	_ = s.protectionRepo.ClearAttempts(ctx, username.String())

	// 🎫 6. GESTION DE LA SESSION (Stateful side-effect)
	sessionID := domain.SessionID(domain.NewUUID())
	activeSession, err := domain.NewActiveSession(
		sessionID,
		user.ID(),
		"api-gateway", // Source de la connexion
		nil,           // Pas de MAC lock par défaut sur l'API
		domain.PolicySnapshot{DataQuota: user.DataQuota()},
		s.accessTTL,
		s.clock,
	)
	if err != nil {
		return nil, domain.Wrap(err, "failed to initialize security session")
	}

	// Persistance de la session dans le cache distribué (Redis/Upstash)
	if err := s.sessionRepo.StartSession(ctx, activeSession); err != nil {
		return nil, domain.Wrap(err, "session persistence failure")
	}

	// 💎 7. GÉNÉRATION DES ARTEFACTS DE SÉCURITÉ (JWT)
	claims := domain.UserClaims{
		Jti:      string(sessionID),
		UserID:   user.ID(),
		TenantID: user.TenantID(),
		Role:     user.Role(),
		Username: user.Username().String(),
	}

	accessToken, err := s.generateToken(claims, s.accessTTL)
	if err != nil {
		return nil, domain.Wrap(err, "access token issuance failed")
	}

	refreshToken, err := s.generateToken(claims, s.refreshTTL)
	if err != nil {
		return nil, domain.Wrap(err, "refresh token issuance failed")
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    s.clock.Now().Add(s.accessTTL),
	}, nil
}

// RefreshToken valide le refresh token et génère une nouvelle paire.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, err
	}
	fmt.Println("🔹 Vérification session Redis ID:", claims.Jti)

	exists, err := s.sessionRepo.Exists(ctx, domain.SessionID(claims.Jti))
	if err != nil || !exists {
		fmt.Println("❌ Session expirée ou révoquée")
		return nil, errors.New("session expired or revoked")
	}
	fmt.Println("✅ Session existante dans Redis")

	// Supprime l'ancienne session (rotation)
	_ = s.sessionRepo.TerminateSession(ctx, domain.SessionID(claims.Jti))

	// Récupération utilisateur depuis DB
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Nouvelle session + tokens
	newSessionID := domain.SessionID(domain.NewUUID())
	activeSession, err := domain.NewActiveSession(
		newSessionID,
		user.ID(),
		"api-gateway",
		nil,
		domain.PolicySnapshot{DataQuota: user.DataQuota()},
		s.accessTTL,
		s.clock,
	)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to create new session: %w", err)
	}

	if err := s.sessionRepo.StartSession(ctx, activeSession); err != nil {
		return nil, fmt.Errorf("auth: failed to store session: %w", err)
	}

	newClaims := domain.UserClaims{
		UserID:   user.ID(),
		TenantID: user.TenantID(),
		Role:     user.Role(),
		Username: user.Username().String(),
		Jti:      string(newSessionID),
	}

	accessToken, err := s.generateToken(newClaims, s.accessTTL)
	if err != nil {
		return nil, err
	}

	refreshTokenStr, err := s.generateToken(newClaims, s.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresAt:    s.clock.Now().Add(s.accessTTL),
	}, nil
}

// ValidateToken parse un JWT et retourne les claims.
func (s *AuthService) ValidateToken(tokenStr string) (*domain.UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &domain.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*domain.UserClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// generateToken signe un JWT.
func (s *AuthService) generateToken(claims domain.UserClaims, duration time.Duration) (string, error) {
	now := s.clock.Now()

	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        claims.Jti,
		Subject:   claims.Username,
		Issuer:    s.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		NotBefore: jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
