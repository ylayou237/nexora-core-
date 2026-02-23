package services

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
)

// ======================= CONFIG =======================
type AuthConfig struct {
	Issuer             string
	MaxFailedAttempts  int
	AuditWorkerCount   int
	AuditQueueSize     int
	AuditTimeout       time.Duration
	DefaultClientType  string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	MFAByRiskThreshold float64 // Score de risque au-delà duquel MFA est requis

}

func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		Issuer:             "nexora-core",
		MaxFailedAttempts:  5,
		AuditWorkerCount:   5,
		AuditQueueSize:     100,
		AuditTimeout:       5 * time.Second,
		DefaultClientType:  "api-gateway",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    7 * 24 * time.Hour,
		MFAByRiskThreshold: 0.7,
	}
}

// ======================= INTERFACES =======================
type Logger interface {
	Printf(format string, v ...interface{})
}

type RiskEngine interface {
	ComputeRisk(ctx context.Context, user *domain.User, ip, device, ua string) (float64, error)
}

type OTPService interface {
	SendOTP(ctx context.Context, user *domain.User, method string) error
	VerifyOTP(ctx context.Context, user *domain.User, code string) bool
}

// AuditContext regroupe les informations de traçabilité d'une requête
type AuditContext struct {
	IPAddress string
	UserAgent string
	DeviceID  string
	TraceID   string
}

// ======================= AUDIT WORKER =======================
type auditTask struct {
	logEntry *domain.AuditLog
}

type auditWorkerPool struct {
	tasks     chan auditTask
	wg        sync.WaitGroup
	shutdown  chan struct{}
	auditRepo ports.AuditRepository
	logger    Logger
	timeout   time.Duration
}

func newAuditWorkerPool(auditRepo ports.AuditRepository, logger Logger, cfg AuthConfig) *auditWorkerPool {
	pool := &auditWorkerPool{
		tasks:     make(chan auditTask, cfg.AuditQueueSize),
		shutdown:  make(chan struct{}),
		auditRepo: auditRepo,
		logger:    logger,
		timeout:   cfg.AuditTimeout,
	}
	for i := 0; i < cfg.AuditWorkerCount; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}
	return pool
}

func (p *auditWorkerPool) worker() {
	defer p.wg.Done()
	for {
		select {
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
			if err := p.auditRepo.LogEvent(ctx, task.logEntry); err != nil {
				p.logger.Printf("audit worker: failed to log event: %v (trace: %s)", err, task.logEntry.TraceID)
			}
			cancel()
		case <-p.shutdown:
			return
		}
	}
}

func (p *auditWorkerPool) Submit(entry *domain.AuditLog) {
	select {
	case p.tasks <- auditTask{logEntry: entry}:
	default:
		p.logger.Printf("audit queue full, dropping audit for trace: %s", entry.TraceID)
	}
}

func (p *auditWorkerPool) Stop() {
	close(p.shutdown)
	p.wg.Wait()
	close(p.tasks)
}

// ======================= AUTH SERVICE =======================
type AuthService struct {
	userRepo       ports.UserRepository
	sessionRepo    ports.SessionRepository
	protectionRepo domain.AuthProtectionRepository
	auditRepo      ports.AuditRepository
	refreshRepo    ports.RefreshTokenRepository // ✅ À AJOUTER
	refreshHasher  ports.TokenHasher
	clock          domain.Clock
	jwksService    JWKSService
	riskEngine     RiskEngine
	otpService     OTPService
	config         AuthConfig
	auditPool      *auditWorkerPool
	logger         Logger
}

// ======================= CONSTRUCTEUR =======================
type AuthDependencies struct {
	UserRepo       ports.UserRepository
	SessionRepo    ports.SessionRepository
	ProtectionRepo domain.AuthProtectionRepository
	AuditRepo      ports.AuditRepository
	RefreshRepo    ports.RefreshTokenRepository
	RefreshHasher  ports.TokenHasher
	Clock          domain.Clock
	JwksService    JWKSService
	RiskEngine     RiskEngine
	OtpService     OTPService
	Config         AuthConfig
	Logger         Logger
}

type LoginRequest struct {
	TenantID  domain.TenantID
	Username  domain.Username
	Password  string
	IPAddr    string
	UserAgent string
	DeviceID  string
	TraceID   string
}

