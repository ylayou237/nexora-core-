package middlewares

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"

	"github.com/yvan/nexora-core/internal/core/domain"
)

// RequireRole vérifie que le rôle de l'utilisateur est dans la liste autorisée.
func RequireRole(allowedRoles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// ✅ CORRECTION : On récupère l'objet Claims complet
			claims, ok := GetAuthClaims(r.Context())
			if !ok {
				log.Println("🚨 [RBAC] Claims introuvables dans le context")
				writeForbiddenError(w, "Accès refusé : Session non authentifiée")
				return
			}

			// ✅ On extrait le rôle depuis les claims
			userRole := domain.Role(claims.Role)

			if !slices.Contains(allowedRoles, userRole) {
				log.Printf("🔒 [RBAC] Accès bloqué. TenantID=%s, UserID=%s, rôle=%s",
					claims.TenantID, claims.UserID, userRole)

				writeForbiddenError(w, "Accès refusé : Privilèges insuffisants")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeForbiddenError renvoie un JSON standardisé pour les 403
func writeForbiddenError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "error",
		"error":  msg,
	})
}
