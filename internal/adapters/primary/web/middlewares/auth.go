package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

const (
	AuthClaimsKey contextKey = "auth_claims"
)

// RequireAuth valide le JWT et enrichit le contexte pour les middlewares suivants (Audit, RBAC).
func RequireAuth(authService *services.AuthService) func(http.Handler) http.Handler {
	if authService == nil {
		panic("🔥 CRITICAL: RequireAuth middleware initialized with nil authService")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// 1. Extraction
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				renderError(w, "Authentification requise", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// 2. Validation
			claims, err := authService.ValidateAccessToken(tokenString)
			if err != nil {
				renderError(w, "Session invalide ou expirée", http.StatusUnauthorized)
				return
			}

			// 3. Injection (Le pont vers ton RBAC et ton Audit)
			ctx := r.Context()
			ctx = context.WithValue(ctx, AuthClaimsKey, claims)
			ctx = context.WithValue(ctx, TenantIDKey, string(claims.TenantID))
			ctx = context.WithValue(ctx, ActorIDKey, claims.UserID.String())

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAuthClaims est l'outil que ton package RBAC doit utiliser pour voir le rôle
func GetAuthClaims(ctx context.Context) (*domain.UserClaims, bool) {
	claims, ok := ctx.Value(AuthClaimsKey).(*domain.UserClaims)
	return claims, ok
}

func renderError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}
