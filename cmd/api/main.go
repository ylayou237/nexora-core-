package main

import (
	"context"
	"fmt"
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
	fmt.Println("DEBUG: Début du main()")

	// 1️⃣ Charge le fichier .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Aucun fichier .env trouvé")
	}

	ctx := context.Background()

	// --- A. PostgreSQL (Neon ou autre) ---
	dbURL := os.Getenv("DATABASE_URL")
	pgAdapter, err := postgres.NewAdapter(ctx, dbURL)
	if err != nil {
		log.Fatalf("❌ Erreur PostgreSQL: %v", err)
	}
	defer pgAdapter.Close()
	fmt.Println("✅ PostgreSQL OK")

	// --- B. Redis (Upstash) ---
	redisAddr := "diverse-iguana-58874.upstash.io:6379"
	redisPassword := os.Getenv("REDIS_PASSWORD")
	if redisPassword == "" {
		log.Fatalf("❌ REDIS_PASSWORD non défini dans le .env")
	}

	redisAdapter, err := redis.NewAdapter(ctx, redisPassword, []string{redisAddr}, "")
	// 🔹 Test Ping Redis pour debug
	pong, err := redisAdapter.Ping(ctx) // ⚠️ adapter doit avoir Ping() qui retourne string ou err
	if err != nil {
		log.Fatalf("❌ Redis KO: %v", err)
	}
	fmt.Println("✅ Redis OK:", pong)

	// --- C. Initialisation des composants ---
	userRepo := postgres.NewUserRepository(pgAdapter)
	sessionRepo := redis.NewSessionRepository(redisAdapter)
	clock := domain.NewRealClock()
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatalf("❌ JWT_SECRET non défini dans le .env")
	}

	authService := services.NewAuthService(userRepo, sessionRepo, clock, jwtSecret)

	// --- D. Serveur Fiber ---
	app := fiber.New(fiber.Config{
		AppName:      "Nexora Core API",
		ErrorHandler: web.DefaultErrorHandler,
	})

	app.Use(logger.New())
	app.Use(recover.New())

	v1 := app.Group("/v1")

	// Route Santé
	v1.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "up", "db": "connected", "redis": "connected"})
	})

	// --- Module Auth: login ---
	v1.Post("/login", func(c *fiber.Ctx) error {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			TenantID string `json:"tenant_id"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "JSON invalide"})
		}

		fmt.Printf("\n--- 🔍 DEBUG TENTATIVE LOGIN ---\n")
		fmt.Printf("📥 Reçu: User=[%s], Tenant=[%s]\n", req.Username, req.TenantID)

		tID, err := domain.NewTenantID(req.TenantID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "TenantID invalide"})
		}

		uName, err := domain.NewUsername(req.Username)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Username invalide"})
		}

		pair, err := authService.Login(c.Context(), tID, uName, req.Password)
		if err != nil {
			fmt.Printf("❌ ERREUR CRITIQUE JWT/REDIS: %v\n", err)
			return c.Status(401).JSON(fiber.Map{"error": "invalid credentials", "debug": err.Error()})
		}

		fmt.Println("💎 TOKEN GÉNÉRÉ AVEC SUCCÈS !")
		return c.JSON(pair)
	})

	// --- E. Démarrage du serveur ---
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "9000"
	}

	log.Printf("🚀 Nexora API démarrée sur le port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("❌ Erreur lors du démarrage du serveur: %v", err)
	}
}
