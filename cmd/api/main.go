package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/yvan/nexora-core/internal/adapters/primary/web/handlers"
	"github.com/yvan/nexora-core/internal/adapters/secondary/postgres"
	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
	"github.com/yvan/nexora-core/internal/config"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

func main() {
	// 1. Chargement de la configuration
	cfg := config.Load()

	// Contexte racine pour les initialisations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("🛠️  Initialisation du système Nexora...")

	// 2. Connexion à Postgres (Neon)
	dbAdapter, err := postgres.NewAdapter(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Erreur Postgres: %v", err)
	}
	// Note: On suppose que dbAdapter expose le pool via une méthode ou un champ DB
	// Si ton adapter retourne directement le pool, adapte la ligne suivante
	userRepo := postgres.NewUserRepository(dbAdapter)

	// 3. Connexion à Redis (Upstash)
	// On transforme l'adresse unique en slice pour UniversalClient
	redisAddrs := []string{cfg.RedisAddr}

	// Appel de ton nouvel Adapter avec TLS et UniversalClient
	redisAdapter, err := redis.NewAdapter(ctx, cfg.RedisPassword, redisAddrs, "")
	if err != nil {
		log.Fatalf("❌ Erreur Redis (Upstash): %v", err)
	}
	defer redisAdapter.Close()

	// On injecte le client Redis dans les repositories
	sessionRepo := redis.NewSessionRepository(redisAdapter.Client)
	protectionRepo := redis.NewAuthProtectionRepo(redisAdapter.Client)

	// 4. Initialisation des Services
	clock := domain.NewRealClock()
	authService := services.NewAuthService(
		userRepo,
		sessionRepo,
		protectionRepo,
		clock,
		cfg.JWTSecret,
		cfg.JWTAccessTTL,
		7*24*time.Hour, // Refresh Token TTL (1 semaine)
	)

	// 5. Initialisation des Handlers
	authHandler := handlers.NewAuthHandler(authService)

	// 6. Définition des Routes (Go 1.22+ syntax)
	mux := http.NewServeMux()

	// Route de Login
	mux.HandleFunc("POST /v1/auth/login", authHandler.Login)

	// Route de Santé (Healthcheck)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 7. Lancement du Serveur
	serverAddr := ":" + cfg.APIPort
	log.Printf("🚀 Nexora API démarrée sur %s (Mode: %s)", serverAddr, cfg.AppEnv)

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Erreur serveur: %v", err)
	}
}