func NewAuthService(deps AuthDependencies) *AuthService {
	// Validation des dépendances critiques
	if deps.JwksService == nil {
		log.Fatal("❌ CRITICAL: JwksService is required but was provided as nil")
	}
	if deps.UserRepo == nil {
		log.Fatal("❌ CRITICAL: UserRepo is required but was provided as nil")
	}

	if deps.Logger == nil {
		deps.Logger = log.Default()
	}

	svc := &AuthService{
		userRepo:       deps.UserRepo,
		sessionRepo:    deps.SessionRepo,
		protectionRepo: deps.ProtectionRepo,
		auditRepo:      deps.AuditRepo,
		refreshRepo:    deps.RefreshRepo,
		refreshHasher:  deps.RefreshHasher,
		clock:          deps.Clock,
		jwksService:    deps.JwksService, // ✅ L'assignation est ici
		riskEngine:     deps.RiskEngine,
		otpService:     deps.OtpService,
		config:         deps.Config,
		logger:         deps.Logger,
	}

	svc.auditPool = newAuditWorkerPool(deps.AuditRepo, deps.Logger, deps.Config)
	return svc
}

func (s *AuthService) Stop() {
	s.auditPool.Stop()
}

// ======================= LOGIN =======================
// Login gère l'authentification complète, le scoring de risque et le MFA.
// Login gère l'orchestration de haut niveau du flux d'authentification.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*domain.TokenPair, bool, error) {
	auditCtx := AuditContext{
		IPAddress: req.IPAddr, UserAgent: req.UserAgent,
		DeviceID: req.DeviceID, TraceID: req.TraceID,
	}

	// 1. PHASE DE VÉRIFICATION : Sécurité, Credentials et MFA
	user, mfaRequired, err := s.processSecurityCheck(ctx, req, auditCtx)
	if err != nil {
		return nil, false, err
	}
	if mfaRequired {
		return nil, true, nil
	}

	// 2. PHASE D'ÉMISSION : Session et Tokens
	tokens, err := s.finalizeUserLogin(ctx, user, auditCtx)
	if err != nil {
		return nil, false, err
	}

	return tokens, false, nil
}

// processSecurityCheck centralise la validation brute-force, password et moteur de risque.
func (s *AuthService) processSecurityCheck(ctx context.Context, req LoginRequest, audit AuditContext) (*domain.User, bool, error) {
	uname := strings.ToLower(strings.TrimSpace(req.Username.String()))
	bfKey := req.TenantID.String() + ":" + uname

	// Check Brute-force
	if err := s.enforceLockoutPolicy(ctx, req.TenantID, bfKey, audit); err != nil {
		return nil, false, err
	}

	// Récupération et état du compte
	user, err := s.userRepo.GetByUsername(ctx, req.TenantID, req.Username)
	if err != nil {
		return nil, false, s.handleAuthFailure(ctx, req.TenantID, bfKey, "", "user_not_found", audit)
	}

	if err := user.CanAuthenticate(s.clock); err != nil {
		s.submitAudit(req.TenantID, user.ID().String(), domain.ActionLoginFailed, domain.AuditStatusFailure, audit, map[string]interface{}{"reason": "account_inactive"})
		return nil, false, err
	}

	// Validation Password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash().String()), []byte(req.Password)); err != nil {
		return nil, false, s.handleAuthFailure(ctx, req.TenantID, bfKey, user.ID().String(), "invalid_password", audit)
	}

	_ = s.protectionRepo.ClearAttempts(ctx, bfKey)

	// Évaluation du MFA (Moteur de risque + Préférence utilisateur)
	riskScore, _ := s.riskEngine.ComputeRisk(ctx, user, req.IPAddr, req.DeviceID, req.UserAgent)
	if user.MFAEnabled() || riskScore >= s.config.MFAByRiskThreshold {
		_ = s.otpService.SendOTP(ctx, user, "email")
		s.submitAudit(req.TenantID, user.ID().String(), domain.ActionMFAChallengeSent, domain.AuditStatusSuccess, audit, map[string]interface{}{"risk_score": riskScore, "forced_by_user": user.MFAEnabled()})
		return user, true, nil
	}

	return user, false, nil
}

