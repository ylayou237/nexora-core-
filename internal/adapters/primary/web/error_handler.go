package web

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/yvan/nexora-core/internal/core/domain"
)

// DefaultErrorHandler est le filet de sécurité de ton API.
// Il intercepte toutes les erreurs qui remontent des services et les traduit en JSON propre.
func DefaultErrorHandler(c *fiber.Ctx, err error) error {
	// Par défaut, on part sur une erreur 500 (Erreur Serveur)
	code := fiber.StatusInternalServerError
	message := "Une erreur interne inattendue est survenue"

	// --- 1. Mapping des erreurs de Domaine ---
	// Ici, on traduit les erreurs métier en codes HTTP
	if errors.Is(err, domain.ErrInvalidCredentials) {
		code = fiber.StatusUnauthorized
		message = "Identifiants de connexion invalides"
	} else if errors.Is(err, domain.ErrUserInactive) {
		code = fiber.StatusForbidden
		message = "Ce compte est actuellement désactivé"
	} else if errors.Is(err, domain.ErrUserNotFound) {
		code = fiber.StatusNotFound
		message = "Utilisateur ou Tenant introuvable"
	}

	// --- 2. Gestion des erreurs propres à Fiber ---
	// Si Fiber lui-même génère une erreur (ex: 404 route non trouvée)
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}

	// --- 3. Envoi de la réponse JSON ---
	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   message,
		"code":    code,
	})
}
