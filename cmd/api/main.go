package main

import (
	"context"
	"log"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
	"golang.org/x/crypto/bcrypt"
)

// --- MOCK REPOSITORY (Temporaire en attendant Postgres) ---

type mockUserRepo struct {
	fixedHash string
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, t domain.TenantID, u domain.Username) (*domain.User, error) {
	// Création d'un utilisateur factice pour le test de login
	// Note: On utilise RehydrateUser ici car on simule une lecture depuis une DB
	id, _ := domain.NewUserID("550e8400-e29b-41d4-a716-446655440001")
	email, _ := domain.NewEmail("yvan@nexora.com")
	passHash, _ := domain.NewPasswordHash(m.fixedHash)
	now := time.Now()

	usr, err := domain.RehydrateUser(
		id, u, email, passHash, nil,
		domain.RoleCustomer, t, true, nil,
		0, 0, 1, now, now,
	)
	return usr, err
}

func (m *mockUserRepo) GetByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (m *mockUserRepo) Delete(ctx context.Context, id domain.UserID) error  { return nil }

// --- MAIN ---

func main() {
	ctx := context.Background()

	// 1. Initialisation de Redis (Miniredis pour le test, ou NewAdapter pour le vrai Redis)
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatalf("Erreur Miniredis: %v", err)
	}

	// L'Adapter gère la connexion physique
	redisAdapter, err := redis.NewAdapter(ctx, "", []string{mr.Addr()}, "")
	if err != nil {
		log.Fatalf("Erreur Redis Adapter: %v", err)
	}
	defer redisAdapter.Close()

	// 2. Initialisation des Repositories (Les "Ponts" vers les Ports)
	// On ne passe plus l'Adapter directement aux services, mais ces Repositories
	sessionRepo := redis.NewSessionRepository(redisAdapter)
	cacheRepo := redis.NewCacheRepository(redisAdapter)

	// 3. Préparation des dépendances de l'AuthService
	h, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	userRepo := &mockUserRepo{fixedHash: string(h)}
	clock := domain.NewRealClock()
	jwtSecret := "nexora-super-secret-key-32-chars-min"

	// 4. Initialisation des Services métier
	authService := services.NewAuthService(userRepo, sessionRepo, clock, jwtSecret)
	cacheService := services.NewCacheService(cacheRepo)

	// On "utilise" cacheService pour éviter l'erreur "declared and not used"
	_ = cacheService

	// 5. Configuration de Fiber
	app := fiber.New(fiber.Config{
		AppName: "Nexora API v1",
	})
	app.Use(logger.New())

	// 6. Routes
	v1 := app.Group("/v1")

	// Route de Santé
	v1.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "nexora-core"})
	})

	// Route de Login (Test direct)
	v1.Post("/login", func(c *fiber.Ctx) error {
		type LoginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
			TenantID string `json:"tenant_id"`
		}

		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		// Appel au service d'authentification
		pair, err := authService.Login(c.Context(), domain.TenantID(req.TenantID), req.Username, req.Password)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
		}

		return c.JSON(pair)
	})

	// 7. Lancement du serveur
	log.Printf("Nexora Core démarré sur :8080")
	log.Fatal(app.Listen(":8080"))
}