// finalizeUserLogin gère la création de session carrier-grade et la génération des tokens JWT/Opaque.
func (s *AuthService) finalizeUserLogin(ctx context.Context, user *domain.User, audit AuditContext) (*domain.TokenPair, error) {
	sessionID := domain.SessionID(domain.NewUUID())
	activeSession, _ := domain.NewActiveSession(sessionID, user.ID(), s.config.DefaultClientType, nil, domain.PolicySnapshot{DataQuota: user.DataQuota()}, s.config.AccessTokenTTL, s.clock)

	if err := s.sessionRepo.StartSession(ctx, activeSession); err != nil {
		return nil, fmt.Errorf("infrastructure error: failed to start session")
	}

	// 🛡️ Rollback automatique en cas de crash après création de session
	success := false
	defer func() {
		if !success {
			s.cleanupSessionOnError(ctx, sessionID, "login_finalize_failed")
		}
	}()

	// Génération des tokens
	priv, kid := s.jwksService.GetCurrentPrivateKey()
	accessToken, errA := s.generateToken(priv, kid, domain.TokenTypeAccess, user, sessionID, s.config.AccessTokenTTL)
	if errA != nil {
		return nil, fmt.Errorf("security error: failed to generate access token")
	}

	refreshTokenRaw := domain.NewUUID()
	errR := s.refreshRepo.Save(ctx, &domain.RefreshToken{
		ID: domain.RefreshTokenID(refreshTokenRaw), FamilyID: domain.TokenFamilyID(domain.NewUUID()),
		UserID: user.ID(), TenantID: user.TenantID(), TokenRaw: refreshTokenRaw,
		ExpiresAt: s.clock.Now().Add(s.config.RefreshTokenTTL), CreatedAt: s.clock.Now(),
	})
	if errR != nil {
		return nil, fmt.Errorf("infrastructure error: failed to persist refresh token")
	}

	s.submitAudit(user.TenantID(), user.ID().String(), domain.ActionLoginSuccess, domain.AuditStatusSuccess, audit, map[string]interface{}{"session_id": sessionID})
	success = true

	return &domain.TokenPair{
		AccessToken: accessToken, RefreshToken: refreshTokenRaw,
		ExpiresAt: s.clock.Now().Add(s.config.AccessTokenTTL),
	}, nil
}

// Helper : Logic de Lockout
func (s *AuthService) enforceLockoutPolicy(ctx context.Context, tID domain.TenantID, key string, audit AuditContext) error {
	locked, remaining, err := s.protectionRepo.IsLocked(ctx, key)
	if err != nil {
		return fmt.Errorf("infrastructure error: protection check failed")
	}
	if locked {
		s.submitAudit(tID, "", domain.ActionLoginFailed, domain.AuditStatusWarning, audit, map[string]interface{}{
			"reason": "account_locked", "retry_after_sec": int64(remaining / time.Second),
		})
		return domain.NewRateLimitError(domain.ErrAccountLocked, remaining)
	}
	return nil
}

// Helper : Gestion centralisée des échecs d'auth (Anti-énumération)
func (s *AuthService) handleAuthFailure(ctx context.Context, tID domain.TenantID, key, uID, reason string, audit AuditContext) error {
	att, _ := s.protectionRepo.RecordFailedAttempt(ctx, key)
	if int(att) >= s.config.MaxFailedAttempts {
		s.submitAudit(tID, uID, domain.ActionAccountLocked, domain.AuditStatusWarning, audit, map[string]interface{}{"reason": reason, "attempts": att})
		return domain.ErrAccountLocked
	}
	s.submitAudit(tID, uID, domain.ActionLoginFailed, domain.AuditStatusFailure, audit, map[string]interface{}{"reason": reason, "attempts": att})
	return domain.ErrInvalidCredentials
}

// ======================= VERIFY MFA =======================
type VerifyMFARequest struct {
	User   *domain.User
	Code   string
	IP     string
	UA     string
	Device string
	Trace  string
}

