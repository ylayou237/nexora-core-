package main

import (
	"context"
	"log"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover" // <--- INDISPENSABLE POUR LA PROD
	"github.com/yvan/nexora-core/internal/adapters/primary/web"
	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
	"golang.org/x/crypto/bcrypt"
)

// --- MOCK REPOSITORY (Simulation Postgres) ---

type mockUserRepo struct {
	fixedHash string
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, t domain.TenantID, u domain.Username) (*domain.User, error) {
	// Simulation d'un utilisateur trouvé en base de données
	id, _ := domain.NewUserID("550e8400-e29b-41d4-a716-446655440001")
	email, _ := domain.NewEmail("yvan@nexora.com")
	passHash, _ := domain.NewPasswordHash(m.fixedHash) // Le hash Bcrypt pré-calculé
	now := time.Now()

	// On utilise RehydrateUser pour reconstruire l'objet depuis les "données brutes"
	usr, err := domain.RehydrateUser(
		id, u, email, passHash, nil,
		domain.RoleCustomer, t, true, nil,
		0, 0, 1, now, now,
	)
	return usr, err
}

// Méthodes non utilisées pour ce test
func (m *mockUserRepo) GetByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (m *mockUserRepo) Delete(ctx context.Context, id domain.UserID) error  { return nil }

// --- MAIN APPLICATION ---

func main() {
	ctx := context.Background()

	// =========================================================================
	// 1. INFRASTRUCTURE & ADAPTERS
	// =========================================================================

	// A. Démarrage de Miniredis (Simulateur Redis en mémoire pour le dev/test)
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatalf("❌ Erreur Miniredis: %v", err)
	}
	log.Printf("🔹 Miniredis démarré sur %s", mr.Addr())

	// B. Initialisation de l'Adapter Redis (Connexion physique)
	redisAdapter, err := redis.NewAdapter(ctx, "", []string{mr.Addr()}, "")
	if err != nil {
		log.Fatalf("❌ Erreur Redis Adapter: %v", err)
	}
	defer redisAdapter.Close()

	// C. Initialisation des Repositories (Logique de stockage)
	// On injecte l'adapter physique dans le repository logique
	sessionRepo := redis.NewSessionRepository(redisAdapter)

	// =========================================================================
	// 2. CORE DOMAIN & SERVICES
	// =========================================================================

	// Préparation du UserRepo Mocké avec un hash valide pour "password123"
	h, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	userRepo := &mockUserRepo{fixedHash: string(h)}

	clock := domain.NewRealClock()
	jwtSecret := "nexora-super-secret-key-32-chars-min"

	// Initialisation du Service d'Authentification (Le cerveau)
	authService := services.NewAuthService(userRepo, sessionRepo, clock, jwtSecret)

	// =========================================================================
	// 3. HTTP SERVER (FIBER)
	// =========================================================================

	app := fiber.New(fiber.Config{
		AppName:       "Nexora Core API",
		CaseSensitive: true,
		StrictRouting: true,
	})

	// --- GLOBAL MIDDLEWARES ---
	app.Use(logger.New())  // Logs des requêtes
	app.Use(recover.New()) // 🛡️ Protection anti-crash (Capture les panics)

	// --- ROUTES ---
	v1 := app.Group("/v1")

	// -> Health Check (Public)
	v1.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "timestamp": time.Now()})
	})

	// -> Login (Public - Heavy CPU Load)
	v1.Post("/login", func(c *fiber.Ctx) error {
		// DTO local pour parser la requête JSON
		type LoginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
			TenantID string `json:"tenant_id"`
		}

		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON format"})
		}

		// Appel au service métier
		pair, err := authService.Login(c.Context(), domain.TenantID(req.TenantID), req.Username, req.Password)
		if err != nil {
			// En prod, on ne retourne pas l'erreur brute pour ne pas aider l'attaquant
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
		}

		return c.Status(fiber.StatusOK).JSON(pair)
	})

	// -> Routes Protégées (User Zone - Fast IO Load)
	// On applique le middleware AuthRequired défini dans le package "web"
	// Il utilise sessionRepo pour vérifier dans Redis si le token est encore valide.
	userRoutes := v1.Group("/user", web.AuthMiddleware(sessionRepo, jwtSecret))

	userRoutes.Get("/me", func(c *fiber.Ctx) error {
		// Récupération des données injectées par le middleware
		userID := c.Locals("user_id")
		tenantID := c.Locals("tenant_id")

		return c.JSON(fiber.Map{
			"message":   "Accès autorisé à la zone sécurisée",
			"user_id":   userID,
			"tenant_id": tenantID,
			"data":      "Voici vos données confidentielles...",
		})
	})

	// =========================================================================
	// 4. START
	// =========================================================================

	log.Println("🚀 Nexora Core est prêt à recevoir du trafic sur le port :8080")
	if err := app.Listen(":8080"); err != nil {
		log.Fatalf("❌ Erreur lors du démarrage du serveur: %v", err)
	}
}
