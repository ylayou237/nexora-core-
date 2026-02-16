package web

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

func AuthMiddleware(authService *services.AuthService, cache *services.CacheService, secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Extraire le header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{"error": "Missing or invalid token"})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 2. Parser et Valider la signature du JWT
		claims := &domain.UserClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}

		// 3. VÉRIFICATION CRITIQUE : Check Blacklist dans Redis
		// C'est ici qu'on teste la charge de Redis !
		isBlacklisted, _ := cache.IsTokenBlacklisted(c.Context(), claims.ID)
		if isBlacklisted {
			return c.Status(401).JSON(fiber.Map{"error": "Token revoked"})
		}

		// 4. Injecter les infos dans le contexte pour les handlers suivants
		c.Locals("user", claims)
		return c.Next()
	}
}
