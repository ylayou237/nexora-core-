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
	userRepo    ports.UserRepository
	sessionRepo ports.SessionRepository
	clock       domain.Clock
	jwtSecret   []byte
	issuer      string
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

// NewAuthService crée une instance du service d'authentification.
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
		issuer:      "nexora-core",
		accessTTL:   15 * time.Minute,
		refreshTTL:  7 * 24 * time.Hour,
	}
}

// Login vérifie les identifiants, crée une session et génère la paire de tokens JWT.
func (s *AuthService) Login(
	ctx context.Context,
	tenantID domain.TenantID,
	username domain.Username,
	password string,
) (*domain.TokenPair, error) {

	fmt.Println("--- 🔍 Début Login ---")
	fmt.Printf("📥 Tentative login: User=[%s], Tenant=[%s]\n", username, tenantID)

	// 1️⃣ Récupération utilisateur depuis DB
	user, err := s.userRepo.GetByUsername(ctx, tenantID, username)
	if err != nil {
		fmt.Println("❌ Utilisateur non trouvé en DB")
		return nil, domain.ErrInvalidCredentials
	}
	fmt.Println("✅ Utilisateur trouvé en DB:", user.Username().String())

	// 2️⃣ Vérification que l'utilisateur peut s'authentifier
	if err := user.CanAuthenticate(s.clock); err != nil {
		fmt.Println("❌ Utilisateur ne peut pas s'authentifier:", err)
		return nil, err
	}

	// 3️⃣ Vérification du mot de passe via bcrypt
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash().String()),
		[]byte(password),
	); err != nil {
		fmt.Println("❌ Mot de passe incorrect")
		return nil, domain.ErrInvalidCredentials
	}

	// 4️⃣ Création de la session
	sessionID := domain.SessionID(domain.NewUUID())
	activeSession, err := domain.NewActiveSession(
		sessionID,
		user.ID(),
		"api-gateway",
		nil,
		domain.PolicySnapshot{DataQuota: user.DataQuota()},
		s.accessTTL,
		s.clock,
	)
	if err != nil {
		return nil, fmt.Errorf("auth: cannot create session: %w", err)
	}
	fmt.Println("🔹 Tentative de création session:", sessionID.String())

	// 5️⃣ Stockage de la session dans Redis (appel unique)
	if err := s.sessionRepo.StartSession(ctx, activeSession); err != nil {
		fmt.Println("❌ Échec stockage session:", err)
		return nil, fmt.Errorf("auth: cannot store session: %w", err)
	}
	fmt.Println("✅ Session stockée avec succès")

	// 6️⃣ Création des claims JWT
	claims := domain.UserClaims{
		Jti:      string(sessionID), // sessionID comme JTI
		UserID:   user.ID(),
		TenantID: user.TenantID(),
		Role:     user.Role(),
		Username: user.Username().String(),
	}

	// 7️⃣ Génération des tokens JWT
	accessToken, err := s.generateToken(claims, s.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateToken(claims, s.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate refresh token: %w", err)
	}

	// 8️⃣ Retour de la paire token + expiration
	fmt.Println("✅ Login réussi, tokens générés")
	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    s.clock.Now().Add(s.accessTTL),
	}, nil
}

// Logout révoque une session.
func (s *AuthService) Logout(ctx context.Context, sessionID domain.SessionID) error {
	if err := s.sessionRepo.TerminateSession(ctx, sessionID); err != nil {
		return fmt.Errorf("auth: logout failed: %w", err)
	}
	return nil
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