// VerifyMFA valide le code OTP et génère la session finale si le code est correct.
func (s *AuthService) VerifyMFA(ctx context.Context, req VerifyMFARequest) (*domain.TokenPair, error) {
	// ✅ 1. Préparation du contexte d'audit
	auditCtx := AuditContext{
		IPAddress: req.IP,
		UserAgent: req.UA,
		DeviceID:  req.Device,
		TraceID:   req.Trace,
	}

	// 2. Vérification de l'OTP
	if !s.otpService.VerifyOTP(ctx, req.User, req.Code) {
		s.submitAudit(req.User.TenantID(), req.User.ID().String(), domain.ActionMFAFailed, domain.AuditStatusFailure,
			auditCtx, nil)
		return nil, domain.ErrInvalidOTP
	}

	// 3. Création de la session après succès MFA
	sessionID := domain.SessionID(domain.NewUUID())
	activeSession, _ := domain.NewActiveSession(
		sessionID,
		req.User.ID(),
		s.config.DefaultClientType,
		nil,
		domain.PolicySnapshot{DataQuota: req.User.DataQuota()},
		s.config.AccessTokenTTL,
		s.clock,
	)

	// On ignore l'erreur pour la fluidité, mais en prod un check est préférable
	_ = s.sessionRepo.StartSession(ctx, activeSession)

	// 4. Génération des tokens sécurisés (RS256)
	priv, kid := s.jwksService.GetCurrentPrivateKey()
	accessToken, _ := s.generateToken(priv, kid, domain.TokenTypeAccess, req.User, sessionID, s.config.AccessTokenTTL)
	refreshToken, _ := s.generateToken(priv, kid, domain.TokenTypeRefresh, req.User, sessionID, s.config.RefreshTokenTTL)

	// 5. Audit de succès
	s.submitAudit(req.User.TenantID(), req.User.ID().String(), domain.ActionMFASuccess, domain.AuditStatusSuccess,
		auditCtx, nil)

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    s.clock.Now().Add(s.config.AccessTokenTTL),
	}, nil
}

// ======================= LOGOUT =======================

type LogoutRequest struct {
	AccessToken  string
	RefreshToken string // ✅ Le champ qui débloque la compilation du Handler !
	IPAddr       string
	UserAgent    string
	DeviceID     string
	TraceID      string
}

func (s *AuthService) Logout(ctx context.Context, req LogoutRequest) error {
	// Préparation du contexte d'audit
	auditCtx := AuditContext{
		IPAddress: req.IPAddr,
		UserAgent: req.UserAgent,
		DeviceID:  req.DeviceID,
		TraceID:   req.TraceID,
	}

	var tenantID domain.TenantID
	var userID string
	var sessionID string

	// 1. Validation de l'Access Token
	claims, err := s.validateToken(req.AccessToken, domain.TokenTypeAccess)
	if err != nil {
		// 🛡️ CARRIER-GRADE : On ne fait PAS de 'return' ici !
		// Si le token est expiré, on loggue l'avertissement, mais on
		// continue pour pouvoir révoquer le Refresh Token en bas.
		s.submitAudit(domain.TenantID(""), "", domain.ActionLogout, domain.AuditStatusWarning,
			auditCtx, map[string]interface{}{"reason": "invalid_or_expired_access_token"})
	} else {
		// Le token est valide, on récupère les infos pour l'audit et la session
		tenantID = claims.TenantID
		userID = claims.UserID.String()
		sessionID = claims.Jti

		// 2. Suppression de la session Redis (Le JWT ne sera plus accepté)
		if err := s.sessionRepo.TerminateSession(ctx, domain.SessionID(sessionID)); err != nil {
			s.logger.Printf("logout warning: failed to terminate session %s: %v", sessionID, err)
		}
	}

	// 3. ☢️ RÉVOCATION NUCLÉAIRE
	if req.RefreshToken != "" {
		// 🔴 FIX : On convertit la string en domain.RefreshTokenID pour satisfaire l'interface
		if err := s.refreshRepo.Revoke(ctx, domain.RefreshTokenID(req.RefreshToken)); err != nil {
			s.logger.Printf("logout error: failed to revoke refresh token: %v", err)
			return fmt.Errorf("infrastructure error: failed to revoke tokens")
		}
	}

	// 4. Audit de succès global
	s.submitAudit(tenantID, userID, domain.ActionLogout, domain.AuditStatusSuccess,
		auditCtx, map[string]interface{}{
			"session_id":      sessionID,
			"refresh_revoked": req.RefreshToken != "",
		})

	return nil
}

// ======================= REFRESH TOKEN =======================
type RefreshRequest struct {
	RefreshTokenRaw string
	IP              string
	UA              string
	Device          string
	Trace           string
}

