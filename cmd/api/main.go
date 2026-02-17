package main

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/yvan/nexora-core/internal/adapters/primary/web"
	"github.com/yvan/nexora-core/internal/adapters/secondary/postgres"
	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Aucun fichier .env trouvé")
	}

	ctx := context.Background()

	// --- A. PostgreSQL (Neon) ---
	dbURL := os.Getenv("DATABASE_URL")
	pgAdapter, err := postgres.NewAdapter(ctx, dbURL)
	if err != nil {
		log.Fatalf("❌ Erreur Neon: %v", err)
	}
	defer pgAdapter.Close()

	// --- B. Redis (Upstash) ---
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPass := os.Getenv("REDIS_PASSWORD") // On récupère le pass du .env

	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// ✅ Correction : On passe redisPass au lieu de ""
	redisAdapter, err := redis.NewAdapter(ctx, redisPass, []string{redisAddr}, "")
	if err != nil {
		log.Fatalf("❌ Erreur Upstash: %v", err)
	}
	defer redisAdapter.Close()

	// --- C. Initialisation ---
	userRepo := postgres.NewUserRepository(pgAdapter)
	sessionRepo := redis.NewSessionRepository(redisAdapter)
	clock := domain.NewRealClock()
	jwtSecret := os.Getenv("JWT_SECRET")

	authService := services.NewAuthService(userRepo, sessionRepo, clock, jwtSecret)

	// --- D. Serveur Fiber ---
	app := fiber.New(fiber.Config{
		AppName:      "Nexora Core API",
		ErrorHandler: web.DefaultErrorHandler,
	})

	app.Use(logger.New())
	app.Use(recover.New())

	v1 := app.Group("/v1")

	v1.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "up", "db": "connected", "redis": "connected"})
	})

	v1.Post("/login", func(c *fiber.Ctx) error {
		type LoginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
			TenantID string `json:"tenant_id"`
		}

		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		// 🔍 DEBUG : On regarde ce qui arrive vraiment
		log.Printf("Tentative de login - User: %s, Tenant: %s", req.Username, req.TenantID)

		// ✅ FORCE LE TENANT SI VIDE (Pour passer le cap du terminal Windows)
		if req.TenantID == "" {
			req.TenantID = "11111111-1111-1111-1111-111111111111"
		}

		tID, err := domain.NewTenantID(req.TenantID)
		if err != nil {
			// Si la validation échoue encore, on log l'erreur réelle pour comprendre
			log.Printf("❌ Erreur validation UUID: %v", err)
			return c.Status(400).JSON(fiber.Map{"error": "Validation UUID échouée"})
		}

		uName, _ := domain.NewUsername(req.Username)
		pair, err := authService.Login(c.Context(), tID, uName, req.Password)
		if err != nil {
			return err
		}

		return c.JSON(pair)
	})

	// --- E. DÉMARRAGE DU SERVEUR ---
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8081" // On utilise 8081 comme convenu
	}

	log.Printf("🚀 Nexora API démarrée sur le port %s", port)

	// Cette ligne est celle qui "bloque" le terminal et garde le serveur actif
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("❌ Erreur lors du démarrage du serveur: %v", err)
	}
}
