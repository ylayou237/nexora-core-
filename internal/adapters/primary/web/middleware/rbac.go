package middleware

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
			role, ok := r.Context().Value(RoleKey).(domain.Role)
			if !ok {
				log.Println("🚨 [RBAC] Rôle introuvable dans le context ou type incorrect")
				writeForbiddenError(w, "Accès refusé : Impossible de vérifier vos permissions")
				return
			}

			if !slices.Contains(allowedRoles, role) {
				// Ici on peut loguer uniquement l'ID utilisateur/tenant pour plus de sécurité
				userID := r.Context().Value(UserIDKey)
				tenantID := r.Context().Value(TenantIDKey)
				log.Printf("🔒 [RBAC] Accès bloqué. TenantID=%v, UserID=%v, rôle=%v", tenantID, userID, role)
				writeForbiddenError(w, "Accès refusé : Vous n'avez pas les droits nécessaires pour cette action")
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
	json.NewEncoder(w).Encode(map[string]string{
		"status": "error",
		"error":  msg,
	})
}