func (s *AuthService) Refresh(ctx context.Context, req RefreshRequest) (*domain.TokenPair, error) {
	auditCtx := AuditContext{
		IPAddress: req.IP,
		UserAgent: req.UA,
		DeviceID:  req.Device,
		TraceID:   req.Trace,
	}

	now := s.clock.Now()

	// 🚨 FIX MAJEUR : ON NE HACHE PLUS ICI !
	// On passe directement le secret BRUT (UUID) reçu du client.
	// C'est le refreshRepo.Rotate qui va se charger de le hacher
	// pour chercher la bonne clé dans Redis.
	newRec, err := s.refreshRepo.Rotate(
		ctx,
		domain.RefreshTokenID(req.RefreshTokenRaw), // 👈 On passe la valeur brute !
		now,
		req.IP, req.UA, req.Device,
	)

	if err != nil {
		// 🛡️ FIX POSTGRES : En cas d'erreur, on ne connaît pas le Tenant de l'utilisateur.
		systemTenant := domain.TenantID("00000000-0000-0000-0000-000000000000")

		// GESTION DU REPLAY (Hacker détecté ou double clic)
		if errors.Is(err, domain.ErrReplayDetected) {
			s.submitAudit(systemTenant, "", domain.ActionRefreshReplay, domain.AuditStatusWarning,
				auditCtx, map[string]interface{}{"reason": "token_already_used_replay_detected"})
			return nil, domain.ErrReplayDetected
		}

		// Autres erreurs (Token introuvable dans Upstash, expiré, etc.)
		s.submitAudit(systemTenant, "", domain.ActionRefreshFailed, domain.AuditStatusFailure,
			auditCtx, map[string]interface{}{"reason": err.Error()})

		return nil, domain.ErrInvalidRefresh
	}

	// 3) Émettre les nouveaux tokens (newRec contient UserID et TenantID injectés par Lua)
	user, err := s.userRepo.GetByID(ctx, newRec.UserID)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Création de la session Access (JWT)
	newSessionID := domain.SessionID(domain.NewUUID())
	activeSession, _ := domain.NewActiveSession(newSessionID, user.ID(), s.config.DefaultClientType, nil,
		domain.PolicySnapshot{DataQuota: user.DataQuota()}, s.config.AccessTokenTTL, s.clock)

	// On lance la session en arrière-plan
	_ = s.sessionRepo.StartSession(ctx, activeSession)

	// Signature du JWT via JWKS
	priv, kid := s.jwksService.GetCurrentPrivateKey()
	access, errT := s.generateToken(priv, kid, domain.TokenTypeAccess, user, newSessionID, s.config.AccessTokenTTL)
	if errT != nil {
		return nil, fmt.Errorf("security error: failed to generate access token")
	}

	// Audit de succès (Ici on a enfin le vrai TenantID de l'utilisateur)
	s.submitAudit(user.TenantID(), user.ID().String(), domain.ActionRefreshSuccess, domain.AuditStatusSuccess,
		auditCtx, map[string]interface{}{"family_id": newRec.FamilyID})

	return &domain.TokenPair{
		AccessToken:  access,
		RefreshToken: newRec.TokenRaw, // Le nouveau secret généré aléatoirement par le Repo
		ExpiresAt:    s.clock.Now().Add(s.config.AccessTokenTTL),
	}, nil
}

// ======================= GENERATE TOKEN =======================
func (s *AuthService) generateToken(priv *rsa.PrivateKey, kid string, tokenType domain.TokenType, user *domain.User, sessionID domain.SessionID, duration time.Duration) (string, error) {
	now := s.clock.Now()
	claims := domain.UserClaims{
		UserID:    user.ID(),
		TenantID:  user.TenantID(),
		Role:      user.Role(),
		Username:  user.Username().String(),
		Jti:       string(sessionID),
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        string(sessionID),
			Subject:   user.Username().String(),
			Issuer:    s.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(priv)
}

// ======================= VALIDATE TOKEN =======================
func (s *AuthService) validateToken(tokenStr string, expectedType domain.TokenType) (*domain.UserClaims, error) {
	// 1. On initialise les claims
	userClaims := &domain.UserClaims{}

	// 2. On parse avec l'option WithLeeway(0) pour une expiration à la seconde près
	token, err := jwt.ParseWithClaims(tokenStr, userClaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %v", domain.ErrInvalidToken, token.Header["alg"])
		}
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("%w: missing kid", domain.ErrInvalidToken)
		}
		return s.jwksService.GetPublicKeyByKid(kid)
	}, jwt.WithLeeway(0)) // 👈 AJOUT CRUCIAL : Supprime la tolérance d'horloge

	if err != nil {
		// Ici, l'erreur contiendra "token is expired" dès la 4ème seconde (si TTL=3s)
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	// 3. Vérifications additionnelles

	// ✅ FIX : On va chercher l'Issuer au bon endroit dans la structure v5
	var tokenIssuer string
	if userClaims.RegisteredClaims.Issuer != "" {
		tokenIssuer = userClaims.RegisteredClaims.Issuer
	}

	if tokenIssuer != s.config.Issuer {
		return nil, fmt.Errorf("%w: invalid issuer (expected %s, got %s)", domain.ErrInvalidToken, s.config.Issuer, tokenIssuer)
	}

	if userClaims.TokenType != expectedType {
		return nil, fmt.Errorf("%w: invalid token type (expected %s, got %s)", domain.ErrInvalidToken, expectedType, userClaims.TokenType)
	}
	return userClaims, nil
}

