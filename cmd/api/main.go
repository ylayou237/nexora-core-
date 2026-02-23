package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/yvan/nexora-core/internal/adapters/primary/web/handlers"
	"github.com/yvan/nexora-core/internal/adapters/primary/web/middlewares"
	"github.com/yvan/nexora-core/internal/adapters/secondary/audit"
	"github.com/yvan/nexora-core/internal/adapters/secondary/postgres"
	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
	"github.com/yvan/nexora-core/internal/config"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

// ✅ NOUVEAU : Bloc de constantes HTTP pour un code 100% DRY (Don't Repeat Yourself)
const (
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	log.Println("🛠️  Démarrage de Nexora Core (Distributed Carrier-Grade Architecture)...")

	// --- 1. ADAPTATEURS SECONDAIRES ---
	dbAdapter, err := postgres.NewAdapter(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ CRITICAL: Erreur Postgres: %v", err)
	}

	redisAdapter, err := redis.NewAdapter(ctx, cfg.RedisPassword, []string{cfg.RedisAddr}, "")
	if err != nil {
		log.Fatalf("❌ CRITICAL: Erreur Redis: %v", err)
	}
	defer redisAdapter.Close()

	redisClient, ok := redisAdapter.Client.(*goredis.Client)
	if !ok {
		log.Fatal("❌ CRITICAL: Redis Client non compatible avec les scripts LUA")
	}

	// --- 2. REPOSITORIES ---
	userRepo := postgres.NewUserRepository(dbAdapter)
	postgresAuditRepo := postgres.NewAuditRepository(dbAdapter)
	sessionRepo := redis.NewSessionRepository(redisClient)
	protectionRepo := redis.NewAuthProtectionRepo(redisClient, redis.AuthProtectionConfig{
		MaxFailedAttempts: cfg.AuthMaxFailedAttempts,
		LockDuration:      cfg.AuthLockDuration,
		WindowDuration:    cfg.AuthWindowDuration,
	})
	refreshRepo := redis.NewRedisRefreshTokenRepo(redisClient)

	refreshHasher := redis.NewSHA256Hasher()

	redisAuditRepo := redis.NewRedisAuditRepository(redisClient, redis.DefaultRedisAuditConfig(), log.Default())
	multiAuditRepo := audit.NewMultiAuditRepository(postgresAuditRepo, redisAuditRepo, log.Default())

	// --- 3. SÉCURITÉ DISTRIBUÉE (JWKS & AES) ---
	encryptionKey := os.Getenv("JWKS_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		log.Fatal("❌ CRITICAL: JWKS_ENCRYPTION_KEY manquante dans le .env")
	}

	jwksStore, err := redis.NewRedisJWKSStore(redisClient, "nexora:jwks:state", encryptionKey)
	if err != nil {
		log.Fatalf("❌ CRITICAL: Impossible d'initialiser le Store JWKS: %v", err)
	}

	jwksService, err := services.NewDistributedJWKSService(services.DefaultJWKSConfig(), jwksStore, log.Default())
	if err != nil {
		log.Fatalf("❌ CRITICAL: Erreur Service JWKS: %v", err)
	}

	// --- 4. CORE SERVICES ---
	clock := domain.NewRealClock()

	// 🛠️ CONFIGURATION DE TEST : On réduit le TTL à 3 secondes
	testConfig := services.DefaultAuthConfig()
	testConfig.AccessTokenTTL = 1 * time.Hour
	authDeps := services.AuthDependencies{
		UserRepo:       userRepo,
		SessionRepo:    sessionRepo,
		ProtectionRepo: protectionRepo,
		RefreshRepo:    refreshRepo,
		RefreshHasher:  refreshHasher,
		AuditRepo:      multiAuditRepo,
		JwksService:    jwksService,
		RiskEngine:     &services.NoOpRiskEngine{},
		OtpService:     &services.NoOpOTPService{},
		Clock:          clock,
		Logger:         log.Default(),
		Config:         testConfig,
	}
	// ✅ INITIALISATION DU SERVICE
	authService := services.NewAuthService(authDeps)

	// --- 5. HANDLERS & ROUTING ---
	authHandler := handlers.NewAuthHandler(authService, userRepo, 10*time.Second)

	mux := http.NewServeMux()

	// 🚀 ROUTES PUBLIQUES (Standardisées OAuth2/OIDC)
	mux.HandleFunc("POST /v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /v1/auth/verify-mfa", authHandler.VerifyMFA)
	mux.HandleFunc("POST /v1/auth/refresh-token", authHandler.RefreshToken)
	mux.HandleFunc("POST /v1/auth/logout", authHandler.Logout)

	// Bilans de santé
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /ready", handleReady(authService))

	// 🛡️ ROUTES PROTÉGÉES (Carrier-Grade Security)
	protectedAuth := middlewares.RequireAuth(authService)

	// 🛡️ Définition de la barrière RBAC
	requireAdmin := middlewares.RequireRole(domain.RoleProviderAdmin, domain.RoleSuperAdmin)

	mux.Handle("GET /v1/me", protectedAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middlewares.GetAuthClaims(r.Context())
		if !ok {
			w.Header().Set(headerContentType, contentTypeJSON) // 👈 Constantes
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Claims introuvables ou contexte corrompu"}`))
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON) // 👈 Constantes

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Bienvenue dans la zone sécurisée de Nexora",
			"data": map[string]interface{}{
				"user_id":   claims.UserID,
				"username":  claims.Username,
				"role":      claims.Role,
				"tenant_id": claims.TenantID,
			},
		})
	})))

	// 🚨 La Route Piège pour le test d'intrusion RBAC
	mux.Handle("POST /v1/users", protectedAuth(requireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON) // 👈 Constantes
		w.Write([]byte(`{"status":"success","message":"Utilisateur créé (Si tu lis ça en étant 'customer', le RBAC a échoué !)"}`))
	}))))

	// --- 6. SERVER & GRACEFUL SHUTDOWN ---
	server := &http.Server{
		Addr:    ":" + cfg.APIPort,
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 Nexora API opérationnelle sur %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Erreur serveur: %v", err)
		}
	}()

	<-stop
	log.Println("\n🛑 Signal d'arrêt reçu, nettoyage en cours...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	server.Shutdown(shutdownCtx)
	jwksService.Stop()
	authService.Stop()

	log.Println("👋 Nexora Core arrêté proprement.")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerContentType, contentTypeJSON) // 👈 Constantes
	w.Write([]byte(`{"status":"up"}`))
}

func handleReady(svc *services.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.CheckIntegrity(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
