package web

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/ports"
)

// AuthMiddleware protège les routes en vérifiant le JWT ET la session Redis
// Note : On retire authService et cacheService car on a juste besoin du SessionRepository
func AuthMiddleware(sessionRepo ports.SessionRepository, secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Extraction du header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or invalid token"})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 2. Parser et Valider la signature du JWT
		claims := &domain.UserClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Vérifie toujours la méthode de signature !
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, domain.ErrInvalidCredentials
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		// 3. VÉRIFICATION CRITIQUE : La session existe-t-elle toujours ?
		// On utilise le JTI (Session ID) contenu dans le token
		exists, err := sessionRepo.Exists(c.Context(), domain.SessionID(claims.Jti))
		if err != nil || !exists {
			// Si la clé n'est pas dans Redis, c'est que l'utilisateur a Logout
			// ou que la session a expiré.
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Session revoked or expired"})
		}

		// 4. Injecter les infos dans le contexte
		// c.Locals permet de passer user_id au handler final sans refaire de parsing
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)

		return c.Next()
	}
}