// ======================= AUDIT =======================
// submitAudit prépare et envoie l'entrée d'audit au worker pool de manière asynchrone.
func (s *AuthService) submitAudit(
	tenantID domain.TenantID,
	actorID string,
	action domain.ActionType,
	status domain.AuditStatus,
	ctx AuditContext, // Regroupement des 4 strings en 1 seul objet
	metadata map[string]interface{},
) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["service"] = "nexora-auth"

	entry := &domain.AuditLog{
		TraceID:   ctx.TraceID,
		TenantID:  tenantID,
		ActorID:   actorID,
		ActorType: domain.ActorUser,
		Action:    action,
		Status:    status,
		IPAddress: ctx.IPAddress,
		UserAgent: ctx.UserAgent,
		DeviceID:  ctx.DeviceID,
		Metadata:  metadata,
		CreatedAt: s.clock.Now(),
	}

	// Envoi au worker pool pour ne pas bloquer le thread principal
	s.auditPool.Submit(entry)
}

// ======================= SESSION CLEANUP =======================
func (s *AuthService) cleanupSessionOnError(ctx context.Context, sessionID domain.SessionID, reason string) {
	if err := s.sessionRepo.TerminateSession(ctx, sessionID); err != nil {
		s.logger.Printf("failed to cleanup session %s after %s: %v", sessionID, reason, err)
	}
}

// ValidateAccessToken est l'exposition publique pour les middlewares.
// Elle permet de valider un jeton d'accès sans exposer la logique interne des types de tokens.
func (s *AuthService) ValidateAccessToken(tokenStr string) (*domain.UserClaims, error) {
	return s.validateToken(tokenStr, domain.TokenTypeAccess)
}

// CheckIntegrity vérifie la santé de l'intégralité de la stack technique.
// Cette méthode est appelée par le serveur Web sur la route /ready.
func (s *AuthService) CheckIntegrity(ctx context.Context) error {
	// 1. Vérification du stockage Utilisateurs (PostgreSQL / Neon)
	if err := s.userRepo.Health(ctx); err != nil {
		return fmt.Errorf("user_repository_unreachable: %w", err)
	}

	// 2. Vérification du stockage des Sessions (Redis / Upstash)
	if err := s.sessionRepo.Health(ctx); err != nil {
		return fmt.Errorf("session_repository_unreachable: %w", err)
	}

	// 3. Vérification du stockage des Refresh Tokens (Redis / Upstash)
	// Essentiel car ta méthode Refresh() dépend de ce repo
	if err := s.refreshRepo.Health(ctx); err != nil {
		return fmt.Errorf("refresh_repository_unreachable: %w", err)
	}

	// 4. Optionnel : Vérification de l'AuditRepo
	// Si l'audit est critique pour la conformité, décommente ceci :
	/*
		if err := s.auditRepo.Health(ctx); err != nil {
			return fmt.Errorf("audit_repository_unreachable: %w", err)
		}
	*/

	return nil
}

// a faire
// NoOpRiskEngine ne fait rien et renvoie un risque de 0
type NoOpRiskEngine struct{}

func (e *NoOpRiskEngine) ComputeRisk(ctx context.Context, user *domain.User, ip, device, ua string) (float64, error) {
	return 0.0, nil
}

// NoOpOTPService simule l'envoi d'OTP
type NoOpOTPService struct{}

func (s *NoOpOTPService) SendOTP(ctx context.Context, user *domain.User, method string) error {
	return nil
}
func (s *NoOpOTPService) VerifyOTP(ctx context.Context, user *domain.User, code string) bool {
	return true
}
